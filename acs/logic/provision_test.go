package logic

import (
	"goacs/acs"
	acshttp "goacs/acs/http"
	acsxml "goacs/acs/types"
	"goacs/models/provisions"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvisionCondition(t *testing.T) {
	cases := []struct {
		name       string
		paramValue string
		ruleValue  string
		operator   string
		want       bool
	}{
		{"equal true", "600", "600", "==", true},
		{"equal false", "600", "601", "==", false},
		{"not equal true", "600", "601", "!=", true},
		{"not equal false", "600", "600", "!=", false},
		{"in true", "b", "a,b,c", "in", true},
		{"in false", "z", "a,b,c", "in", false},
		{"not in true", "z", "a,b,c", "not in", true},
		{"not in false", "b", "a,b,c", "not in", false},
		{"gt true", "10", "5", ">", true},
		{"gt false", "5", "10", ">", false},
		{"gte equal", "5", "5", ">=", true},
		{"lt true", "3", "5", "<", true},
		{"lte equal", "5", "5", "<=", true},
		{"numeric garbage is not a match", "not-a-number", "5", ">", false},
		{"unknown operator", "5", "5", "~=", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, provisionCondition(tc.paramValue, tc.ruleValue, tc.operator))
		})
	}
}

func TestEventListMatches(t *testing.T) {
	assert.True(t, eventListMatches(nil, []string{"1 BOOT"}), "empty configured list matches any event")
	assert.True(t, eventListMatches([]string{"0 BOOTSTRAP", "1 BOOT"}, []string{"1 BOOT"}))
	assert.False(t, eventListMatches([]string{"0 BOOTSTRAP"}, []string{"2 PERIODIC"}))
}

func TestMatches_DisabledSkipped(t *testing.T) {
	matcher := &ProvisionMatcher{reqRes: &acshttp.CPERequest{Session: &acs.ACSSession{}}}

	disabled := provisions.Provision{Enabled: false}
	assert.False(t, matcher.matches(disabled, nil, "Inform"), "disabled provision must never match, even with empty event/request/rules")

	enabled := provisions.Provision{Enabled: true}
	assert.True(t, matcher.matches(enabled, nil, "Inform"), "sanity check: an otherwise-matching enabled provision does match")
}

// TestEvaluateProvisionMatch exercises the resolver-backed evaluator the /provision/simulate
// endpoint uses (see http/controllers/provision.go), proving it shares the same enabled/
// event/request/condition semantics as the live-session matcher above, just against a
// caller-supplied resolve func instead of a CPE session.
func TestEvaluateProvisionMatch(t *testing.T) {
	p := provisions.Provision{
		Enabled:  true,
		Events:   "0 BOOTSTRAP",
		Requests: "",
		Rules: []provisions.ProvisionRule{
			{Parameter: "DeviceInfo.SoftwareVersion", Operator: ">=", Value: "2.0"},
		},
	}
	resolveKnown := func(parameter string) string {
		if parameter == "DeviceInfo.SoftwareVersion" {
			return "2.5"
		}
		return ""
	}

	eval := EvaluateProvisionMatch(p, []string{"0 BOOTSTRAP"}, "Inform", resolveKnown)
	assert.True(t, eval.EventMatch)
	assert.True(t, eval.RequestMatch, "empty configured Requests matches anything")
	assert.True(t, eval.ConditionsMatch)
	assert.True(t, eval.OverallMatch)
	if assert.Len(t, eval.ConditionResults, 1) {
		assert.Equal(t, "2.5", eval.ConditionResults[0].Actual)
		assert.True(t, eval.ConditionResults[0].Passed)
	}

	disabled := p
	disabled.Enabled = false
	evalDisabled := EvaluateProvisionMatch(disabled, []string{"0 BOOTSTRAP"}, "Inform", resolveKnown)
	assert.False(t, evalDisabled.OverallMatch, "disabled must short-circuit OverallMatch even when everything else matches")

	resolveMissing := func(string) string { return "" }
	evalMissing := EvaluateProvisionMatch(p, []string{"0 BOOTSTRAP"}, "Inform", resolveMissing)
	assert.False(t, evalMissing.ConditionsMatch, "an unresolved/missing parameter must resolve to empty string and fail the numeric condition")
	assert.Equal(t, "", evalMissing.ConditionResults[0].Actual)
}

func TestRequestListMatches(t *testing.T) {
	assert.True(t, requestListMatches(nil, "Inform"), "empty configured list matches any request type")
	assert.True(t, requestListMatches([]string{"Inform", "GetParameterValuesResponse"}, "Inform"))
	assert.False(t, requestListMatches([]string{"Inform"}, "SetParameterValuesProcessor"))
}

// TestEvaluateRule_DeviceRootPrefix proves a rule's Parameter resolves "device.root." to
// the session's actual root the same way Lua scripts resolve device.root (see
// acs/scripts/README.md), so writing a provisioning rule and writing a script read the
// same way.
func TestEvaluateRule_DeviceRootPrefix(t *testing.T) {
	session := &acs.ACSSession{}
	session.CPE.Root = "InternetGatewayDevice"
	session.CPE.AddParameterValues([]acsxml.ParameterValueStruct{
		{Name: "InternetGatewayDevice.DeviceInfo.ProductClass", ValueStruct: acsxml.ValueStruct{Value: "ONT-5G"}},
	})

	matcher := &ProvisionMatcher{reqRes: &acshttp.CPERequest{Session: session}}

	rule := provisions.ProvisionRule{
		Parameter: "device.root.DeviceInfo.ProductClass",
		Operator:  "==",
		Value:     "ONT-5G",
	}
	assert.True(t, matcher.evaluateRule(rule))

	rule.Value = "some-other-class"
	assert.False(t, matcher.evaluateRule(rule))
}
