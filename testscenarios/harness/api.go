//go:build scenario

package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// Flag mirrors acs/types.Flag's JSON shape (a ParameterRequest/TemplateParameter
// flag object), spelled out here rather than imported so this package never
// depends on the engine it's black-box testing.
type Flag struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	AddObject    bool `json:"add_object"`
	System       bool `json:"system"`
	PeriodicRead bool `json:"periodic_read"`
	Important    bool `json:"important"`
	Send         bool `json:"send"`
}

// FlagRWS is the common default flag combination scripts/templates use for a
// value that should be read, writable, and pushed to the CPE.
var FlagRWS = Flag{Read: true, Write: true, Send: true}

type ValueStruct struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type ParameterValue struct {
	Name        string      `json:"name"`
	ValueStruct ValueStruct `json:"valuestruct"`
	Flag        Flag        `json:"flag"`
}

type CPE struct {
	UUID            string `json:"uuid"`
	SerialNumber    string `json:"serial_number"`
	OUI             string `json:"oui"`
	ProductClass    string `json:"product_class"`
	Manufacturer    string `json:"manufacturer"`
	SoftwareVersion string `json:"software_version"`
	HardwareVersion string `json:"hardware_version"`
}

type ProvisionRule struct {
	Parameter string `json:"parameter"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
}

type Provision struct {
	Id       int64           `json:"id"`
	Name     string          `json:"name"`
	Events   string          `json:"events"`
	Requests string          `json:"requests"`
	Script   []string        `json:"script"`
	Rules    []ProvisionRule `json:"rules"`
}

type Template struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type LogEntry struct {
	SessionId string `json:"session_id"`
	From      string `json:"from"`
	Type      string `json:"type"`
	FullXML   string `json:"full_xml"`
}

type DeviceTask struct {
	Id      int64  `json:"id"`
	ForID   string `json:"for_id"`
	Event   string `json:"event"`
	Task    string `json:"task"`
	Payload string `json:"payload"`
}

// Client is a thin REST client for goacs-go's admin API, authenticated with a
// JWT obtained via Login. It exercises the same envelope/validation path the
// Vue admin panel does, per AGENTS.md's documented response shapes.
type Client struct {
	BaseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

type envelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type paginatedEnvelope struct {
	Total int             `json:"total"`
	Data  json.RawMessage `json:"data"`
}

// Login authenticates against the seeded default admin/admin account (see
// contrib/database/01_initial.sql) and stores the JWT for subsequent calls.
func (c *Client) Login(t *testing.T, username, password string) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	var env envelope
	c.doRaw(t, http.MethodPost, "/api/auth/login", body, &env)

	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &login); err != nil {
		t.Fatalf("harness: parsing login response: %v", err)
	}
	if login.Token == "" {
		t.Fatalf("harness: login did not return a token (response: %s)", env.Message)
	}
	c.token = login.Token
}

// do performs an authenticated request and unmarshals the {"message","data"}
// envelope's data field into out (pass nil to ignore the body).
func (c *Client) do(t *testing.T, method, path string, body any, out any) {
	t.Helper()

	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("harness: marshaling request body for %s %s: %v", method, path, err)
		}
	}

	var env envelope
	c.doRaw(t, method, path, raw, &env)

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			t.Fatalf("harness: parsing response data for %s %s: %v (raw: %s)", method, path, err, env.Data)
		}
	}
}

// doPaginated is like do, but unwraps the paginator envelope instead
// ({"page","per_page",...,"data":[...]}) and returns the total row count.
func (c *Client) doPaginated(t *testing.T, method, path string, out any) int {
	t.Helper()

	var env paginatedEnvelope
	c.doRaw(t, method, path, nil, &env)

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			t.Fatalf("harness: parsing paginated response for %s %s: %v (raw: %s)", method, path, err, env.Data)
		}
	}
	return env.Total
}

func (c *Client) doRaw(t *testing.T, method, path string, body []byte, out any) {
	t.Helper()

	req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("harness: building request %s %s: %v", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("harness: %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("harness: %s %s: unexpected status %d: %s", method, path, resp.StatusCode, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		t.Fatalf("harness: %s %s: parsing response envelope: %v (raw: %s)", method, path, err, respBody)
	}
}

// --- Provisions ---

func (c *Client) CreateProvision(t *testing.T, p Provision) Provision {
	t.Helper()
	var created Provision
	c.do(t, http.MethodPost, "/api/provision", p, &created)
	return created
}

func (c *Client) DeleteProvision(t *testing.T, id int64) {
	t.Helper()
	c.do(t, http.MethodDelete, fmt.Sprintf("/api/provision/%d", id), nil, nil)
}

func (c *Client) ListProvisions(t *testing.T) []Provision {
	t.Helper()
	var page []Provision
	c.doPaginated(t, http.MethodGet, "/api/provision?per_page=1000", &page)
	return page
}

// DeleteProvisionsNamed removes every provision matching one of the given
// names, if present - used to strip seed/demo provisions
// (contrib/database/04_multiplay_wifi_provision.sql's "Multiplay WiFi + ACS
// credentials", matching Events="0 BOOTSTRAP,1 BOOT" and Requests="" so it
// runs on every device's every bootstrap/boot session) that would otherwise
// run alongside every scenario's own provisions and, since it uses several
// blocking calls itself, add unpredictable extra round-trips to sessions
// scenarios depend on behaving deterministically.
func (c *Client) DeleteProvisionsNamed(t *testing.T, names ...string) {
	t.Helper()
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	for _, p := range c.ListProvisions(t) {
		if want[p.Name] {
			c.DeleteProvision(t, p.Id)
		}
	}
}

// --- Templates ---
//
// CreateTemplate's response does not echo back the created row (see
// http/controllers/templates.go CreateTemplate), and there is no by-name
// lookup endpoint - only List (GetTemplatesList). CreateAndFindTemplate seeds
// then resolves the id by scanning the list for the (test-unique) name.

func (c *Client) CreateAndFindTemplate(t *testing.T, name string) Template {
	t.Helper()
	c.do(t, http.MethodPost, "/api/template", map[string]string{"name": name}, nil)

	var page []Template
	total := c.doPaginated(t, http.MethodGet, "/api/template?per_page=1000", &page)
	for _, tpl := range page {
		if tpl.Name == name {
			return tpl
		}
	}
	t.Fatalf("harness: template %q not found after creation (listed %d templates)", name, total)
	return Template{}
}

func (c *Client) StoreTemplateParameter(t *testing.T, templateId int64, name, value string, flag Flag) {
	t.Helper()
	c.do(t, http.MethodPost, fmt.Sprintf("/api/template/%d/parameters", templateId), map[string]any{
		"template_id": templateId,
		"name":        name,
		"value":       value,
		"flag":        flag,
	}, nil)
}

func (c *Client) AssignTemplateToDevice(t *testing.T, cpeUUID string, templateId, priority int64) {
	t.Helper()
	c.do(t, http.MethodPost, fmt.Sprintf("/api/device/%s/templates", cpeUUID), map[string]int64{
		"template_id": templateId,
		"priority":    priority,
	}, nil)
}

func (c *Client) UnassignTemplateFromDevice(t *testing.T, cpeUUID string, templateId int64) {
	t.Helper()
	c.do(t, http.MethodDelete, fmt.Sprintf("/api/device/%s/templates/%d", cpeUUID, templateId), nil, nil)
}

// --- Devices ---

// FindDeviceBySerial waits (up to timeout) for a CPE with this serial number
// to exist - i.e. for goacs-client to have Informed at least once - and
// returns it. Serial numbers are allocated per-scenario and unique, so
// filter[serial_number] always resolves to at most one row.
func (c *Client) FindDeviceBySerial(t *testing.T, serial string, timeout time.Duration) CPE {
	t.Helper()

	deadline := time.Now().Add(timeout)
	path := "/api/device?filter[serial_number]=" + url.QueryEscape(serial) + "&per_page=1"
	for {
		var page []CPE
		c.doPaginated(t, http.MethodGet, path, &page)
		if len(page) == 1 {
			return page[0]
		}
		if time.Now().After(deadline) {
			t.Fatalf("harness: no CPE with serial_number=%s found within %s", serial, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Client) GetDeviceParameters(t *testing.T, cpeUUID string) []ParameterValue {
	t.Helper()
	var params []ParameterValue
	c.doPaginated(t, http.MethodGet, fmt.Sprintf("/api/device/%s/parameters?per_page=10000", cpeUUID), &params)
	return params
}

// FindDeviceParameter returns the named parameter and true, or a zero value
// and false if it isn't (yet) recorded for this device.
func (c *Client) FindDeviceParameter(t *testing.T, cpeUUID, name string) (ParameterValue, bool) {
	t.Helper()
	for _, p := range c.GetDeviceParameters(t, cpeUUID) {
		if p.Name == name {
			return p, true
		}
	}
	return ParameterValue{}, false
}

// PutDeviceParameter ensures a cpe_parameters row exists for name and sets
// it to value. It always tries POST (CreateParameter, a plain INSERT) first
// and tolerates its failure - repository/mysql/cperepository.go's
// UpdateParameter (used by PUT) is a plain SQL UPDATE with no insert
// fallback, so calling PUT alone on a parameter name with no existing row
// (e.g. one this device's own parameter walk hasn't reached yet) would
// silently do nothing.
func (c *Client) PutDeviceParameter(t *testing.T, cpeUUID, name, value string, flag Flag) {
	t.Helper()

	body, _ := json.Marshal(map[string]any{"name": name, "value": value, "flag": flag})

	createReq, _ := http.NewRequest(http.MethodPost, c.BaseURL+fmt.Sprintf("/api/device/%s/parameters", cpeUUID), bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+c.token)
	if resp, err := c.http.Do(createReq); err == nil {
		resp.Body.Close()
	}

	// PUT /device/:uuid/parameters returns a bare 204 "" (no envelope) - see
	// the comment in frontend/src/api/endpoints/device.api.ts - so this call
	// bypasses the envelope-unwrapping helpers and just checks the status.
	req, _ := http.NewRequest(http.MethodPut, c.BaseURL+fmt.Sprintf("/api/device/%s/parameters", cpeUUID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("harness: PUT device parameter: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("harness: PUT device parameter: unexpected status %d: %s", resp.StatusCode, respBody)
	}
}

func (c *Client) GetDeviceLogs(t *testing.T, cpeUUID string) []LogEntry {
	t.Helper()
	var logs []LogEntry
	c.doPaginated(t, http.MethodGet, fmt.Sprintf("/api/device/%s/logs?per_page=10000", cpeUUID), &logs)
	return logs
}

func (c *Client) GetDeviceTasks(t *testing.T, cpeUUID string) []DeviceTask {
	t.Helper()
	var tasks []DeviceTask
	c.do(t, http.MethodGet, fmt.Sprintf("/api/device/%s/tasks", cpeUUID), nil, &tasks)
	return tasks
}

func (c *Client) GetDeviceTemplates(t *testing.T, cpeUUID string) []struct {
	Template
	Priority int64 `json:"priority"`
} {
	t.Helper()
	var out []struct {
		Template
		Priority int64 `json:"priority"`
	}
	c.do(t, http.MethodGet, fmt.Sprintf("/api/device/%s/templates", cpeUUID), nil, &out)
	return out
}

// TriggerProvisionNow forces the device's next Inform into a full
// GetParameterNames/GetParameterValues walk, then issues a Connection Request
// so that happens immediately (see http/controllers/device.go GetDeviceProvision).
func (c *Client) TriggerProvisionNow(t *testing.T, cpeUUID string) {
	t.Helper()
	c.do(t, http.MethodGet, fmt.Sprintf("/api/device/%s/provision", cpeUUID), nil, nil)
}

func (c *Client) Kick(t *testing.T, cpeUUID string) {
	t.Helper()
	c.do(t, http.MethodGet, fmt.Sprintf("/api/device/%s/kick", cpeUUID), nil, nil)
}

// --- Config ---

func (c *Client) SaveConfig(t *testing.T, values map[string]string) {
	t.Helper()
	c.do(t, http.MethodPost, "/api/config", map[string]any{"config": values}, nil)
}
