//go:build scenario

// Package harness builds and drives real goacs-go and goacs-client binaries as
// subprocesses so testscenarios/*_test.go can exercise the CWMP engine
// black-box, the same way a real deployment would run. Nothing here imports
// acs/ or http/ - it only talks to the running server over HTTP/REST, and to
// the simulated CPE by shelling out to the goacs-client binary.
package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// RepoRoot returns the goacs-go repository root (the parent of testscenarios/),
// resolved from this file's own path so it works regardless of the test
// binary's working directory.
func RepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("harness: unable to determine source file location")
	}
	// file is <repoRoot>/testscenarios/harness/paths.go
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// ClientRepoRoot returns the goacs-client repository root. It defaults to the
// sibling directory next to goacs-go (matching this workstation's layout: both
// repos live under the same parent directory), overridable via the
// GOACS_CLIENT_DIR environment variable for any other layout.
func ClientRepoRoot() (string, error) {
	if dir := os.Getenv("GOACS_CLIENT_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			return "", fmt.Errorf("harness: GOACS_CLIENT_DIR=%s does not look like a goacs-client checkout: %w", dir, err)
		}
		return dir, nil
	}

	guess := filepath.Join(filepath.Dir(RepoRoot()), "goacs-client")
	if _, err := os.Stat(filepath.Join(guess, "go.mod")); err != nil {
		return "", fmt.Errorf("harness: could not find goacs-client checkout at %s (set GOACS_CLIENT_DIR to override): %w", guess, err)
	}
	return guess, nil
}
