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

func TestOpenAICompatibleProvider_GenerateScript_SendsExpectedRequestAndParsesReply(t *testing.T) {
	var captured openAIChatRequestBody
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&captured))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "```lua\nkick()\n```"}},
			},
		})
	}))
	defer server.Close()

	// BaseURL carries no version segment here - the provider must not assume "/v1", since
	// not every OpenAI-compatible server exposes one.
	provider := newOpenAICompatibleProvider(AIConfig{APIKey: "test-key", Model: "gpt-4.1", BaseURL: server.URL})

	result, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.NoError(t, err)
	assert.Equal(t, "/chat/completions", capturedPath)
	assert.Equal(t, "kick()", result.Script)
	assert.Equal(t, "gpt-4.1", captured.Model)
	require.Len(t, captured.Messages, 2)
	assert.Equal(t, "system", captured.Messages[0].Role)
	assert.Contains(t, captured.Messages[0].Content, "TR-069/CWMP")
	assert.Equal(t, "user", captured.Messages[1].Role)
	assert.Contains(t, captured.Messages[1].Content, "kick the device")
}

func TestOpenAICompatibleProvider_GenerateScript_RespectsCustomVersionSegment(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"role": "assistant", "content": "```lua\nkick()\n```"}},
			},
		})
	}))
	defer server.Close()

	provider := newOpenAICompatibleProvider(AIConfig{APIKey: "test-key", BaseURL: server.URL + "/api"})

	_, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.NoError(t, err)
	assert.Equal(t, "/api/chat/completions", capturedPath)
}

func TestOpenAICompatibleProvider_GenerateScript_ReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "invalid api key"},
		})
	}))
	defer server.Close()

	provider := newOpenAICompatibleProvider(AIConfig{APIKey: "bad-key", BaseURL: server.URL})

	_, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestOpenAICompatibleProvider_GenerateScript_ErrorsOnNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []map[string]interface{}{}})
	}))
	defer server.Close()

	provider := newOpenAICompatibleProvider(AIConfig{APIKey: "test-key", BaseURL: server.URL})

	_, err := provider.GenerateScript(context.Background(), GenerateRequest{Prompt: "kick the device"})

	require.Error(t, err)
}
