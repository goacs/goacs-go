package http

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
	"goacs/lib"
	"goacs/models/log"
	stdlog "log"
	"net/http"
	"strings"
)

var socketServer *socketio.Server

// deviceRoom mirrors goacs-php's Pusher channel naming ("device.{id}"), just
// keyed by CPE UUID instead of an autoincrement id.
func deviceRoom(uuid string) string {
	return "device." + uuid
}

type joinDevicePayload struct {
	Token string `json:"token"`
	UUID  string `json:"uuid"`
}

func NewSocketIO(router *gin.Engine) {
	checkOrigin := allowedOriginChecker()

	var err error
	socketServer, err = socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{CheckOrigin: checkOrigin},
			&websocket.Transport{CheckOrigin: checkOrigin},
		},
	})

	if err != nil {
		panic(fmt.Sprint("Cannot create socketio ", err.Error()))
	}

	router.GET("/socket.io/*any", gin.WrapH(socketServer))
	router.POST("/socket.io/*any", gin.WrapH(socketServer))

	socketServer.OnConnect("/", onConnect)
	socketServer.OnDisconnect("/", OnDisconnect)

	socketServer.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	// go-socket.io has no built-in per-channel auth like Pusher's private
	// channels, so the client authenticates the join itself by sending its
	// JWT alongside the device uuid it wants live logs for.
	socketServer.OnEvent("/", "join-device", func(s socketio.Conn, payload joinDevicePayload) {
		if !validSocketToken(payload.Token) {
			stdlog.Println("join-device rejected: invalid token")
			return
		}
		s.Join(deviceRoom(payload.UUID))
	})

	socketServer.OnEvent("/", "leave-device", func(s socketio.Conn, payload joinDevicePayload) {
		s.Leave(deviceRoom(payload.UUID))
	})
}

func GetSocketServer() *socketio.Server {
	if socketServer == nil {
		panic("SocketServer nil")
	}

	return socketServer
}

// EmitDeviceLogged broadcasts a saved log entry to whichever clients joined
// that device's room, event name "device.logged" (matching goacs-php's
// broadcastAs()). Wired into repository.OnLogSaved at server startup so
// repository/mysql (which cannot import this package - see
// repository/hooks.go) can trigger it without an import cycle. l is expected
// to be *log.Log; typed as interface{} to match the OnLogSaved signature.
func EmitDeviceLogged(l interface{}) {
	entry, ok := l.(*log.Log)
	if !ok || entry.CPEUUID == "" {
		return
	}

	GetSocketServer().BroadcastToRoom("/", deviceRoom(entry.CPEUUID), "device.logged", entry)
}

func onConnect(s socketio.Conn) error {
	s.SetContext("")
	stdlog.Println("Client connected", s.ID())
	return nil
}

func OnDisconnect(s socketio.Conn, reason string) {
	fmt.Println("closed", reason)
}

// allowedOriginChecker mirrors the CORS_ALLOWED_ORIGINS allow-list used for
// the REST API (see server.go) so the WebSocket/polling handshake isn't
// rejected by gorilla/websocket's default same-origin check, which would
// otherwise 403 every cross-origin request from the separately-deployed
// frontend dev server / production build.
func allowedOriginChecker() func(r *http.Request) bool {
	env := new(lib.Env)
	allowed := strings.Split(env.Get("CORS_ALLOWED_ORIGINS", "http://localhost:8080"), ",")

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, candidate := range allowed {
			if strings.TrimSpace(candidate) == origin {
				return true
			}
		}
		return false
	}
}

func validSocketToken(tokenString string) bool {
	if tokenString == "" {
		return false
	}

	env := new(lib.Env)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(env.Get("JWT_SECRET", "")), nil
	})

	return err == nil && token.Valid
}
