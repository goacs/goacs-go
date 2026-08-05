//go:build scenario

// Covers the admin panel's "lookup now" action (GET /api/device/:uuid/lookup,
// http/controllers/device.go GetDeviceLookup): it arms a cache flag that
// forces the device's NEXT Inform into a full, read-only
// GetParameterNames/GetParameterValues walk (Session.LookupOnly,
// acs/methods/informmethods.go), whose results must reach the
// LookupParamsPrefix cache (readable via GET /api/device/:uuid/parameters/cached)
// but must NEVER be written to cpe_parameters - unlike TriggerProvisionNow's
// forced walk, which is meant to persist.
//
// Also covers the main parameter list's "cached value" column
// (http/controllers/device.go GetDeviceParameters/withCachedValue): a
// cpe_parameters row whose name is also present in the current lookup
// snapshot must carry the value the device actually just reported - letting
// the admin panel compare a possibly-stale stored value against a live one,
// without ever overwriting the stored value itself. Before any lookup has
// run, the same row must carry no cached value at all.
package testscenarios

import (
	"strings"
	"testing"

	"goacs/testscenarios/harness"
)

func TestLookupNowIsReadOnly(t *testing.T) {
	srv, client := newEnv(t)

	_, profile, profilesDir, serial := scopedRule(t)

	// Session A: bootstrap registers the CPE. Since it's a brand new device,
	// this session runs the seeded "init" provision
	// (contrib/database/06_init_provision.sql), which does persist
	// DeviceInfo./ManagementServer. - lookup mode must not additionally
	// persist anything outside that.
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
	cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

	// DeviceInfo.Manufacturer is already a cpe_parameters row (via the init
	// provision's own curated read) - simulate it having gone stale (e.g. an
	// out-of-band change on the CPE the ACS never re-read) by overwriting it
	// with a value the simulated device does NOT actually report. The real
	// value is "GoACS Simulations" (testscenarios/profiles + goacs-client's
	// tr098-router base profile).
	manufacturerPath := "InternetGatewayDevice.DeviceInfo.Manufacturer"
	const staleValue = "OldVendorNameInDB"
	const liveValue = "GoACS Simulations"
	client.PutDeviceParameter(t, cpe.UUID, manufacturerPath, staleValue, harness.Flag{Read: true, Write: true})

	// Before any lookup: no cached value yet - nothing has populated the
	// lookup cache for this device.
	before, ok := findParameterValue(client.GetDeviceParameters(t, cpe.UUID), manufacturerPath)
	if !ok {
		t.Fatalf("expected %q to already be a cpe_parameters row after bootstrap (via the init provision)", manufacturerPath)
	}
	if before.CachedValue != nil {
		t.Fatalf("expected no cached value before any lookup, got %v", *before.CachedValue)
	}

	// Baseline: an ordinary periodic Inform on an already-known device does
	// NOT re-walk the whole tree.
	plainPeriodic := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if strings.Contains(plainPeriodic.Stdout, "<cwmp:GetParameterNames") {
		t.Fatalf("a plain periodic Inform on an already-known device should not trigger a full GetParameterNames walk; stdout:\n%s", plainPeriodic.Stdout)
	}

	client.TriggerLookupNow(t, cpe.UUID)

	looked := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if !strings.Contains(looked.Stdout, "<cwmp:GetParameterNames") {
		t.Fatalf("lookup-now should force the next Inform (even a periodic one) into a full GetParameterNames walk; stdout:\n%s", looked.Stdout)
	}

	cached := client.GetDeviceCachedParameters(t, cpe.UUID)
	if len(cached) == 0 {
		t.Fatalf("expected the lookup walk's results to be readable back from the cache, got none")
	}

	// The walk necessarily reaches parameters outside DeviceInfo./
	// ManagementServer. (e.g. under LANDevice.) - assert at least one such
	// name reached the cache, then assert it did NOT reach cpe_parameters
	// (the durable store), proving the walk stayed read-only.
	var outsideBasic string
	for _, p := range cached {
		if !strings.Contains(p.Name, ".DeviceInfo.") && !strings.Contains(p.Name, ".ManagementServer.") {
			outsideBasic = p.Name
			break
		}
	}
	if outsideBasic == "" {
		t.Fatalf("expected the lookup walk to have read at least one parameter outside DeviceInfo./ManagementServer., cached=%+v", cached)
	}

	if _, ok := client.FindDeviceParameter(t, cpe.UUID, outsideBasic); ok {
		t.Fatalf("lookup mode must never persist parameters to cpe_parameters, but %q was found there", outsideBasic)
	}

	// After the lookup: the same DB row still holds our injected stale value
	// (read-only, confirmed again here) but now also carries the device's
	// real, live value in CachedValue - and the two differ, exactly the
	// drift the "cached value" column exists to surface.
	after, ok := findParameterValue(client.GetDeviceParameters(t, cpe.UUID), manufacturerPath)
	if !ok {
		t.Fatalf("expected %q to still be a cpe_parameters row after the lookup", manufacturerPath)
	}
	if after.ValueStruct.Value != staleValue {
		t.Fatalf("lookup must never overwrite the stored value, expected %q, got %q", staleValue, after.ValueStruct.Value)
	}
	if after.CachedValue == nil || *after.CachedValue != liveValue {
		t.Fatalf("expected the cached value to be the device's real, live value %q, got %+v", liveValue, after.CachedValue)
	}
}

func findParameterValue(parameters []harness.ParameterValue, name string) (harness.ParameterValue, bool) {
	for _, p := range parameters {
		if p.Name == name {
			return p, true
		}
	}
	return harness.ParameterValue{}, false
}
