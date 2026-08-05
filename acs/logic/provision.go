package logic

import (
	acshttp "goacs/acs/http"
	"goacs/models/provisions"
	"goacs/models/tasks"
	"goacs/repository/mysql"
	"log"
	"strconv"
	"strings"
)

// ProvisionMatcher selects provisioning rules matching the current session/request and
// queues their scripts as RunScriptTasks. Port of goacs-php's App\ACS\Logic\Provision.
type ProvisionMatcher struct {
	reqRes *acshttp.CPERequest
}

func NewProvisionMatcher(reqRes *acshttp.CPERequest) *ProvisionMatcher {
	return &ProvisionMatcher{reqRes: reqRes}
}

// QueueTasks loads every provisioning rule, filters by event codes and request type,
// evaluates each rule's parameter conditions, and queues one RunScriptTask per script
// body (in order) for every provision that matches.
func (m *ProvisionMatcher) QueueTasks(eventCodes []string, requestType string) error {
	repo := mysql.NewProvisionRepository(m.reqRes.DBConnection)
	all, err := repo.GetAllWithRules()
	if err != nil {
		return err
	}

	for _, provision := range all {
		if !m.matches(provision, eventCodes, requestType) {
			continue
		}

		log.Println("Provision matched:", provision.Name)

		for _, script := range provision.Script {
			task := tasks.NewCPETask(m.reqRes.Session.CPE.UUID)
			task.AsScript(script)
			m.reqRes.Session.AddTask(task)
		}
	}

	return nil
}

func (m *ProvisionMatcher) matches(p provisions.Provision, eventCodes []string, requestType string) bool {
	if !p.Enabled {
		return false
	}
	return EvaluateProvisionMatch(p, eventCodes, requestType, m.resolveLiveParameter).OverallMatch
}

// ConditionEvaluation is the per-rule result of evaluating a ProvisionRule against a
// resolved parameter value.
type ConditionEvaluation struct {
	Rule   provisions.ProvisionRule
	Actual string
	Passed bool
}

// MatchEvaluation is the full detail of matching one provision against a trigger, not
// just the pass/fail bool QueueTasks needs internally - used by the simulator to show
// which parts of a provision matched or not.
type MatchEvaluation struct {
	Provision        provisions.Provision
	EventMatch       bool
	RequestMatch     bool
	ConditionResults []ConditionEvaluation
	ConditionsMatch  bool
	OverallMatch     bool
}

// EvaluateProvisionMatch computes match details for one provision against the given
// trigger, resolving each rule's parameter value via resolve. This is the single
// implementation of the event/request/condition matching semantics - QueueTasks (via
// matches/resolveLiveParameter) uses it against a live CPE session, and the
// /provision/simulate endpoint (see http/controllers/provision.go) uses it against a
// caller-supplied parameter map, so there is exactly one place to get this logic right.
func EvaluateProvisionMatch(p provisions.Provision, eventCodes []string, requestType string, resolve func(string) string) MatchEvaluation {
	eventMatch := eventListMatches(p.EventsList(), eventCodes)
	requestMatch := requestListMatches(p.RequestsList(), requestType)

	results := make([]ConditionEvaluation, len(p.Rules))
	conditionsMatch := true
	for i, rule := range p.Rules {
		actual := resolve(rule.Parameter)
		passed := provisionCondition(actual, rule.Value, rule.Operator)
		results[i] = ConditionEvaluation{Rule: rule, Actual: actual, Passed: passed}
		if !passed {
			conditionsMatch = false
		}
	}

	return MatchEvaluation{
		Provision:        p,
		EventMatch:       eventMatch,
		RequestMatch:     requestMatch,
		ConditionResults: results,
		ConditionsMatch:  conditionsMatch,
		OverallMatch:     p.Enabled && eventMatch && requestMatch && conditionsMatch,
	}
}

// eventListMatches: an empty configured list matches any event (same as an empty PHP
// "events" field); otherwise at least one configured code must be among the CWMP
// Inform's event codes carried by the session.
func eventListMatches(configured []string, current []string) bool {
	if len(configured) == 0 {
		return true
	}
	for _, want := range configured {
		for _, have := range current {
			if want == have {
				return true
			}
		}
	}
	return false
}

func requestListMatches(configured []string, requestType string) bool {
	if len(configured) == 0 {
		return true
	}
	for _, want := range configured {
		if want == requestType {
			return true
		}
	}
	return false
}

// evaluateRule resolves a rule's parameter value via the live session and checks it
// against the rule - kept as a thin wrapper for existing direct-call sites/tests.
func (m *ProvisionMatcher) evaluateRule(rule provisions.ProvisionRule) bool {
	return provisionCondition(m.resolveLiveParameter(rule.Parameter), rule.Value, rule.Operator)
}

// resolveLiveParameter resolves a "device.root." prefix in a rule's Parameter to the
// current session's actual root (e.g. "InternetGatewayDevice." or "Device.") - the same
// prefix Lua provisioning scripts use for the same purpose (see acs/scripts/README.md) -
// then reads its value, preferring the value already loaded into this session (freshly
// read from the CPE) and falling back to the last known value cached in the DB.
func (m *ProvisionMatcher) resolveLiveParameter(parameter string) string {
	parameter = strings.Replace(parameter, "device.root.", m.reqRes.Session.CPE.Root+".", 1)

	value, err := m.reqRes.Session.CPE.GetParameterValue(parameter)
	if err != nil {
		cpeRepository := mysql.NewCPERepository(m.reqRes.DBConnection)
		dbParam, dbErr := cpeRepository.FindParameter(&m.reqRes.Session.CPE, parameter)
		if dbErr == nil && dbParam != nil {
			value = dbParam.ValueStruct.Value
		}
	}

	return value
}

func provisionCondition(paramValue, ruleValue, operator string) bool {
	switch operator {
	case "==":
		return paramValue == ruleValue
	case "!=":
		return paramValue != ruleValue
	case "in":
		return csvContains(ruleValue, paramValue)
	case "not in":
		return !csvContains(ruleValue, paramValue)
	case ">", ">=", "<", "<=":
		return numericCompare(paramValue, ruleValue, operator)
	default:
		log.Println("unknown provision rule operator:", operator)
		return false
	}
}

func csvContains(csv, value string) bool {
	for _, part := range strings.Split(csv, ",") {
		if strings.TrimSpace(part) == value {
			return true
		}
	}
	return false
}

func numericCompare(a, b, operator string) bool {
	af, aerr := strconv.ParseFloat(strings.TrimSpace(a), 64)
	bf, berr := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if aerr != nil || berr != nil {
		return false
	}

	switch operator {
	case ">":
		return af > bf
	case ">=":
		return af >= bf
	case "<":
		return af < bf
	case "<=":
		return af <= bf
	}
	return false
}
