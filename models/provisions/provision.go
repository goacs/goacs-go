// Package provisions holds the provisioning rule model: named bundles of
// (events, requests, parameter conditions) -> scripts, evaluated on every CWMP request
// to decide which RunScriptTasks get queued. Port of goacs-php's
// App\Models\Provision / App\Models\ProvisionRule and App\ACS\Logic\Provision.
package provisions

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gopkg.in/guregu/null.v4"
)

// ScriptList is a JSON array of script bodies executed sequentially - each entry is
// queued as its own RunScriptTask and commits independently, mirroring PHP's
// Provision.script (also cast to a JSON array) together with Stack::commandIsSequential.
type ScriptList []string

func (s *ScriptList) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *ScriptList) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == 0 {
			*s = nil
			return nil
		}
		return json.Unmarshal(v, s)
	case string:
		if v == "" {
			*s = nil
			return nil
		}
		return json.Unmarshal([]byte(v), s)
	case nil:
		*s = nil
		return nil
	default:
		return errors.New("invalid ScriptList value")
	}
}

type Provision struct {
	Id        int64      `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	Events    string     `db:"events" json:"events"`
	Requests  string     `db:"requests" json:"requests"`
	Script    ScriptList `db:"script" json:"script"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt null.Time  `db:"deleted_at" json:"deleted_at"`

	Rules []ProvisionRule `db:"-" json:"rules"`
}

type ProvisionRule struct {
	Id          int64  `db:"id" json:"id"`
	ProvisionId int64  `db:"provision_id" json:"provision_id"`
	Parameter   string `db:"parameter" json:"parameter"`
	Operator    string `db:"operator" json:"operator"`
	Value       string `db:"value" json:"value"`
}

// EventsList splits the CSV "events" column, e.g. "0 BOOTSTRAP,1 BOOT" -> both codes.
// An empty column matches any event, same as an empty "events" field in PHP.
func (p *Provision) EventsList() []string { return splitCSV(p.Events) }

// RequestsList splits the CSV "requests" column. An empty column matches any request
// type, same as PHP.
func (p *Provision) RequestsList() []string { return splitCSV(p.Requests) }

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}
