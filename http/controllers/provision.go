package controllers

import (
	"github.com/gin-gonic/gin"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/models/provisions"
	"goacs/repository"
	"goacs/repository/mysql"
	"strconv"
)

type ProvisionRuleRequest struct {
	Parameter string `json:"parameter" validate:"required"`
	Operator  string `json:"operator" validate:"required"`
	Value     string `json:"value"`
}

type ProvisionStoreRequest struct {
	Name     string                 `json:"name" validate:"required"`
	Events   string                 `json:"events"`
	Requests string                 `json:"requests"`
	Script   []string               `json:"script" validate:"required"`
	Rules    []ProvisionRuleRequest `json:"rules"`
}

func GetProvisionsList(ctx *gin.Context) {
	paginatorRequest := repository.PaginatorRequestFromContext(ctx)
	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	list, total := provisionRepository.List(paginatorRequest)
	response.ResponsePaginatior(ctx, repository.NewPaginatorResponse(paginatorRequest, total, list))
}

func GetProvision(ctx *gin.Context) {
	provision, err := provisionFromContext(ctx)
	if err != nil {
		return
	}

	response.ResponseData(ctx, provision)
}

func CreateProvision(ctx *gin.Context) {
	var storeRequest ProvisionStoreRequest
	_ = ctx.ShouldBindJSON(&storeRequest)

	validator := request.NewApiValidator(ctx, storeRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	provision := provisionFromRequest(storeRequest)

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	if err := provisionRepository.Create(&provision); err != nil {
		response.Response500(ctx, "Cannot create provision", err)
		return
	}

	response.ResponseData(ctx, provision)
}

func UpdateProvision(ctx *gin.Context) {
	existing, err := provisionFromContext(ctx)
	if err != nil {
		return
	}

	var storeRequest ProvisionStoreRequest
	_ = ctx.ShouldBindJSON(&storeRequest)

	validator := request.NewApiValidator(ctx, storeRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	provision := provisionFromRequest(storeRequest)
	provision.Id = existing.Id

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	if err := provisionRepository.Update(&provision); err != nil {
		response.Response500(ctx, "Cannot update provision", err)
		return
	}

	response.ResponseData(ctx, provision)
}

func DeleteProvision(ctx *gin.Context) {
	existing, err := provisionFromContext(ctx)
	if err != nil {
		return
	}

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	if err := provisionRepository.Delete(existing.Id); err != nil {
		response.Response500(ctx, "Cannot delete provision", err)
		return
	}

	response.ResponseData(ctx, "")
}

func CloneProvision(ctx *gin.Context) {
	existing, err := provisionFromContext(ctx)
	if err != nil {
		return
	}

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	clone, err := provisionRepository.Clone(existing.Id)
	if err != nil {
		response.Response500(ctx, "Cannot clone provision", err)
		return
	}

	response.ResponseData(ctx, clone)
}

func provisionFromRequest(storeRequest ProvisionStoreRequest) provisions.Provision {
	provision := provisions.Provision{
		Name:     storeRequest.Name,
		Events:   storeRequest.Events,
		Requests: storeRequest.Requests,
		Script:   storeRequest.Script,
	}

	for _, rule := range storeRequest.Rules {
		provision.Rules = append(provision.Rules, provisions.ProvisionRule{
			Parameter: rule.Parameter,
			Operator:  rule.Operator,
			Value:     rule.Value,
		})
	}

	return provision
}

func provisionFromContext(ctx *gin.Context) (*provisions.Provision, error) {
	id, err := strconv.ParseInt(ctx.Param("provision"), 10, 64)
	if err != nil {
		response.ResponseError(ctx, 400, "Invalid provision id", "")
		return nil, err
	}

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	provision, err := provisionRepository.Find(id)
	if err != nil {
		response.ResponseError(ctx, 404, "Not found", "")
		return nil, err
	}

	return provision, nil
}
