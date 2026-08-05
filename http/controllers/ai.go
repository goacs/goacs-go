package controllers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"goacs/acs/ai"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/repository"
	"goacs/repository/mysql"
)

type GenerateScriptRequest struct {
	Prompt   string `json:"prompt" validate:"required"`
	Events   string `json:"events"`
	Requests string `json:"requests"`
}

// aiConfigFromValues reads the ai_* keys out of the generic config store (same table/
// mechanism as pii_min, connection_request_password, etc. - see repository/mysql/
// configrepository.go). The second return value is false when the assistant hasn't been
// enabled or is missing an API key, which the caller turns into a 400 without ever
// constructing a provider or making an outbound request.
func aiConfigFromValues(values map[string]string) (ai.AIConfig, bool) {
	cfg := ai.AIConfig{
		Provider: values["ai_provider"],
		APIKey:   values["ai_api_key"],
		Model:    values["ai_model"],
		BaseURL:  values["ai_base_url"],
	}

	if values["ai_enabled"] != "1" || cfg.APIKey == "" {
		return cfg, false
	}

	return cfg, true
}

func GenerateAiScript(ctx *gin.Context) {
	var genRequest GenerateScriptRequest
	_ = ctx.ShouldBindJSON(&genRequest)

	validator := request.NewApiValidator(ctx, genRequest)
	if verr := validator.Validate(); verr != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	configRepository := mysql.NewConfigRepository(repository.GetConnection())
	cfg, configured := aiConfigFromValues(configRepository.GetValues())
	if !configured {
		// response.ResponseError's message parameter is discarded (it always hardcodes
		// "Error" - see responseMap in http/response/response.go, same as every other
		// ResponseError call in this codebase), so the human-readable text has to travel
		// in data instead for the frontend to actually display it.
		response.ResponseError(ctx, 400, "Error", "AI assistant is not configured - set it up in Settings")
		return
	}

	provider, err := ai.NewProvider(cfg)
	if err != nil {
		response.ResponseError(ctx, 400, "Error", err.Error())
		return
	}

	// Self-hosted/local model backends can take a lot longer than a hosted API to answer,
	// especially with this large a system prompt (the full scripts README), so this is
	// generous on purpose.
	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 120*time.Second)
	defer cancel()

	result, err := provider.GenerateScript(reqCtx, ai.GenerateRequest{
		Prompt:   genRequest.Prompt,
		Events:   genRequest.Events,
		Requests: genRequest.Requests,
	})
	if err != nil {
		response.ResponseError(ctx, 502, "Error", "AI script generation failed: "+err.Error())
		return
	}

	response.ResponseData(ctx, result)
}
