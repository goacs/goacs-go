package controllers

import (
	"goacs/lib"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveStoragePath_Valid(t *testing.T) {
	env := lib.Env{}
	full, safe, err := resolveStoragePath(env, "firmware-v1.bin")

	assert.NoError(t, err)
	assert.Equal(t, "firmware-v1.bin", safe)

	storageAbs, _ := filepath.Abs("./storage")
	assert.Equal(t, filepath.Join(storageAbs, "firmware-v1.bin"), full)
}

func TestResolveStoragePath_RejectsTraversal(t *testing.T) {
	env := lib.Env{}
	cases := []string{
		"../../etc/passwd",
		"..",
		"foo/../../bar",
		"/etc/passwd",
		"a/b",
		"a\\b",
		"",
	}

	for _, name := range cases {
		_, _, err := resolveStoragePath(env, name)
		assert.Error(t, err, "expected error for filename %q", name)
	}
}
