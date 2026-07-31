package http

import (
	"github.com/gin-gonic/gin"
	"goacs/http/controllers"
	"goacs/http/middleware/jwt"
	"goacs/lib"
)

func RegisterApiRoutes(gin *gin.Engine) {
	var env lib.Env
	gin.GET("/file/:filename", controllers.DownloadFile)
	apiGroup := gin.Group("/api")
	apiGroup.Use()
	apiGroup.POST("/auth/login", controllers.Login)

	apiGroup.Use(jwt.JWTAuthMiddleware(env.Get("JWT_SECRET", "")))
	{
		apiGroup.POST("/auth/logout", controllers.Logout)
		apiGroup.POST("/auth/refresh", controllers.Refresh)

		apiGroup.GET("/dashboard", controllers.GetDashboardData)
		apiGroup.POST("/user/create", controllers.UserCreate)

		apiGroup.GET("/config", controllers.GetConfig)
		apiGroup.POST("/config", controllers.SaveConfig)
		apiGroup.GET("/settings", controllers.GetConfig)
		apiGroup.POST("/settings", controllers.SaveConfig)
		apiGroup.GET("/settings/debug", controllers.GetDebugSettings)
		apiGroup.POST("/settings/debug", controllers.SaveDebugSettings)

		apiGroup.GET("/settings/user", controllers.UserList)
		apiGroup.GET("/settings/user/:uuid", controllers.UserShow)
		apiGroup.POST("/settings/user", controllers.UserCreate)
		apiGroup.PUT("/settings/user/:uuid", controllers.UserUpdate)
		apiGroup.DELETE("/settings/user/:uuid", controllers.UserDelete)

		apiGroup.GET("/device", controllers.GetDevicesList)
		apiGroup.GET("/device/:uuid", controllers.GetDevice)
		apiGroup.DELETE("/device/:uuid", controllers.DeleteDevice)
		apiGroup.GET("/device/:uuid/kick", controllers.Kick)
		apiGroup.GET("/device/:uuid/provision", controllers.GetDeviceProvision)
		apiGroup.GET("/device/:uuid/lookup", controllers.GetDeviceLookup)
		apiGroup.DELETE("/device/:uuid/cache", controllers.ClearDeviceCache)
		apiGroup.GET("/device/:uuid/parameters/cached/download", controllers.DownloadDeviceCachedParametersCSV)
		apiGroup.GET("/device/:uuid/parameters/cached", controllers.GetDeviceCachedParameters)
		apiGroup.PATCH("/device/:uuid/parameters/patch", controllers.PatchDeviceParameters)
		apiGroup.GET("/device/:uuid/logs/download", controllers.DownloadDeviceLogs)
		apiGroup.GET("/device/:uuid/logs", controllers.GetDeviceLogs)
		apiGroup.GET("/device/:uuid/parameters", controllers.GetDeviceParameters)
		apiGroup.POST("/device/:uuid/parameters", controllers.CreateParameter)
		apiGroup.PUT("/device/:uuid/parameters", controllers.UpdateParameter)
		apiGroup.DELETE("/device/:uuid/parameters", controllers.DeleteParameter)
		apiGroup.POST("/device/:uuid/addobject", controllers.AddObject)
		apiGroup.POST("/device/:uuid/getparametervalues", controllers.GetParameterValues)
		apiGroup.GET("/device/:uuid/tasks", controllers.GetDeviceQueuedTasks)
		apiGroup.POST("/device/:uuid/tasks", controllers.AddDeviceTask)
		apiGroup.GET("/device/:uuid/tasks/:taskid", controllers.GetDeviceTask)
		apiGroup.PUT("/device/:uuid/tasks/:taskid", controllers.UpdateDeviceTask)
		apiGroup.DELETE("/device/:uuid/tasks/:taskid", controllers.DeleteDeviceTask)
		apiGroup.GET("/device/:uuid/templates", controllers.GetDeviceTemplates)
		apiGroup.POST("/device/:uuid/templates", controllers.AssignTemplateToDevice)
		apiGroup.PATCH("/device/:uuid/templates", controllers.UpdateDeviceTemplatePriority)
		apiGroup.DELETE("/device/:uuid/templates/:template_id", controllers.UnassignTemplateFromDevice)

		apiGroup.POST("/template", controllers.CreateTemplate)
		apiGroup.GET("/template", controllers.GetTemplatesList)
		apiGroup.GET("/template/:templateid", controllers.GetTemplate)
		apiGroup.POST("/template/:templateid/parameters", controllers.StoreTemplateParameter)
		apiGroup.GET("/template/:templateid/parameters", controllers.GetTemplateParameters)
		apiGroup.POST("/template/:templateid/parameters/:parameter_uuid", controllers.UpdateTemplateParameter)
		apiGroup.DELETE("/template/:templateid/parameters/:parameter_uuid", controllers.DeleteTemplateParameter)

		apiGroup.GET("/tasks", controllers.GetGlobalTasks)
		apiGroup.POST("/tasks", controllers.AddGlobalTask)
		apiGroup.GET("/tasks/:taskid", controllers.GetGlobalTask)
		apiGroup.POST("/tasks/:taskid", controllers.UpdateGlobalTask)
		apiGroup.DELETE("/tasks/:taskid", controllers.DeleteGlobalTask)

		apiGroup.GET("/provision", controllers.GetProvisionsList)
		apiGroup.POST("/provision", controllers.CreateProvision)
		apiGroup.GET("/provision/:provision", controllers.GetProvision)
		apiGroup.POST("/provision/:provision", controllers.UpdateProvision)
		apiGroup.DELETE("/provision/:provision", controllers.DeleteProvision)
		apiGroup.GET("/provision/:provision/clone", controllers.CloneProvision)

		apiGroup.GET("/faults", controllers.GetFaults)
		apiGroup.GET("/faults/today", controllers.GetTodayFaults)

		apiGroup.GET("/file", controllers.ListFiles)
		apiGroup.POST("/file", controllers.UploadFile)
		apiGroup.GET("/file/:filename", controllers.ShowFile)
		apiGroup.DELETE("/file/:filename", controllers.DeleteFile)

	}
}
