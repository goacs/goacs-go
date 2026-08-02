//go:build scenario

// Package testscenarios holds black-box scenario tests that drive a real
// goacs-go server with the goacs-client CPE simulator and assert results
// through goacs-go's own REST API - see testscenarios/README.md.
package testscenarios

import (
	"testing"
	"time"

	"goacs/testscenarios/harness"
)

const (
	adminUser = "admin"
	adminPass = "admin"

	// findDeviceTimeout bounds how long FindDeviceBySerial waits for a CPE
	// row to appear after RunDevice returns - in practice this is
	// near-instant since the Inform that creates it is already fully
	// processed by the time the goacs-client process exits.
	findDeviceTimeout = 10 * time.Second
)

// newEnv starts a fresh goacs-go server on its own disposable database and
// returns a logged-in REST client for it. Call once per top-level Test
// function; every t.Run subtest under it shares the same server, scoping
// itself to its own unique serial/provision/template names.
//
// The migrated database seeds a demo provision ("Multiplay WiFi + ACS
// credentials", contrib/database/04_multiplay_wifi_provision.sql) that
// matches every device's every bootstrap/boot session and uses several
// blocking Lua calls itself - left in place, it adds its own unpredictable
// extra round-trips to sessions these scenarios depend on behaving
// deterministically, so it's removed up front.
func newEnv(t *testing.T) (*harness.Server, *harness.Client) {
	t.Helper()
	srv := harness.StartServer(t)
	client := harness.NewClient(srv.BaseURL)
	client.Login(t, adminUser, adminPass)
	client.DeleteProvisionsNamed(t, "Multiplay WiFi + ACS credentials")
	return srv, client
}

// mustCreateProvision creates a provision and registers its cleanup, so it
// stops matching future sessions on the shared server once this subtest
// ends. Provisions are the one resource that's genuinely global (matched
// against every session that satisfies its events/requests/rules), so every
// scenario that creates one must clean it up.
func mustCreateProvision(t *testing.T, client *harness.Client, p harness.Provision) harness.Provision {
	t.Helper()
	created := client.CreateProvision(t, p)
	t.Cleanup(func() { client.DeleteProvision(t, created.Id) })
	return created
}

// runDevice runs one goacs-client `device` session and fails the test if the
// process itself errored (a CPE Fault from the ACS is not this - the
// simulator handles that gracefully and still exits cleanly).
func runDevice(t *testing.T, srv *harness.Server, opts harness.DeviceOpts) harness.DeviceResult {
	t.Helper()
	if opts.AcsURL == "" {
		opts.AcsURL = srv.BaseURL + "/acs"
	}
	res := harness.RunDevice(t, opts)
	if res.Err != nil {
		t.Fatalf("goacs-client device session failed: %v\n--- stdout ---\n%s", res.Err, res.Stdout)
	}
	return res
}

// outputSince returns everything the server has printed to stdout/stderr
// after the given length mark (take mark from len(srv.Output()) before the
// session you want to isolate). This lets a "this provision should NOT
// match" assertion check for a fresh occurrence of a log marker without a
// false positive from an earlier session/subtest that already logged the
// same text.
func outputSince(srv *harness.Server, mark int) string {
	out := srv.Output()
	if mark > len(out) {
		return ""
	}
	return out[mark:]
}

// warmUpDevice runs a plain bootstrap session for a brand-new device. Any
// provision that should NOT fire during this warm-up must be scoped to a
// different event (e.g. Events: "2 PERIODIC") than "0 BOOTSTRAP"/"1 BOOT".
//
// The seeded "init" provision (contrib/database/06_init_provision.sql) makes
// this session populate DeviceInfo.*/ManagementServer.* in cpe_parameters via
// a blocking getParameterValues() + saveDevice() - see
// acs/methods/informmethods.go, which no longer auto-walks brand-new devices
// itself. That covers most of what setParameter/setParameterValues tests
// need before asserting a persisted write (both are a plain SQL UPDATE with
// no insert fallback - repository/mysql/cperepository.go UpdateParameter,
// called from acs/scripts/functions.go and acs/scripts/bridge.go - so a
// write to a parameter name with no existing cpe_parameters row is a silent
// no-op). A test that targets a parameter OUTSIDE DeviceInfo./
// ManagementServer. (e.g. LANDevice./WLANConfiguration.) still needs to seed
// that specific row itself first, e.g. via client.PutDeviceParameter - see
// templates_test.go's templateTargetPath tests for that pattern.
func warmUpDevice(t *testing.T, srv *harness.Server, profile, profilesDir, serial string) {
	t.Helper()
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
}

// scopedRule builds a Rule + matching profile that scopes a provision to
// exactly one simulated device via a unique DeviceInfo.HardwareVersion value
// - see harness.WriteScenarioProfile for why HardwareVersion specifically.
func scopedRule(t *testing.T) (rule harness.ProvisionRule, profileName, profilesDir, serial string) {
	t.Helper()
	selector := harness.UniqueName(t)
	profileName, profilesDir = harness.WriteScenarioProfile(t, "InternetGatewayDevice", "tr098-router", map[string]string{
		"HardwareVersion": selector,
	})
	rule = harness.ProvisionRule{Parameter: "device.root.DeviceInfo.HardwareVersion", Operator: "==", Value: selector}
	return rule, profileName, profilesDir, selector
}
