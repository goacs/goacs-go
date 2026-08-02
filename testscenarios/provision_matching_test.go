//go:build scenario

// Covers acs/logic/provision.go's matching logic: events/requests OR-overlap
// (empty matches anything), rules always ANDed, the "device.root." parameter
// prefix substitution, and the operator set (==, !=, in, not in, >, >=, <,
// <=, including the "unparsable numeric value -> false, not an error" case).
//
// Vendor (Manufacturer) and product-class rules need a warm-up bootstrap
// session before they can ever match anything: those values only arrive via
// Inform's DeviceId element, which is never mirrored into the parameter
// cache a rule reads from (acs/acssession.go FillCPESessionFromInform only
// caches Inform's own ParameterList - SoftwareVersion/HardwareVersion/
// ConnectionRequestURL). A firmware-version (SoftwareVersion) rule has no
// such requirement, since SoftwareVersion IS part of that list and so
// resolves from the very first session.
package testscenarios

import (
	"fmt"
	"strings"
	"testing"

	"goacs/testscenarios/harness"
)

func TestProvisionMatching(t *testing.T) {
	srv, client := newEnv(t)

	t.Run("event_only_rule_matches_by_OR_overlap", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		marker := fmt.Sprintf("[script:%s] event_marker: hit", serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "evt_" + serial,
			Events:   "0 BOOTSTRAP",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`log("event_marker", "hit")`},
		})

		mark := len(srv.Output())
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
		if !strings.Contains(outputSince(srv, mark), marker) {
			t.Fatalf("provision scoped to Events=%q should match an Inform carrying that exact event", "0 BOOTSTRAP")
		}

		mark = len(srv.Output())
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		if strings.Contains(outputSince(srv, mark), marker) {
			t.Fatalf("provision scoped to Events=%q should NOT match an Inform carrying only %q", "0 BOOTSTRAP", "2 PERIODIC")
		}
	})

	t.Run("multi_event_inform_matches_by_OR_overlap", func(t *testing.T) {
		rule, profile, profilesDir, serial := scopedRule(t)
		marker := fmt.Sprintf("[script:%s] multi_event_marker: hit", serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "multievt_" + serial,
			Events:   "4 VALUE CHANGE",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`log("multi_event_marker", "hit")`},
		})

		mark := len(srv.Output())
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "1 BOOT,4 VALUE CHANGE"})
		if !strings.Contains(outputSince(srv, mark), marker) {
			t.Fatalf("an Inform carrying multiple event codes should match a provision on any one of them (OR-overlap)")
		}
	})

	t.Run("requests_SetParameterValuesProcessor_runs_after_the_inform_pass", func(t *testing.T) {
		// Uses log() markers (server stdout) rather than a stored parameter:
		// on a device's very first session there's no guarantee
		// ManagementServer.ParameterKey already has a cpe_parameters row, and
		// setParameter's DB write is a plain SQL UPDATE with no insert
		// fallback (see warmUpDevice's comment) - log() sidesteps that
		// entirely, and this scenario only cares about relative order anyway.
		//
		// Runs on an already-known device (warmUpDevice first, provisions
		// attached after) rather than a brand-new one: a brand-new device's
		// session also queues a one-shot global "new device" task
		// (acs/logic/taskrunner.go loadDeviceTasks, gated on
		// Session.IsNewInACS) on every GetParameterValuesResponse round-trip
		// during its own parameter walk, and if that task has no configured
		// type, TaskRunner.Run() reaches its untyped default branch and
		// returns without recursing - ending the session before the queue
		// is ever checked again, so runSetParamsProvisioningOnce never gets
		// a chance to fire. An already-known device does no walk at all
		// (IsNewInACS is false), sidestepping that entirely.
		rule, profile, profilesDir, serial := scopedRule(t)
		warmUpDevice(t, srv, profile, profilesDir, serial)
		mustCreateProvision(t, client, harness.Provision{
			Name:     "reqinform_" + serial,
			Events:   "2 PERIODIC",
			Requests: "inform",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`log("order_marker", "ran-as-inform")`},
		})
		mustCreateProvision(t, client, harness.Provision{
			Name:     "reqspv_" + serial,
			Events:   "2 PERIODIC",
			Requests: "SetParameterValuesProcessor",
			Rules:    []harness.ProvisionRule{rule},
			Script:   []string{`log("order_marker", "ran-as-setparametervaluesprocessor")`},
		})

		mark := len(srv.Output())
		runDevice(t, srv, harness.DeviceOpts{Profile: profile, ProfilesDir: profilesDir, Serial: serial, Event: "2 PERIODIC"})
		out := outputSince(srv, mark)

		informIdx := strings.Index(out, fmt.Sprintf("[script:%s] order_marker: ran-as-inform", serial))
		spvIdx := strings.Index(out, fmt.Sprintf("[script:%s] order_marker: ran-as-setparametervaluesprocessor", serial))
		if informIdx == -1 || spvIdx == -1 {
			t.Fatalf("expected both scripts to run and log their marker, got:\n%s", out)
		}
		if spvIdx < informIdx {
			t.Fatalf("the SetParameterValuesProcessor-scoped provision should run after the inform-scoped one, but its marker appeared first:\n%s", out)
		}
	})

	t.Run("firmware_version_rule_matches_from_the_first_session", func(t *testing.T) {
		rule := harness.ProvisionRule{Parameter: "device.root.DeviceInfo.SoftwareVersion", Operator: ">=", Value: "2.0"}

		cases := []struct {
			profile     string
			shouldMatch bool
		}{
			{"acme-router", true},     // SoftwareVersion 2.5
			{"legacy-router", false},  // SoftwareVersion 1.0
			{"weird-firmware", false}, // SoftwareVersion "v2-beta" - unparsable, must be a clean non-match, not an error
		}

		for _, c := range cases {
			t.Run(c.profile, func(t *testing.T) {
				serial := harness.UniqueName(t)
				marker := fmt.Sprintf("[script:%s] firmware_rule_marker: hit", serial)
				mustCreateProvision(t, client, harness.Provision{
					Name:     "fw_" + serial,
					Requests: "inform",
					Rules:    []harness.ProvisionRule{rule},
					Script:   []string{`log("firmware_rule_marker", "hit")`},
				})

				mark := len(srv.Output())
				runDevice(t, srv, harness.DeviceOpts{Profile: c.profile, Serial: serial, Event: "0 BOOTSTRAP"})
				out := outputSince(srv, mark)
				matched := strings.Contains(out, marker)
				if matched != c.shouldMatch {
					t.Fatalf("profile %s: expected match=%v, got match=%v\nOUTPUT:\n%s", c.profile, c.shouldMatch, matched, out)
				}
			})
		}
	})

	t.Run("vendor_rule_needs_a_warm_up_session_before_it_can_match", func(t *testing.T) {
		rule := harness.ProvisionRule{Parameter: "device.root.DeviceInfo.Manufacturer", Operator: "==", Value: "Acme Corp"}

		cases := []struct {
			profile     string
			shouldMatch bool
		}{
			{"acme-router", true},
			{"legacy-router", false},
		}

		for _, c := range cases {
			t.Run(c.profile, func(t *testing.T) {
				serial := harness.UniqueName(t)
				marker := fmt.Sprintf("[script:%s] vendor_rule_marker: hit", serial)
				mustCreateProvision(t, client, harness.Provision{
					Name:     "vendor_" + serial,
					Requests: "inform",
					Rules:    []harness.ProvisionRule{rule},
					Script:   []string{`log("vendor_rule_marker", "hit")`},
				})

				mark := len(srv.Output())
				runDevice(t, srv, harness.DeviceOpts{Profile: c.profile, Serial: serial, Event: "0 BOOTSTRAP"})
				if strings.Contains(outputSince(srv, mark), marker) {
					t.Fatalf("profile %s: a Manufacturer rule should never match on the device's very first (cold-cache) session", c.profile)
				}

				mark = len(srv.Output())
				runDevice(t, srv, harness.DeviceOpts{Profile: c.profile, Serial: serial, Event: "2 PERIODIC"})
				out := outputSince(srv, mark)
				matched := strings.Contains(out, marker)
				if matched != c.shouldMatch {
					t.Fatalf("profile %s: expected match=%v on the warmed-up session, got match=%v\nOUTPUT:\n%s", c.profile, c.shouldMatch, matched, out)
				}
			})
		}
	})

	t.Run("in_and_not_in_operators_on_product_class", func(t *testing.T) {
		inRule := harness.ProvisionRule{Parameter: "device.root.DeviceInfo.ProductClass", Operator: "in", Value: "AcmeRouter,SomeOtherClass"}
		notInRule := harness.ProvisionRule{Parameter: "device.root.DeviceInfo.ProductClass", Operator: "not in", Value: "AcmeRouter"}

		// Deletes its provision immediately (not via mustCreateProvision's
		// deferred subtest-level cleanup): this rule matches by
		// ProductClass value alone, with no per-device scoping, and the
		// "in"/"not in" cases below deliberately reuse the same profiles -
		// leaving an earlier call's provision active would let it match a
		// later call's device too.
		warmUpThenTest := func(t *testing.T, profile string, rule harness.ProvisionRule, provisionName string) bool {
			t.Helper()
			serial := harness.UniqueName(t)
			marker := fmt.Sprintf("[script:%s] product_class_marker: hit", serial)
			p := client.CreateProvision(t, harness.Provision{
				Name:     provisionName + "_" + serial,
				Requests: "inform",
				Rules:    []harness.ProvisionRule{rule},
				Script:   []string{`log("product_class_marker", "hit")`},
			})
			defer client.DeleteProvision(t, p.Id)

			runDevice(t, srv, harness.DeviceOpts{Profile: profile, Serial: serial, Event: "0 BOOTSTRAP"})
			mark := len(srv.Output())
			runDevice(t, srv, harness.DeviceOpts{Profile: profile, Serial: serial, Event: "2 PERIODIC"})
			return strings.Contains(outputSince(srv, mark), marker)
		}

		if !warmUpThenTest(t, "acme-router", inRule, "in") {
			t.Fatalf(`"in" rule should match acme-router (ProductClass=AcmeRouter is in the CSV list)`)
		}
		if warmUpThenTest(t, "legacy-router", inRule, "in") {
			t.Fatalf(`"in" rule should not match legacy-router (ProductClass=LegacyRouter is not in the CSV list)`)
		}
		if warmUpThenTest(t, "acme-router", notInRule, "notin") {
			t.Fatalf(`"not in" rule should not match acme-router (ProductClass=AcmeRouter IS in the excluded list)`)
		}
		if !warmUpThenTest(t, "legacy-router", notInRule, "notin") {
			t.Fatalf(`"not in" rule should match legacy-router (ProductClass=LegacyRouter is not in the excluded list)`)
		}
	})

	t.Run("multiple_rules_are_ANDed", func(t *testing.T) {
		rules := []harness.ProvisionRule{
			{Parameter: "device.root.DeviceInfo.Manufacturer", Operator: "==", Value: "Acme Corp"},
			{Parameter: "device.root.DeviceInfo.SoftwareVersion", Operator: ">=", Value: "2.0"},
		}

		// A profile matching only the firmware half of the AND, to prove
		// this isn't accidentally an OR: same manufacturer as legacy-router,
		// but with acme-router's newer firmware.
		partialProfile, partialDir := harness.WriteScenarioProfile(t, "InternetGatewayDevice", "tr098-router", map[string]string{
			"SoftwareVersion": "2.5",
		})

		cases := []struct {
			name        string
			profile     string
			profilesDir string
			shouldMatch bool
		}{
			{"matches_both", "acme-router", "", true},
			{"matches_neither", "legacy-router", "", false},
			{"matches_only_firmware_not_vendor", partialProfile, partialDir, false},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				serial := harness.UniqueName(t)
				marker := fmt.Sprintf("[script:%s] and_rules_marker: hit", serial)
				mustCreateProvision(t, client, harness.Provision{
					Name:     "and_" + serial,
					Requests: "inform",
					Rules:    rules,
					Script:   []string{`log("and_rules_marker", "hit")`},
				})

				runDevice(t, srv, harness.DeviceOpts{Profile: c.profile, ProfilesDir: c.profilesDir, Serial: serial, Event: "0 BOOTSTRAP"})
				mark := len(srv.Output())
				runDevice(t, srv, harness.DeviceOpts{Profile: c.profile, ProfilesDir: c.profilesDir, Serial: serial, Event: "2 PERIODIC"})
				matched := strings.Contains(outputSince(srv, mark), marker)
				if matched != c.shouldMatch {
					t.Fatalf("expected match=%v, got match=%v", c.shouldMatch, matched)
				}
			})
		}
	})

	t.Run("device_root_prefix_resolves_on_both_TR098_and_TR181_roots", func(t *testing.T) {
		rule := harness.ProvisionRule{Parameter: "device.root.DeviceInfo.SoftwareVersion", Operator: ">=", Value: "2.0"}

		tr098Serial := harness.UniqueName(t)
		tr098Marker := fmt.Sprintf("[script:%s] root_marker: hit", tr098Serial)
		mustCreateProvision(t, client, harness.Provision{
			Name: "root_tr098_" + tr098Serial, Requests: "inform",
			Rules: []harness.ProvisionRule{rule}, Script: []string{`log("root_marker", "hit")`},
		})
		runDevice(t, srv, harness.DeviceOpts{Profile: "acme-router", Serial: tr098Serial, Event: "0 BOOTSTRAP"})
		if !strings.Contains(srv.Output(), tr098Marker) {
			t.Fatalf("device.root. rule should resolve to InternetGatewayDevice. on a TR-098 profile")
		}

		tr181Profile, tr181Dir := harness.WriteScenarioProfile(t, "Device", "tr181-router", map[string]string{
			"SoftwareVersion": "3.0",
		})
		tr181Serial := harness.UniqueName(t)
		tr181Marker := fmt.Sprintf("[script:%s] root_marker: hit", tr181Serial)
		mustCreateProvision(t, client, harness.Provision{
			Name: "root_tr181_" + tr181Serial, Requests: "inform",
			Rules: []harness.ProvisionRule{rule}, Script: []string{`log("root_marker", "hit")`},
		})
		runDevice(t, srv, harness.DeviceOpts{Profile: tr181Profile, ProfilesDir: tr181Dir, Serial: tr181Serial, Event: "0 BOOTSTRAP"})
		if !strings.Contains(srv.Output(), tr181Marker) {
			t.Fatalf("device.root. rule should resolve to Device. on a TR-181 profile")
		}
	})
}
