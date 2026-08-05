package controllers

import (
	"goacs/http/response"
	"goacs/models/fault"
	"goacs/repository"
	"goacs/repository/mysql"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

const OnlineHoursOffset = 6

type dashboardResponse struct {
	DevicesCount int64         `json:"devices_count"`
	OnlineCount  int64         `json:"online_count"`
	OnlineOffset int64         `json:"online_offset"`
	FaultsCount  int64         `json:"faults_count"`
	Faults       []fault.Fault `json:"faults"`
}

func GetDashboardData(ctx *gin.Context) {
	cpeRepository := mysql.NewCPERepository(repository.GetConnection())
	fRepository := mysql.NewFaultRepository()
	responseData := dashboardResponse{
		DevicesCount: cpeRepository.Count(),
		OnlineCount:  cpeRepository.CountUpdatedAtAfter(time.Now().Add(-1 * OnlineHoursOffset * time.Hour)),
		OnlineOffset: OnlineHoursOffset,
		FaultsCount:  fRepository.CountLastDay(),
		Faults:       fRepository.GetLastDay(100),
	}

	log.Println(responseData.Faults)

	response.ResponseData(ctx, responseData)

}
