//go:build scenario

package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

var uniqueCounter int64

// UniqueName returns a short, opaque, process-unique token ("sc1", "sc2",
// ...). It deliberately does NOT embed t.Name(): this value is also used as
// the CWMP serial number and, in scopedRule, as a DeviceInfo.HardwareVersion
// selector - and cpe.hardware_version/cpe.software_version are only
// varchar(20) (contrib/database/01_initial.sql), so anything derived from a
// full (sub)test name plus a timestamp overflows it and gets silently
// dropped on insert. A failing scenario is still traceable through the
// standard "--- FAIL: TestFoo/subtest" header Go's test output already
// prints, so nothing is lost by keeping this short.
func UniqueName(t *testing.T) string {
	t.Helper()
	n := atomic.AddInt64(&uniqueCounter, 1)
	return "sc" + strconv.FormatInt(n, 36)
}

// WriteScenarioProfile generates a temporary, self-contained goacs-client
// profile YAML that includes a built-in base profile (e.g. "tr098-router" or
// "tr181-router") and applies the given DeviceInfo.* overrides on top,
// returning the profile name and the directory to pass as --profiles-dir.
//
// Overrides are plain DeviceInfo leaf names (e.g. "SoftwareVersion",
// "HardwareVersion", "Manufacturer", "ProductClass") - this helper always
// prefixes them with "DeviceInfo.".
//
// HardwareVersion (unlike Manufacturer/ProductClass/SerialNumber) is one of
// the handful of parameters goacs-client's Inform always carries in its own
// ParameterList (see internal/session/session.go informBootstrapParams), and
// is therefore resolvable by a provision rule from the CPE's very first
// session. Scenarios that need to scope a provision rule to exactly one
// simulated device without a prior warm-up session should key their rule off
// a unique HardwareVersion value set here, rather than SerialNumber or
// Manufacturer (which only become resolvable via the DB after an earlier
// session's full parameter walk - see provision_matching_test.go for a
// scenario that exercises that distinction directly).
func WriteScenarioProfile(t *testing.T, root, baseProfile string, overrides map[string]string) (profileName, profilesDir string) {
	t.Helper()

	dir := t.TempDir()
	name := UniqueName(t)

	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", name)
	fmt.Fprintf(&b, "root: %s\n", root)
	fmt.Fprintf(&b, "includes:\n  - %s\n", baseProfile)
	b.WriteString("parameters:\n")
	for path, value := range overrides {
		fmt.Fprintf(&b, "  - path: DeviceInfo.%s\n", path)
		fmt.Fprintf(&b, "    value: %q\n", value)
		b.WriteString("    type: xsd:string\n")
		b.WriteString("    writable: false\n")
	}

	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("harness: writing scenario profile: %v", err)
	}

	return name, dir
}
