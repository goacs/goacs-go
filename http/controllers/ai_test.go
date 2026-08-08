package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAiConfigFromValues_NotConfiguredWhenDisabled(t *testing.T) {
	_, configured := aiConfigFromValues(map[string]string{
		"ai_enabled": "0",
		"ai_api_key": "sk-something",
	})

	assert.False(t, configured)
}

func TestAiConfigFromValues_NotConfiguredWhenApiKeyMissing(t *testing.T) {
	_, configured := aiConfigFromValues(map[string]string{
		"ai_enabled": "1",
	})

	assert.False(t, configured)
}

func TestAiConfigFromValues_ConfiguredWhenEnabledWithKey(t *testing.T) {
	cfg, configured := aiConfigFromValues(map[string]string{
		"ai_enabled":  "1",
		"ai_provider": "anthropic",
		"ai_api_key":  "sk-something",
		"ai_model":    "claude-sonnet-5",
	})

	assert.True(t, configured)
	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "sk-something", cfg.APIKey)
	assert.Equal(t, "claude-sonnet-5", cfg.Model)
}
