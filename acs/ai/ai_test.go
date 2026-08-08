package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScriptResponse_ExtractsLuaBlockAndExplanation(t *testing.T) {
	text := "Sure, here you go:\n\n```lua\nlog(\"hi\")\nsetParameter(\"a\", \"b\")\n```\n\nThis logs then sets a parameter."

	result := parseScriptResponse(text)

	assert.Equal(t, "log(\"hi\")\nsetParameter(\"a\", \"b\")", result.Script)
	assert.Contains(t, result.Explanation, "Sure, here you go:")
	assert.Contains(t, result.Explanation, "This logs then sets a parameter.")
	assert.NotContains(t, result.Explanation, "```")
}

func TestParseScriptResponse_FallsBackToGenericFence(t *testing.T) {
	text := "```\nreboot()\n```"

	result := parseScriptResponse(text)

	assert.Equal(t, "reboot()", result.Script)
}

func TestParseScriptResponse_FallsBackToWholeTextWithoutFence(t *testing.T) {
	text := "  kick()  "

	result := parseScriptResponse(text)

	assert.Equal(t, "kick()", result.Script)
	assert.Empty(t, result.Explanation)
}

func TestBuildUserMessage_IncludesContextWhenPresent(t *testing.T) {
	msg := buildUserMessage(GenerateRequest{
		Prompt:   "reboot the device",
		Events:   "1 BOOT",
		Requests: "inform",
	})

	assert.Contains(t, msg, "1 BOOT")
	assert.Contains(t, msg, "inform")
	assert.Contains(t, msg, "reboot the device")
}

func TestBuildUserMessage_OmitsContextSectionWhenAbsent(t *testing.T) {
	msg := buildUserMessage(GenerateRequest{Prompt: "reboot the device"})

	assert.NotContains(t, msg, "Trigger events")
	assert.NotContains(t, msg, "currently in the editor")
	assert.Contains(t, msg, "reboot the device")
}

func TestBuildUserMessage_IncludesCurrentScriptWhenPresent(t *testing.T) {
	msg := buildUserMessage(GenerateRequest{
		Prompt:        "also log the SSID",
		CurrentScript: "local ssid = \"foo\"\nsetParameter(\"a\", ssid)",
	})

	assert.Contains(t, msg, "currently in the editor")
	assert.Contains(t, msg, "local ssid = \"foo\"")
	assert.Contains(t, msg, "Modify or extend that script")
	assert.Contains(t, msg, "also log the SSID")
}

func TestNewProvider_UnknownProviderErrors(t *testing.T) {
	_, err := NewProvider(AIConfig{Provider: "does-not-exist"})
	assert.Error(t, err)
}

func TestNewProvider_KnownProviders(t *testing.T) {
	p, err := NewProvider(AIConfig{Provider: ProviderAnthropic})
	assert.NoError(t, err)
	assert.NotNil(t, p)

	p, err = NewProvider(AIConfig{Provider: ProviderOpenAICompatible})
	assert.NoError(t, err)
	assert.NotNil(t, p)
}
