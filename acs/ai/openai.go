package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4.1"
)

// openAICompatibleProvider targets any server speaking the OpenAI Chat Completions wire
// format - OpenAI itself, Azure OpenAI-compatible proxies, or self-hosted runners
// (Ollama/vLLM/etc.) - selected by pointing BaseURL at it. BaseURL is taken verbatim up to
// and including whatever version segment the server uses (not every OpenAI-compatible
// server exposes /v1 - some use /api or nothing at all), so only "/chat/completions" is
// appended here rather than assuming "/v1/...".
type openAICompatibleProvider struct {
	cfg    AIConfig
	client *http.Client
}

func newOpenAICompatibleProvider(cfg AIConfig) *openAICompatibleProvider {
	// No client-level Timeout: the request already carries the caller's context deadline
	// (see http/controllers/ai.go), so a second, shorter timeout here would just cut
	// slower self-hosted backends off early without adding any real protection.
	return &openAICompatibleProvider{cfg: cfg, client: &http.Client{}}
}

type openAIChatRequestBody struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponseBody struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *openAICompatibleProvider) GenerateScript(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	baseURL := p.cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	model := p.cfg.Model
	if model == "" {
		model = defaultOpenAIModel
	}

	payload, err := json.Marshal(openAIChatRequestBody{
		Model: model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: buildSystemPrompt()},
			{Role: "user", Content: buildUserMessage(req)},
		},
	})
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("ai: openai-compatible request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}

	var parsed openAIChatResponseBody
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return GenerateResponse{}, fmt.Errorf("ai: openai-compatible response decode failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return GenerateResponse{}, fmt.Errorf("ai: openai-compatible error (%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return GenerateResponse{}, fmt.Errorf("ai: openai-compatible error (%d): %s", resp.StatusCode, string(respBody))
	}

	if len(parsed.Choices) == 0 {
		return GenerateResponse{}, fmt.Errorf("ai: openai-compatible response had no choices")
	}

	return parseScriptResponse(parsed.Choices[0].Message.Content), nil
}
