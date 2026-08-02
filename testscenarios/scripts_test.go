//go:build scenario

package testscenarios

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"goacs/testscenarios/harness"
)

const parameterKeyPath = "InternetGatewayDevice.ManagementServer.ParameterKey"

// TestScriptFunctions_NonBlocking exercises every acs/scripts/functions.go
// Lua function - the ones that only touch the session's local parameter
// cache/DB and never suspend the script waiting for a CWMP round-trip.
//
// Every subtest that checks a value written to an arbitrary parameter (via
// setParameter or the marker convention below) runs a warmUpDevice session
// first - see that helper's comment in helpers_test.go for why. Provisions
// are scoped to "2 PERIODIC" so they never accidentally fire during warm-up.
func TestScriptFunctions_NonBlocking(t *testing.T) {
	srv, client := newEnv(t)

	t.Run("setParameter_and_getParameterValue", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "np_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
setParameter(device.root .. ".ManagementServer.PeriodicInformInterval", "654", "RWS")
local v = getParameterValue(device.root .. ".ManagementServer.PeriodicInformInterval")
setParameter(device.root .. ".ManagementServer.ParameterKey", "read-back:" .. v, "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, "InternetGatewayDevice.ManagementServer.PeriodicInformInterval")
		if !ok || p.ValueStruct.Value != "654" {
			t.Fatalf("setParameter did not persist PeriodicInformInterval=654, got %+v (found=%v)", p, ok)
		}

		marker, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || marker.ValueStruct.Value != "read-back:654" {
			t.Fatalf("getParameterValue did not read back the value just set via setParameter, got %+v (found=%v)", marker, ok)
		}
	})

	t.Run("setParameter_invalid_flags_aborts_script", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npbadflag_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
setParameter(device.root .. ".ManagementServer.ParameterKey", "before", "Z")
setParameter(device.root .. ".ManagementServer.ParameterKey", "should-not-be-reached", "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		if p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); ok && p.ValueStruct.Value == "should-not-be-reached" {
			t.Fatalf("an invalid setParameter flag string should raise a Lua error and abort the script, but it continued: %+v", p)
		}
	})

	t.Run("parameterExist_and_parameterNotExist", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npexist_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local existsResult = "no"
if parameterExist(device.root .. ".DeviceInfo.HardwareVersion") then existsResult = "yes" end

local notExistResult = "no"
if parameterNotExist(device.root .. ".DeviceInfo.TotallyMadeUpLeaf") then notExistResult = "yes" end

setParameter(device.root .. ".ManagementServer.ParameterKey", existsResult .. "/" .. notExistResult, "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		marker, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || marker.ValueStruct.Value != "yes/yes" {
			t.Fatalf("parameterExist/parameterNotExist gave unexpected results: %+v (found=%v)", marker, ok)
		}
	})

	t.Run("deleteParameter", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npdel_set_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`setParameter(device.root .. ".ManagementServer.ParameterKey", "to-delete", "RW")`},
		})
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npdel_delete_" + serial,
			Events:   "4 VALUE CHANGE",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`deleteParameter(device.root .. ".ManagementServer.ParameterKey")`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)
		if _, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); !ok {
			t.Fatalf("setParameter did not persist ParameterKey before the delete step")
		}

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "4 VALUE CHANGE"})
		if p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); ok {
			t.Fatalf("deleteParameter did not remove ParameterKey, still found: %+v", p)
		}
	})

	t.Run("saveDevice", func(t *testing.T) {
		// Warm-up still matters here even though BulkInsertOrUpdateParameters
		// (saveDevice's persistence call) is a real upsert: on a brand-new
		// device's own bootstrap session, the parameter walk that same
		// session triggers can still finish (and re-persist the device's own
		// unmodified values) AFTER this script's saveDevice() call, clobbering
		// it - the same race warmUpDevice avoids elsewhere in this file.
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npsave_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
setParameter(device.root .. ".ManagementServer.ParameterKey", "saved-value", "RWX")
saveDevice()
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != "saved-value" {
			t.Fatalf("saveDevice did not bulk-persist the in-memory parameter value, got %+v (found=%v)", p, ok)
		}
	})

	t.Run("log", func(t *testing.T) {
		// No warm-up needed: log() only ever writes to the server process's
		// stdout (log.Printf), never to any DB table.
		rule, profile, profilesDir, serial := scopedRule(t)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "nplog_" + serial,
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`log("scenario-log-marker-%s", "details-payload")`, serial)},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})

		want := fmt.Sprintf("[script:%s] scenario-log-marker-%s: details-payload", serial, serial)
		if !strings.Contains(srv.Output(), want) {
			t.Fatalf("expected server log to contain %q, got:\n%s", want, srv.Output())
		}
	})

	t.Run("piiValue_default_range", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "nppii_default_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`setParameter(device.root .. ".ManagementServer.ParameterKey", tostring(piiValue()), "RW")`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok {
			t.Fatalf("piiValue() result was not persisted")
		}
		v, err := strconv.Atoi(p.ValueStruct.Value)
		if err != nil || v < 300 || v > 900 {
			t.Fatalf("piiValue() with no config override should fall in the documented default 300-900 range, got %q", p.ValueStruct.Value)
		}
	})

	t.Run("piiValue_configured_range", func(t *testing.T) {
		client.SaveConfig(t, map[string]string{"pii_min": "1000", "pii_max": "1001"})
		t.Cleanup(func() { client.SaveConfig(t, map[string]string{"pii_min": "300", "pii_max": "900"}) })

		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "nppii_configured_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`setParameter(device.root .. ".ManagementServer.ParameterKey", tostring(piiValue()), "RW")`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok {
			t.Fatalf("piiValue() result was not persisted")
		}
		v, err := strconv.Atoi(p.ValueStruct.Value)
		if err != nil || v < 1000 || v > 1001 {
			t.Fatalf("piiValue() should honor pii_min/pii_max=1000/1001 from config, got %q", p.ValueStruct.Value)
		}
	})

	t.Run("assignTemplateByName_and_unassignTemplateByName", func(t *testing.T) {
		// No warm-up needed: template assignment touches cpe_to_templates,
		// not cpe_parameters, and isn't subject to the same-row-must-exist
		// limitation.
		rule, profile, profilesDir, serial := scopedRule(t)
		tpl := client.CreateAndFindTemplate(t, "tpl_byname_"+serial)

		mustCreateProvision(t, client, harness.Provision{
			Name:     "npassign_" + serial,
			Events:   "0 BOOTSTRAP",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`assignTemplateByName(%q, 150)`, tpl.Name)},
		})
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npunassign_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`unassignTemplateByName(%q)`, tpl.Name)},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		assigned := client.GetDeviceTemplates(t, cpe.UUID)
		found := false
		for _, a := range assigned {
			if a.Name == tpl.Name && a.Priority == 150 {
				found = true
			}
		}
		if !found {
			t.Fatalf("assignTemplateByName did not create a cpe_to_templates row with priority 150, got %+v", assigned)
		}

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		assigned = client.GetDeviceTemplates(t, cpe.UUID)
		for _, a := range assigned {
			if a.Name == tpl.Name {
				t.Fatalf("unassignTemplateByName did not remove the template assignment, still present: %+v", a)
			}
		}
	})

	t.Run("assignTemplateById_and_unassignTemplateById", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		tpl := client.CreateAndFindTemplate(t, "tpl_byid_"+serial)

		mustCreateProvision(t, client, harness.Provision{
			Name:     "npassignid_" + serial,
			Events:   "0 BOOTSTRAP",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`assignTemplateById(%d, 120)`, tpl.Id)},
		})
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npunassignid_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`unassignTemplateById(%d)`, tpl.Id)},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		assigned := client.GetDeviceTemplates(t, cpe.UUID)
		found := false
		for _, a := range assigned {
			if a.Id == tpl.Id && a.Priority == 120 {
				found = true
			}
		}
		if !found {
			t.Fatalf("assignTemplateById did not create a cpe_to_templates row with priority 120, got %+v", assigned)
		}

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		assigned = client.GetDeviceTemplates(t, cpe.UUID)
		for _, a := range assigned {
			if a.Id == tpl.Id {
				t.Fatalf("unassignTemplateById did not remove the template assignment, still present: %+v", a)
			}
		}
	})

	t.Run("kick", func(t *testing.T) {
		// The device's Connection Request listener requires Digest auth
		// (goacs-client always protects it with the profile's
		// ManagementServer.ConnectionRequestUsername/Password), and on a
		// brand-new device's very first session goacs-go doesn't know those
		// credentials yet - they only arrive via a full parameter walk, not
		// Inform's own ParameterList. So kick()'s HTTP call gets challenged
		// (401 + WWW-Authenticate: Digest) rather than reaching the
		// callback. This checks what's actually verifiable here: that
		// kick() made a real outbound call to the registered Connection
		// Request URL at all - the full authenticated round-trip is a
		// pre-existing credential-bootstrapping gap outside this test's scope.
		rule, profile, profilesDir, serial := scopedRule(t)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npkick_" + serial,
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`kick()`},
		})

		runDevice(t, srv, harness.DeviceOpts{
			Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP",
			ConnRequest: true,
		})

		if !strings.Contains(srv.Output(), "401 Unauthorized") {
			t.Fatalf("kick() should have made an outbound call to the CPE's Connection Request URL; server output:\n%s", srv.Output())
		}
	})

	t.Run("uploadFirmware", func(t *testing.T) {
		// uploadFirmware() queues a Download task that builds its RPC from a
		// real file under FILESTORE_PATH (./storage by default,
		// http/controllers/files.go) - without one present on disk, the ACS
		// fails to build the request and never sends it.
		firmwareName := "scenario-firmware-" + harness.UniqueName(t) + ".bin"
		firmwarePath := filepath.Join(harness.RepoRoot(), "storage", firmwareName)
		if err := os.WriteFile(firmwarePath, []byte("dummy firmware image"), 0o644); err != nil {
			t.Fatalf("writing dummy firmware file: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(firmwarePath) })

		rule, profile, profilesDir, serial := scopedRule(t)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "npupload_" + serial,
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{fmt.Sprintf(`uploadFirmware(%q)`, firmwareName)},
		})

		res := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})

		if !strings.Contains(res.Stdout, "<cwmp:Download") {
			t.Fatalf("uploadFirmware should have queued a Download RPC for this session; stdout:\n%s", res.Stdout)
		}
		if !strings.Contains(res.Stdout, "<cwmp:TransferComplete") {
			t.Fatalf("goacs-client should have sent TransferComplete once its simulated transfer finished, in the same session; stdout:\n%s", res.Stdout)
		}
	})
}

