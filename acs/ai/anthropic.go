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
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicModel   = "claude-sonnet-5"
	anthropicVersion        = "2023-06-01"
)

type anthropicProvider struct {
	cfg    AIConfig
	client *http.Client
}

func newAnthropicProvider(cfg AIConfig) *anthropicProvider {
	// No client-level Timeout: the request already carries the caller's context deadline
	// (see http/controllers/ai.go), so a second, shorter timeout here would just cut
	// slower backends off early without adding any real protection.
	return &anthropicProvider{cfg: cfg, client: &http.Client{}}
}

type anthropicRequestBody struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponseBody struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *anthropicProvider) GenerateScript(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	baseURL := p.cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	model := p.cfg.Model
	if model == "" {
		model = defaultAnthropicModel
	}

	payload, err := json.Marshal(anthropicRequestBody{
		Model:     model,
		MaxTokens: 4096,
		System:    buildSystemPrompt(),
		Messages: []anthropicMessage{
			{Role: "user", Content: buildUserMessage(req)},
		},
	})
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, fmt.Errorf("ai: anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}

	var parsed anthropicResponseBody
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return GenerateResponse{}, fmt.Errorf("ai: anthropic response decode failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return GenerateResponse{}, fmt.Errorf("ai: anthropic error (%d): %s", resp.StatusCode, parsed.Error.Message)
		}
		return GenerateResponse{}, fmt.Errorf("ai: anthropic error (%d): %s", resp.StatusCode, string(respBody))
	}

	var text strings.Builder
	for _, block := range parsed.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return parseScriptResponse(text.String()), nil
}
