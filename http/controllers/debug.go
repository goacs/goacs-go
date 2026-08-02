package controllers

import (
	"github.com/gin-gonic/gin"
	"goacs/http/response"
	"goacs/models/cpe"
	"goacs/repository"
	"goacs/repository/mysql"
)

type DebugSettingsResponse struct {
	Debug           bool      `json:"debug"`
	DebugNewDevices bool      `json:"debug_new_devices"`
	Devices         []cpe.CPE `json:"devices"`
}

type SaveDebugSettingsRequest struct {
	Debug           bool     `json:"debug"`
	DebugNewDevices bool     `json:"debug_new_devices"`
	Devices         []string `json:"devices"`
}

func GetDebugSettings(ctx *gin.Context) {
	configRepository := mysql.NewConfigRepository(repository.GetConnection())
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())

	debugValue, _ := configRepository.GetValue("conversation_log")
	debugNewDevicesValue, _ := configRepository.GetValue("debug_new_devices")

	response.ResponseData(ctx, DebugSettingsResponse{
		Debug:           debugValue == "1" || debugValue == "true",
		DebugNewDevices: debugNewDevicesValue == "1" || debugNewDevicesValue == "true",
		Devices:         cpeRepository.ListDebugEnabled(),
	})
}

func SaveDebugSettings(ctx *gin.Context) {
	var saveRequest SaveDebugSettingsRequest
	_ = ctx.ShouldBindJSON(&saveRequest)

	configRepository := mysql.NewConfigRepository(repository.GetConnection())
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())

	_ = configRepository.SetValue("conversation_log", boolToConfigValue(saveRequest.Debug))
	_ = configRepository.SetValue("debug_new_devices", boolToConfigValue(saveRequest.DebugNewDevices))

	if err := cpeRepository.SetDebugForAll(false); err != nil {
		response.Response500(ctx, "Cannot reset device debug flags", err)
		return
	}

	if err := cpeRepository.SetDebugForUUIDs(saveRequest.Devices, true); err != nil {
		response.Response500(ctx, "Cannot set device debug flags", err)
		return
	}

	response.ResponseData(ctx, "")
}

func boolToConfigValue(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
