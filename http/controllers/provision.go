package controllers

import (
	"github.com/gin-gonic/gin"
	"goacs/acs/logic"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/models/provisions"
	"goacs/repository"
	"goacs/repository/mysql"
	"strconv"
	"strings"
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
	Script   []string               `json:"script" validate:"required,max=1"`
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
	provision.Enabled = true

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
	provision.Priority = existing.Priority
	provision.Enabled = existing.Enabled

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

type ProvisionEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

func UpdateProvisionEnabled(ctx *gin.Context) {
	existing, err := provisionFromContext(ctx)
	if err != nil {
		return
	}

	var req ProvisionEnabledRequest
	_ = ctx.ShouldBindJSON(&req)

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	if err := provisionRepository.UpdateEnabled(existing.Id, req.Enabled); err != nil {
		response.Response500(ctx, "Cannot update provision", err)
		return
	}

	existing.Enabled = req.Enabled
	response.ResponseData(ctx, existing)
}

type ProvisionReorderRequest struct {
	Ids []int64 `json:"ids" validate:"required"`
}

func ReorderProvisions(ctx *gin.Context) {
	var req ProvisionReorderRequest
	_ = ctx.ShouldBindJSON(&req)

	validator := request.NewApiValidator(ctx, req)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	if err := provisionRepository.Reorder(req.Ids); err != nil {
		response.Response500(ctx, "Cannot reorder provisions", err)
		return
	}

	response.ResponseData(ctx, "")
}

type ProvisionSimulateParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProvisionSimulateRequest struct {
	Event   string                   `json:"event"`
	Request string                   `json:"request"`
	Root    string                   `json:"root"`
	Params  []ProvisionSimulateParam `json:"params"`
}

type ProvisionSimulateConditionResult struct {
	Parameter string `json:"parameter"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	Actual    string `json:"actual"`
	Passed    bool   `json:"passed"`
}

type ProvisionSimulateResult struct {
	ProvisionId      int64                              `json:"provision_id"`
	Name             string                             `json:"name"`
	Priority         int                                `json:"priority"`
	Enabled          bool                               `json:"enabled"`
	ScriptCount      int                                `json:"script_count"`
	EventMatch       bool                               `json:"event_match"`
	RequestMatch     bool                               `json:"request_match"`
	ConditionResults []ProvisionSimulateConditionResult `json:"condition_results"`
	ConditionsMatch  bool                               `json:"conditions_match"`
	OverallMatch     bool                               `json:"overall_match"`
}

// SimulateProvisions evaluates every provision (in priority order) against a
// caller-supplied trigger + fake device parameters, without touching a real device.
// It reuses logic.EvaluateProvisionMatch - the exact same matching implementation
// ProvisionMatcher.QueueTasks uses against a live CPE session - just backed by a
// map resolver instead, so the simulator can never drift from real matching behavior.
func SimulateProvisions(ctx *gin.Context) {
	var req ProvisionSimulateRequest
	_ = ctx.ShouldBindJSON(&req)

	params := make(map[string]string, len(req.Params))
	for _, p := range req.Params {
		params[p.Key] = p.Value
	}
	resolve := func(parameter string) string {
		if req.Root != "" {
			parameter = strings.Replace(parameter, "device.root.", req.Root+".", 1)
		}
		return params[parameter]
	}

	var eventCodes []string
	if req.Event != "" {
		eventCodes = []string{req.Event}
	}

	provisionRepository := mysql.NewProvisionRepository(repository.GetConnection())
	all, err := provisionRepository.GetAllWithRules()
	if err != nil {
		response.Response500(ctx, "Cannot load provisions", err)
		return
	}

	results := make([]ProvisionSimulateResult, len(all))
	for i, p := range all {
		eval := logic.EvaluateProvisionMatch(p, eventCodes, req.Request, resolve)

		conditionResults := make([]ProvisionSimulateConditionResult, len(eval.ConditionResults))
		for j, cr := range eval.ConditionResults {
			conditionResults[j] = ProvisionSimulateConditionResult{
				Parameter: cr.Rule.Parameter,
				Operator:  cr.Rule.Operator,
				Value:     cr.Rule.Value,
				Actual:    cr.Actual,
				Passed:    cr.Passed,
			}
		}

		results[i] = ProvisionSimulateResult{
			ProvisionId:      p.Id,
			Name:             p.Name,
			Priority:         p.Priority,
			Enabled:          p.Enabled,
			ScriptCount:      len(p.Script),
			EventMatch:       eval.EventMatch,
			RequestMatch:     eval.RequestMatch,
			ConditionResults: conditionResults,
			ConditionsMatch:  eval.ConditionsMatch,
			OverallMatch:     eval.OverallMatch,
		}
	}

	response.ResponseData(ctx, results)
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
