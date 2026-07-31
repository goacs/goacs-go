package http

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"goacs/acs/logic"
	"goacs/http/middleware/auth"
	"goacs/lib"
	"time"
)

var Instance *gin.Engine

func Start() {
	var env lib.Env
	fmt.Println("Server setup")
	Instance = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:8080", "https://localhost:8080"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Authorization", "Content-Type", "Accept", "Content-Length", "Connection", "Upgrade"}

	Instance.Use(cors.New(corsConfig))

	NewSocketIO(Instance)
	go GetSocketServer().Serve()

	go func() {
		for {
			//log.Println("sending event")
			GetSocketServer().BroadcastToRoom("/", "all", "supa event")
			time.Sleep(time.Second * 2)
		}
	}()

	registerAcsHandler(Instance)
	RegisterApiRoutes(Instance)

	var err error
	if env.Get("HTTP_TLS", "false") == "false" {
		err = Instance.Run(":" + env.Get("HTTP_PORT", "8085"))
	} else {
		err = Instance.RunTLS(
			":"+env.Get("HTTP_PORT", "8085"),
			env.Get("TLS_CERT", ""),
			env.Get("TLS_KEY", ""),
		)
	}
	fmt.Println("Instance started....")

	if err != nil {
		fmt.Println("Unable to start http server")
		return
	}
	fmt.Println("Http server started")
}

func registerAcsHandler(router *gin.Engine) {
	acsGroup := router.Group("/acs")
	acsGroup.Use(auth.ACSAuthMiddleware())

	handler := func(ctx *gin.Context) {
		defer ctx.Request.Body.Close()
		logic.HandleCPERequest(ctx.Request, ctx.Writer)
	}

	acsGroup.GET("", handler)
	acsGroup.POST("", handler)
}
