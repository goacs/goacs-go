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

// TestProvisionNowForcesFullWalk exercises the admin panel's "provision now"
// action (GET /api/device/:uuid/provision, http/controllers/device.go
// GetDeviceProvision): it arms a cache flag that forces the device's NEXT
// Inform into a full GetParameterNames/GetParameterValues walk even if that
// Inform would otherwise be a lightweight periodic check-in, then kicks the
// device. This checks the forcing effect directly via the wire-level
// GetParameterNames call appearing in a session that would not normally
// carry one - not the Kick's live delivery, which is already covered by the
// kick() Lua function scenario in scripts_test.go (and goacs-client's
// single-shot `device` command doesn't keep its Connection Request listener
// alive past its own session anyway).
func TestProvisionNowForcesFullWalk(t *testing.T) {
	srv, client := newEnv(t)

	_, profile, profilesDir, serial := scopedRule(t)

	// Session A: bootstrap registers the CPE. Since it's a brand new device, this
	// session runs the seeded "init" Provision (contrib/database/06_init_provision.sql)
	// instead of a full walk - it does not, on its own, produce a
	// "<cwmp:GetParameterNames" wire log entry.
	runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
	cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

	// Baseline: an ordinary periodic Inform on an already-known device does
	// NOT re-walk the whole tree.
	plainPeriodic := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if strings.Contains(plainPeriodic.Stdout, "<cwmp:GetParameterNames") {
		t.Fatalf("a plain periodic Inform on an already-known device should not trigger a full GetParameterNames walk; stdout:\n%s", plainPeriodic.Stdout)
	}

	client.TriggerProvisionNow(t, cpe.UUID)

	forced := runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
	if !strings.Contains(forced.Stdout, "<cwmp:GetParameterNames") {
		t.Fatalf("provision-now should force the next Inform (even a periodic one) into a full GetParameterNames walk; stdout:\n%s", forced.Stdout)
	}
}
