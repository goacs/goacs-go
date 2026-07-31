package controllers

import (
	"github.com/gin-gonic/gin"
	"goacs/http/response"
	"goacs/repository"
	"goacs/repository/mysql"
)

func GetTodayFaults(ctx *gin.Context) {
	faultRepository := mysql.NewFaultRepository()
	faults := faultRepository.GetLastDay(100)
	response.ResponseData(ctx, faults)
}

func GetFaults(ctx *gin.Context) {
	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	logRepository := mysql.NewLogRepository(repository.GetConnection())
	faults, total := logRepository.ListFaults(paginatorRequest)
	response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(paginatorRequest, total, faults))
}
