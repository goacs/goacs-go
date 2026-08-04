//go:build scenario

// Covers the "init" provision seeded by contrib/database/06_init_provision.sql:
// a brand-new device's very first Inform must no longer trigger the old
// recursive, multi-level GetParameterNames/GetParameterValues walk of the whole
// device model (acs/methods/informmethods.go), and instead run the curated "init"
// script, which reads only DeviceInfo. and ManagementServer. via the blocking
// getParameterValues() bridge function and persists them via saveDevice().
//
// getParameterValues() itself now precedes each of its arguments with one bounded
// GetParameterNames(path, NextLevel=false) round-trip (acs/scripts/bridge.go) to learn
// the Writable flag for what it's about to read - this is NOT the old walk (which
// recursed level-by-level with NextLevel=true across the entire tree): it's exactly one
// round-trip per argument passed to getParameterValues(), here 2 for the seeded
// script's two curated paths.
package testscenarios

import (
	"fmt"
	"strings"
	"testing"

	"goacs/testscenarios/harness"
)

func TestInitProvisionCuratedDiscovery(t *testing.T) {
	srv, client := newEnv(t)

	serial := harness.UniqueName(t)
	marker := fmt.Sprintf("[script:%s] init: basic parameters read and saved", serial)

	mark := len(srv.Output())
	res := runDevice(t, srv, harness.DeviceOpts{Profile: "acme-router", Serial: serial, Event: "0 BOOTSTRAP"})

	if strings.Contains(res.Stdout, "<NextLevel>true</NextLevel>") {
		t.Fatalf("a brand-new device's first Inform should no longer trigger the old recursive, multi-level GetParameterNames walk; stdout:\n%s", res.Stdout)
	}
	if got := strings.Count(res.Stdout, "<cwmp:GetParameterNames>"); got != 2 {
		t.Fatalf("expected exactly one curated GetParameterNames(NextLevel=false) round-trip per getParameterValues() argument in the seeded init script (DeviceInfo., ManagementServer.), got %d; stdout:\n%s", got, res.Stdout)
	}
	if !strings.Contains(res.Stdout, "<cwmp:GetParameterValues") {
		t.Fatalf("expected the seeded init provision's blocking getParameterValues() call to issue a GetParameterValues request; stdout:\n%s", res.Stdout)
	}
	if !strings.Contains(outputSince(srv, mark), marker) {
		t.Fatalf("expected the init provision's script to run to completion and log its marker; server output:\n%s", outputSince(srv, mark))
	}

	cpe := client.FindDeviceBySerial(t, serial, findDeviceTimeout)

	// Per testscenarios/README.md, DeviceInfo.Manufacturer/ProductClass normally
	// need "a prior session's full parameter walk" before they ever reach
	// cpe_parameters - the whole point of the init provision is that its own
	// curated read now does that job, in this very first session, without a
	// full walk.
	manufacturer, ok := client.FindDeviceParameter(t, cpe.UUID, "InternetGatewayDevice.DeviceInfo.Manufacturer")
	if !ok || manufacturer.ValueStruct.Value != "Acme Corp" {
		t.Fatalf("expected DeviceInfo.Manufacturer to be persisted after just the init provision's curated read, got %+v (found=%v)", manufacturer, ok)
	}

	productClass, ok := client.FindDeviceParameter(t, cpe.UUID, "InternetGatewayDevice.DeviceInfo.ProductClass")
	if !ok || productClass.ValueStruct.Value != "AcmeRouter" {
		t.Fatalf("expected DeviceInfo.ProductClass to be persisted after just the init provision's curated read, got %+v (found=%v)", productClass, ok)
	}
}
