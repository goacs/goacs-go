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
	if !eventListMatches(p.EventsList(), eventCodes) {
		return false
	}
	if !requestListMatches(p.RequestsList(), requestType) {
		return false
	}
	return m.evaluateRules(p.Rules)
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

// evaluateRules requires every rule on a provision to pass (logical AND), same as PHP.
func (m *ProvisionMatcher) evaluateRules(rules []provisions.ProvisionRule) bool {
	for _, rule := range rules {
		if !m.evaluateRule(rule) {
			return false
		}
	}
	return true
}

// evaluateRule resolves a "device.root." prefix in the rule's Parameter to the current
// session's actual root (e.g. "InternetGatewayDevice." or "Device.") - the same prefix
// Lua provisioning scripts use for the same purpose (see acs/scripts/README.md), so a
// rule's Parameter and a script's parameter paths read the same way.
func (m *ProvisionMatcher) evaluateRule(rule provisions.ProvisionRule) bool {
	parameter := strings.Replace(rule.Parameter, "device.root.", m.reqRes.Session.CPE.Root+".", 1)

	// Prefer the value already loaded into this session (freshly read from the CPE);
	// fall back to the last known value cached in the DB, same order PHP checks in.
	value, err := m.reqRes.Session.CPE.GetParameterValue(parameter)
	if err != nil {
		cpeRepository := mysql.NewCPERepository(m.reqRes.DBConnection)
		dbParam, dbErr := cpeRepository.FindParameter(&m.reqRes.Session.CPE, parameter)
		if dbErr == nil && dbParam != nil {
			value = dbParam.ValueStruct.Value
		}
	}

	return provisionCondition(value, rule.Value, rule.Operator)
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
