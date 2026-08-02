package http

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"goacs/acs/logic"
	"goacs/http/middleware/auth"
	"goacs/lib"
	"goacs/repository"
	"strings"
)

var Instance *gin.Engine

func Start() {
	var env lib.Env
	fmt.Println("Server setup")
	Instance = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = strings.Split(env.Get("CORS_ALLOWED_ORIGINS", "http://localhost:8080,https://localhost:8080,http://localhost:5173,https://localhost:5173"), ",")
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"Origin", "Authorization", "Content-Type", "Accept", "Content-Length", "Connection", "Upgrade"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

	Instance.Use(cors.New(corsConfig))

	NewSocketIO(Instance)
	go GetSocketServer().Serve()
	repository.OnLogSaved = EmitDeviceLogged

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
