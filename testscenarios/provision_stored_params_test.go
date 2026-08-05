//go:build scenario

// "Provisioning from already-saved parameters, without a script" means: the
// desired values come entirely from cpe_parameters (set directly through the
// REST API, no Lua business logic involved), but a provision still has to
// exist and match with a script - PrepareParametersToSend (the code that
// diffs stored/template values against what the CPE just reported and
// pushes the difference, acs/methods/parametermethods.go) only ever runs
// right after a RunScript task completes. See templates_test.go's package
// comment for why that script's only job is a single blocking
// getParameterValues() fetch of the target parameter.
//
// A stored parameter with no matching provision at all never gets pushed,
// no matter how long the device keeps checking in - checked here via the
// wire-level request log rather than the parameter's own stored value,
// since a direct REST write already makes GetDeviceParameters report that
// value regardless of whether anything ever pushes it to the device.
package testscenarios

import (
	"fmt"
	"strings"
	"testing"

	"goacs/testscenarios/harness"
)

func TestProvisionFromStoredParameters(t *testing.T) {
	srv, client := newEnv(t)

	rule, profile, profilesDir, serial := scopedRule(t)

	// Session A: plain bootstrap with no provision at all yet - just gets the
	// CPE registered so its uuid exists for the REST parameter write below.
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
	cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

	client.PutDeviceParameter(t, cpe.UUID, templateTargetPath, "stored-value-no-script", harness.FlagRWS)

	// Session B: still no provision matches this device - a stored value by
	// itself must never be sent to the CPE.
	noPush := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if strings.Contains(noPush.Stdout, "stored-value-no-script") {
		t.Fatalf("a stored parameter with no matching provision should never be sent to the CPE; stdout:\n%s", noPush.Stdout)
	}

	// Now attach a provision whose script's only job is to fetch the target
	// parameter (a real round-trip, populating the live cache with the
	// CPE's actual value) - the value it then pushes comes entirely from
	// the stored cpe_parameters row above, not from anything the script
	// itself decides.
	noOpScriptProvision(t, client, "stored_params_provision_"+serial, rule)

	pushed := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if !strings.Contains(pushed.Stdout, "stored-value-no-script") {
		t.Fatalf("a provision whose script only fetches the target parameter should be enough to push an already-stored value; stdout:\n%s", pushed.Stdout)
	}
	ch, ok := client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
	if !ok || ch.ValueStruct.Value != "stored-value-no-script" {
		t.Fatalf("expected the pushed value to be confirmed back into cpe_parameters, got %+v (found=%v)", ch, ok)
	}
}

// TestProvisionNowForcesBootProvisioning exercises the admin panel's "provision now"
// action (GET /api/device/:uuid/provision, http/controllers/device.go
// GetDeviceProvision): it forces the device's NEXT Inform to be treated like a boot
// for provisioning rule matching (acs/methods/informmethods.go,
// Session.EnsureEventCode adding "1 BOOT" to the matched event codes) even if that
// Inform would otherwise be a lightweight periodic check-in, then kicks the device.
// There is no separate ACS-side parameter walk anymore - discovery is entirely up to
// whatever provision matches, so this checks the forcing effect through the seeded
// "init" provision (contrib/database/06_init_provision.sql, scoped to
// "0 BOOTSTRAP,1 BOOT") re-running its own script and logging its marker again on a
// session that would not normally match it - not the Kick's live delivery, which is
// already covered by the kick() Lua function scenario in scripts_test.go (and
// goacs-client's single-shot `device` command doesn't keep its Connection Request
// listener alive past its own session anyway).
func TestProvisionNowForcesBootProvisioning(t *testing.T) {
	srv, client := newEnv(t)

	_, profile, profilesDir, serial := scopedRule(t)
	initMarker := fmt.Sprintf("[script:%s] init: basic parameters read and saved", serial)

	// Session A: bootstrap registers the CPE and runs the seeded "init" provision once.
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
	cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

	// Baseline: an ordinary periodic Inform on an already-known device does NOT
	// re-match the boot-scoped "init" provision.
	mark := len(srv.Output())
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if strings.Contains(outputSince(srv, mark), initMarker) {
		t.Fatalf("a plain periodic Inform on an already-known device should not re-run a boot-scoped provision; server output:\n%s", outputSince(srv, mark))
	}

	client.TriggerProvisionNow(t, cpe.UUID)

	mark = len(srv.Output())
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if !strings.Contains(outputSince(srv, mark), initMarker) {
		t.Fatalf("provision-now should force the next Inform (even a periodic one) to match and re-run the boot-scoped init provision; server output:\n%s", outputSince(srv, mark))
	}
}
