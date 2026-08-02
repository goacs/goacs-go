//go:build scenario

package harness

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	buildClientOnce sync.Once
	clientBinary    string
	clientBuildErr  error
)

func buildClientBinary(t *testing.T) string {
	t.Helper()

	buildClientOnce.Do(func() {
		clientRoot, err := ClientRepoRoot()
		if err != nil {
			clientBuildErr = err
			return
		}

		dir, err := os.MkdirTemp("", "goacs-scenario-client-*")
		if err != nil {
			clientBuildErr = err
			return
		}

		binPath := filepath.Join(dir, "goacs-client-scenario")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/goacs-client")
		cmd.Dir = clientRoot
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			clientBuildErr = fmt.Errorf("building goacs-client binary: %w\n%s", err, out.String())
			return
		}

		clientBinary = binPath
	})

	if clientBuildErr != nil {
		t.Fatalf("harness: %v", clientBuildErr)
	}

	return clientBinary
}

// DeviceOpts configures one `goacs-client device` invocation - one simulated
// CPE session (which may itself span several CWMP request/response
// round-trips, e.g. a Download -> TransferComplete exchange, all within the
// single sess.Run call `device` makes).
type DeviceOpts struct {
	AcsURL      string // required
	Profile     string // profile name, resolved against ProfilesDir then the built-in profiles
	Serial      string // required - reusing the same serial across two DeviceOpts runs simulates two sessions of the same CPE
	Event       string // comma-separated CWMP event code(s), e.g. "0 BOOTSTRAP" or "1 BOOT,4 VALUE CHANGE"; defaults to "0 BOOTSTRAP" if empty
	ProfilesDir string // defaults to testscenarios/profiles
	AuthUser    string
	AuthPass    string
	ConnRequest bool
	Timeout     time.Duration // defaults to 30s
}

// DeviceResult captures one `goacs-client device` invocation's outcome.
type DeviceResult struct {
	Stdout          string
	Err             error
	RebootRequested bool
}

// RunDevice builds (once) the goacs-client binary and runs `goacs-client
// device` with the given options as a foreground subprocess, waiting for it
// to finish (or its Timeout to expire). It does not fail the test on a
// non-zero exit by itself - callers assert on DeviceResult the way they'd
// assert on any other black-box outcome (some scenarios, e.g. an unwrapped
// blocking Lua call hitting a CPE fault, are EXPECTED to end in an error on
// the ACS side while the client session itself still completes normally).
func RunDevice(t *testing.T, opts DeviceOpts) DeviceResult {
	t.Helper()

	if opts.Serial == "" {
		t.Fatalf("harness: DeviceOpts.Serial is required")
	}
	if opts.AcsURL == "" {
		t.Fatalf("harness: DeviceOpts.AcsURL is required")
	}

	binPath := buildClientBinary(t)

	profilesDir := opts.ProfilesDir
	if profilesDir == "" {
		profilesDir = filepath.Join(RepoRoot(), "testscenarios", "profiles")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	args := []string{
		"device",
		"--acs-url", opts.AcsURL,
		"--serial", opts.Serial,
		"--profiles-dir", profilesDir,
		"--verbose=true",
	}
	if opts.Profile != "" {
		args = append(args, "--profile", opts.Profile)
	}
	if opts.Event != "" {
		args = append(args, "--event", opts.Event)
	}
	if opts.AuthUser != "" {
		args = append(args, "--auth-user", opts.AuthUser)
	}
	if opts.AuthPass != "" {
		args = append(args, "--auth-pass", opts.AuthPass)
	}
	if opts.ConnRequest {
		args = append(args, "--conn-request")
	}

	cmd := exec.Command(binPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return DeviceResult{Err: fmt.Errorf("starting goacs-client: %w", err)}
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return DeviceResult{
			Stdout:          out.String(),
			Err:             err,
			RebootRequested: strings.Contains(out.String(), "reboot_requested=true"),
		}
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return DeviceResult{
			Stdout: out.String(),
			Err:    fmt.Errorf("goacs-client device timed out after %s", timeout),
		}
	}
}
