// Package ai calls an external LLM to help write Lua provisioning scripts. It has no
// knowledge of CWMP sessions or the Lua sandbox itself - it only builds a prompt grounded
// in acs/scripts.ReferenceDoc (the same README a human would read) and parses a script back
// out of the model's reply. Two providers are supported (Anthropic Messages API and any
// OpenAI-compatible Chat Completions API); which one is used is a runtime config choice,
// not a build-time one, so both live behind the same Provider interface.
package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"goacs/acs/scripts"
)

const (
	ProviderAnthropic        = "anthropic"
	ProviderOpenAICompatible = "openai_compatible"
)

// AIConfig is built from the admin panel's generic config key/value store (ai_provider,
// ai_api_key, ai_model, ai_base_url) - see http/controllers/ai.go.
type AIConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

// GenerateRequest is the free-text prompt plus optional context that helps the model
// produce a script matching how it'll actually be invoked: the provisioning rule's trigger
// events/requests, and the script currently in the editor (if any), so a follow-up prompt
// can ask for a change to what's already there instead of always starting from scratch.
type GenerateRequest struct {
	Prompt        string
	Events        string
	Requests      string
	CurrentScript string
}

type GenerateResponse struct {
	Script      string `json:"script"`
	Explanation string `json:"explanation"`
}

type Provider interface {
	GenerateScript(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
}

func NewProvider(cfg AIConfig) (Provider, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		return newAnthropicProvider(cfg), nil
	case ProviderOpenAICompatible:
		return newOpenAICompatibleProvider(cfg), nil
	default:
		return nil, fmt.Errorf("ai: unknown provider %q", cfg.Provider)
	}
}

const systemPreamble = "You are an assistant that writes Lua provisioning scripts for a " +
	"TR-069/CWMP Auto Configuration Server (GoACS). Only use the globals and functions " +
	"documented in the reference below - never invent a function, global, or module that " +
	"isn't listed there. Standard Lua 5.1 string/table/math libraries are available " +
	"unrestricted.\n\n" +
	"Always build parameter paths off `device.root` instead of hardcoding " +
	"\"InternetGatewayDevice\" or \"Device\", so the script works on both TR-098 and " +
	"TR-181 devices.\n\n" +
	"Respond with a brief explanation followed by exactly one fenced code block " +
	"(```lua ... ```) containing the complete script. Do not include more than one code " +
	"block.\n\n" +
	"Reference:\n\n"

func buildSystemPrompt() string {
	return systemPreamble + scripts.ReferenceDoc
}

func buildUserMessage(req GenerateRequest) string {
	var b strings.Builder
	if req.Events != "" || req.Requests != "" {
		b.WriteString("Provisioning rule context:\n")
		if req.Events != "" {
			fmt.Fprintf(&b, "- Trigger events: %s\n", req.Events)
		}
		if req.Requests != "" {
			fmt.Fprintf(&b, "- Trigger requests: %s\n", req.Requests)
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(req.CurrentScript) != "" {
		b.WriteString("The script currently in the editor is:\n```lua\n")
		b.WriteString(req.CurrentScript)
		b.WriteString("\n```\n\n")
		b.WriteString("Modify or extend that script to satisfy the following request. Return the " +
			"complete updated script, not just the changed lines:\n")
	} else {
		b.WriteString("Write a Lua provisioning script that does the following:\n")
	}
	b.WriteString(req.Prompt)
	return b.String()
}

var (
	luaBlockRe     = regexp.MustCompile("(?s)```lua\\s*\\n(.*?)```")
	genericBlockRe = regexp.MustCompile("(?s)```\\w*\\s*\\n(.*?)```")
)

// parseScriptResponse pulls the Lua code block out of the model's reply, treating anything
// else in the reply as the explanation. Falls back to treating the whole reply as the
// script if the model didn't fence it as instructed - better to hand back something the
// user can inspect than to error out on a formatting slip.
func parseScriptResponse(text string) GenerateResponse {
	match := luaBlockRe.FindStringSubmatch(text)
	if match == nil {
		match = genericBlockRe.FindStringSubmatch(text)
	}
	if match == nil {
		return GenerateResponse{Script: strings.TrimSpace(text)}
	}

	script := strings.TrimSpace(match[1])
	explanation := strings.TrimSpace(strings.Replace(text, match[0], "", 1))
	return GenerateResponse{Script: script, Explanation: explanation}
}
