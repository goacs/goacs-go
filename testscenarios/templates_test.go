//go:build scenario

// Templates are applied by value-diffing (acs/methods/parametermethods.go
// PrepareParametersToSend), which runs after RunScript regardless of
// whether that script used a blocking call or not (acs/logic/taskrunner.go
// and acs/logic/dispatcher.go's scripts.Resume path both trigger it), and
// only ever diffs a parameter NAME already present in the session's live
// parameter cache (models/cpe/cpe.go) - Inform's own ParameterList only
// carries SoftwareVersion/HardwareVersion/ConnectionRequestURL, so an
// arbitrary parameter like LANDevice.1.WLANConfiguration.1.Channel is
// normally missing from that cache entirely.
//
// noOpScriptProvision's script bridges that gap with a single blocking
// getParameterValues() call on the target path: it fetches the CPE's real,
// live value into the cache (a genuine round-trip, no local guesswork) and
// does nothing else - the value it then pushes comes entirely from stored
// cpe_parameters/templates, never from this script's own logic. Each
// `goacs-client device` invocation is its own fresh process with no memory
// of a previous one, so that live value is always the profile's plain
// default (Channel=6) unless a previous session in THIS test already pushed
// something different to cpe_parameters.
package testscenarios

import (
	"fmt"
	"testing"

	"goacs/testscenarios/harness"
)

const templateTargetPath = "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"

func noOpScriptProvision(t *testing.T, client *harness.Client, name string, rule harness.ProvisionRule) {
	t.Helper()
	mustCreateProvision(t, client, harness.Provision{
		Name:     name,
		Requests: "inform",
		Rules:    []harness.ProvisionRule{rule},
		Script:   []string{fmt.Sprintf(`getParameterValues(%q)`, templateTargetPath)},
	})
}

func TestTemplates(t *testing.T) {
	srv, client := newEnv(t)

	t.Run("template_alone_without_a_matching_provision_never_pushes", func(t *testing.T) {
		_, profile, profilesDir, serial := scopedRule(t)
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		// Bootstrap alone no longer reports this LANDevice.* path into
		// cpe_parameters (acs/methods/informmethods.go only auto-walks
		// already-known devices now; a new device instead runs the much
		// narrower seeded "init" provision - see
		// contrib/database/06_init_provision.sql), so seed the CPE's real
		// reported default directly, matching what the old full walk would
		// have captured.
		client.PutDeviceParameter(t, cpe.UUID, templateTargetPath, "6", harness.Flag{Read: true, Write: true})

		tpl := client.CreateAndFindTemplate(t, "tpl_alone_"+serial)
		client.StoreTemplateParameter(t, tpl.Id, templateTargetPath, "should-never-appear", harness.FlagRWS)
		client.AssignTemplateToDevice(t, cpe.UUID, tpl.Id, 150)

		// No provision exists at all, so no RunScript task ever runs for
		// this device - assigning a template by itself must not push
		// anything.
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})

		ch, ok := client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
		if !ok || ch.ValueStruct.Value != "6" {
			t.Fatalf("assigning a template with no matching provision/script should not push anything, got %+v (found=%v)", ch, ok)
		}
	})

	t.Run("template_with_script_pushes_then_unassign_stops_it", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		tpl := client.CreateAndFindTemplate(t, "tpl_push_"+serial)
		client.StoreTemplateParameter(t, tpl.Id, templateTargetPath, "template-value", harness.FlagRWS)
		client.AssignTemplateToDevice(t, cpe.UUID, tpl.Id, 150)
		noOpScriptProvision(t, client, "tplpush_provision_"+serial, rule)

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		ch, ok := client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
		if !ok || ch.ValueStruct.Value != "template-value" {
			t.Fatalf("a script fetching the target parameter should still trigger the template diff push, got %+v (found=%v)", ch, ok)
		}

		// Now unassign the template and directly set a different desired
		// value via the plain REST parameters endpoint (no script, no
		// template). The next session's device process is fresh (reports
		// Channel=6 again, same as always), so if the template still
		// applied it would force "template-value" again regardless of what
		// was just stored; getting the manually-stored value instead
		// confirms unassigning genuinely stopped enforcement.
		client.UnassignTemplateFromDevice(t, cpe.UUID, tpl.Id)
		client.PutDeviceParameter(t, cpe.UUID, templateTargetPath, "manually-set-after-unassign", harness.FlagRWS)

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		ch, ok = client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
		if !ok || ch.ValueStruct.Value != "manually-set-after-unassign" {
			t.Fatalf("after unassigning, the template should no longer override a manually stored value, got %+v (found=%v)", ch, ok)
		}
	})

	t.Run("higher_priority_template_wins", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		// See the comment in "template_alone_without_a_matching_provision_never_pushes":
		// this LANDevice.* path needs a seeded baseline row since bootstrap no
		// longer walks it automatically.
		client.PutDeviceParameter(t, cpe.UUID, templateTargetPath, "6", harness.Flag{Read: true, Write: true})

		low := client.CreateAndFindTemplate(t, "tpl_low_"+serial)
		client.StoreTemplateParameter(t, low.Id, templateTargetPath, "low-priority-value", harness.FlagRWS)
		client.AssignTemplateToDevice(t, cpe.UUID, low.Id, 50)

		high := client.CreateAndFindTemplate(t, "tpl_high_"+serial)
		client.StoreTemplateParameter(t, high.Id, templateTargetPath, "high-priority-value", harness.FlagRWS)
		client.AssignTemplateToDevice(t, cpe.UUID, high.Id, 150)

		noOpScriptProvision(t, client, "tplpriority_provision_"+serial, rule)
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})

		ch, ok := client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
		if !ok || ch.ValueStruct.Value != "high-priority-value" {
			t.Fatalf("only the priority>100 template should win an overlapping parameter name, got %+v (found=%v)", ch, ok)
		}
	})

	t.Run("template_parameter_with_write_flag_false_is_never_pushed", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

		// See the comment in "template_alone_without_a_matching_provision_never_pushes":
		// this LANDevice.* path needs a seeded baseline row since bootstrap no
		// longer walks it automatically.
		client.PutDeviceParameter(t, cpe.UUID, templateTargetPath, "6", harness.Flag{Read: true, Write: true})

		tpl := client.CreateAndFindTemplate(t, "tpl_readonly_"+serial)
		client.StoreTemplateParameter(t, tpl.Id, templateTargetPath, "should-not-apply", harness.Flag{Read: true})
		client.AssignTemplateToDevice(t, cpe.UUID, tpl.Id, 150)
		noOpScriptProvision(t, client, "tplreadonly_provision_"+serial, rule)

		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})

		// The diff never fires for this parameter (Flag.Write=false on the
		// template side), so the device's own reported value (6) survives
		// untouched.
		ch, ok := client.FindDeviceParameter(t, cpe.UUID, templateTargetPath)
		if !ok || ch.ValueStruct.Value != "6" {
			t.Fatalf("a template parameter with Flag.Write=false should never enter the diff, got %+v (found=%v)", ch, ok)
		}
	})
}
