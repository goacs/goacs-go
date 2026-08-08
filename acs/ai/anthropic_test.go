package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicProvider_GenerateScript_SendsExpectedRequestAndParsesReply(t *testing.T) {
	var captured anthropicRequestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, anthropicVersion, r.Header.Get("anthropic-version"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Here:\n\n```lua\nkick()\n```"},
			},
		})
	}))
	defer server.Close()

	provider := newAnthropicProvider(AIConfig{APIKey: "test-key", Model: "claude-sonnet-5", BaseURL: server.URL})

	result, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.NoError(t, err)
	assert.Equal(t, "kick()", result.Script)
	assert.Equal(t, "claude-sonnet-5", captured.Model)
	assert.Contains(t, captured.System, "TR-069/CWMP")
	require.Len(t, captured.Messages, 1)
	assert.Contains(t, captured.Messages[0].Content, "kick the device")
}

func TestAnthropicProvider_GenerateScript_ReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "invalid x-api-key"},
		})
	}))
	defer server.Close()

	provider := newAnthropicProvider(AIConfig{APIKey: "bad-key", BaseURL: server.URL})

	_, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid x-api-key")
}

func TestAnthropicProvider_GenerateScript_UsesDefaultModelWhenUnset(t *testing.T) {
	var captured anthropicRequestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": "```lua\nreboot()\n```"}},
		})
	}))
	defer server.Close()

	provider := newAnthropicProvider(AIConfig{APIKey: "test-key", BaseURL: server.URL})

	_, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "reboot"})

	require.NoError(t, err)
	assert.Equal(t, defaultAnthropicModel, captured.Model)
}
