package logic

import (
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

func TestRequestListMatches(t *testing.T) {
	assert.True(t, requestListMatches(nil, "Inform"), "empty configured list matches any request type")
	assert.True(t, requestListMatches([]string{"Inform", "GetParameterValuesResponse"}, "Inform"))
	assert.False(t, requestListMatches([]string{"Inform"}, "SetParameterValuesProcessor"))
}