// TestScriptFunctions_Blocking exercises every acs/scripts/bridge.go Lua
// function - the ones that issue a real CWMP RPC and suspend the script
// until the CPE actually replies.
func TestScriptFunctions_Blocking(t *testing.T) {
	srv, client := newEnv(t)

	t.Run("addObject_success", func(t *testing.T) {
		// Verified via the wire-level request/response XML (--verbose=true),
		// not GetDeviceParameters: the new instance's parameters have never
		// existed in cpe_parameters before, and setParameterValues persists
		// via a plain SQL UPDATE with no insert fallback (see
		// warmUpDevice's comment) - a warm-up session can't help here, since
		// the row this needs is only ever created *by* this exact call.
		rule, profile, profilesDir, serial := scopedRule(t)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpadd_" + serial,
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local obj = addObject(device.root .. ".LANDevice.1.Hosts.Host.")
setParameterValues({ [obj.path .. "HostName"] = "added-by-scenario" })
`},
		})

		res := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})

		if !strings.Contains(res.Stdout, "<cwmp:AddObjectResponse") {
			t.Fatalf("expected a successful AddObjectResponse in the wire log; stdout:\n%s", res.Stdout)
		}
		if !strings.Contains(res.Stdout, "added-by-scenario") {
			t.Fatalf("expected the SetParameterValues request to carry HostName=added-by-scenario; stdout:\n%s", res.Stdout)
		}
	})

	t.Run("addObject_fault_aborts_script", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpaddfault_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
addObject(device.root .. ".NoSuchAddableContainer.")
setParameter(device.root .. ".ManagementServer.ParameterKey", "should-not-be-reached", "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		if p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); ok && p.ValueStruct.Value == "should-not-be-reached" {
			t.Fatalf("an unwrapped addObject on an unknown container should fault and abort the script, but it continued: %+v", p)
		}
	})

	t.Run("deleteObject_success", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpdel_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			// LANDevice.1.Hosts.Host's template fields are all read-only
			// (tr098-router.yaml), so this deliberately doesn't try to
			// configure the new instance the way addObject_success does -
			// deleteObject itself doesn't need that, and a failed
			// setParameterValues here would only add an unrelated fault.
			Script: []string{`
local obj = addObject(device.root .. ".LANDevice.1.Hosts.Host.")
deleteObject(obj.path)
setParameter(device.root .. ".ManagementServer.ParameterKey", "delete-object-completed", "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != "delete-object-completed" {
			t.Fatalf("deleteObject should complete its round-trip without a Lua error, letting the script continue; got %+v (found=%v)", p, ok)
		}
	})

	t.Run("reboot_and_reinform_on_boot", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpreboot_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`reboot("scenario-reboot")`},
		})
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpreboot_marker_" + serial,
			Events:   "1 BOOT",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`setParameter(device.root .. ".ManagementServer.ParameterKey", "boot-session-ran", "RW")`},
		})

		res := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		if !res.RebootRequested {
			t.Fatalf("expected the session to end with reboot_requested=true, stdout:\n%s", res.Stdout)
		}

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "1 BOOT"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != "boot-session-ran" {
			t.Fatalf("the device should have re-Informed with event \"1 BOOT\" and matched the BOOT-scoped provision, got %+v (found=%v)", p, ok)
		}
	})

	t.Run("getParameterValues_multiple_paths", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpgpvmulti_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local vals = getParameterValues(device.root .. ".DeviceInfo.SoftwareVersion", device.root .. ".DeviceInfo.HardwareVersion")
setParameter(device.root .. ".ManagementServer.ParameterKey",
    vals[device.root .. ".DeviceInfo.SoftwareVersion"] .. "/" .. vals[device.root .. ".DeviceInfo.HardwareVersion"], "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		want := "1.0.0/" + serial // tr098-router's default SoftwareVersion, scoped device's HardwareVersion selector
		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != want {
			t.Fatalf("getParameterValues with multiple full paths gave %+v (found=%v), want %q", p, ok, want)
		}
	})

	t.Run("getParameterValues_partial_path_branch", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpgpvbranch_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local vals = getParameterValues(device.root .. ".LANDevice.1.WLANConfiguration.1.")
setParameter(device.root .. ".ManagementServer.ParameterKey", vals[device.root .. ".LANDevice.1.WLANConfiguration.1.SSID"], "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != "GoACS-Sim" {
			t.Fatalf("getParameterValues with a trailing-dot partial path should fetch the whole branch (expected SSID=GoACS-Sim), got %+v (found=%v)", p, ok)
		}
	})

	t.Run("setParameterValues_success", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		// setParameterValues' confirmation write is a plain SQL UPDATE with
		// no insert fallback (repository/mysql/cperepository.go
		// UpdateParameter) - seed a baseline row for this LANDevice.* path
		// since bootstrap no longer walks it automatically (see
		// templates_test.go's templateTargetPath tests for the same
		// pattern).
		client.PutDeviceParameter(t, cpe.UUID, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID", "GoACS-Sim", harness.FlagRWS)

		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpspv_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local status = setParameterValues({ [device.root .. ".LANDevice.1.WLANConfiguration.1.SSID"] = "scenario-ssid" })
setParameter(device.root .. ".ManagementServer.ParameterKey", "status:" .. tostring(status), "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})

		ssid, ok := client.FindDeviceParameter(t, cpe.UUID, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID")
		if !ok || ssid.ValueStruct.Value != "scenario-ssid" {
			t.Fatalf("setParameterValues should have confirmed the SSID write against the CPE, got %+v (found=%v)", ssid, ok)
		}
		if _, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); !ok {
			t.Fatalf("expected a status marker parameter to be recorded after the blocking call returned")
		}
	})

	t.Run("setParameterValues_fault_unwrapped_aborts_script", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpspvfault_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
setParameterValues({ [device.root .. ".DeviceInfo.SerialNumber"] = "rejected-write" })
setParameter(device.root .. ".ManagementServer.ParameterKey", "should-not-be-reached", "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		if p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath); ok && p.ValueStruct.Value == "should-not-be-reached" {
			t.Fatalf("writing to a non-writable parameter (DeviceInfo.SerialNumber) should fault and abort the script, but it continued: %+v", p)
		}

		logs := client.GetDeviceLogs(t, cpe.UUID)
		foundFault := false
		for _, l := range logs {
			if l.Type == "FAULT" {
				foundFault = true
			}
		}
		if !foundFault {
			t.Fatalf("expected a fault-type device log entry from the rejected SetParameterValues, got %+v", logs)
		}
	})

	t.Run("safe_wraps_faulting_blocking_call", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "bpsafe_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script: []string{`
local ok, status = safe(setParameterValues, { [device.root .. ".DeviceInfo.SerialNumber"] = "rejected-write" })
log("safe result", tostring(ok))
setParameter(device.root .. ".ManagementServer.ParameterKey", "reached-after-safe", "RW")
`},
		})

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		p, ok := client.FindDeviceParameter(t, cpe.UUID, parameterKeyPath)
		if !ok || p.ValueStruct.Value != "reached-after-safe" {
			t.Fatalf("safe() should catch the fault and let the script continue past it, got %+v (found=%v)", p, ok)
		}

		want := fmt.Sprintf("[script:%s] safe result: false", serial)
		if !strings.Contains(srv.Output(), want) {
			t.Fatalf("expected server log to contain %q (safe() reporting ok=false), got:\n%s", want, srv.Output())
		}
	})
}
