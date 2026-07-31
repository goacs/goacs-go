package provisions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptList_ValueAndScanRoundTrip(t *testing.T) {
	original := ScriptList{"println(1)", "println(2)"}

	value, err := original.Value()
	assert.NoError(t, err)

	var scanned ScriptList
	assert.NoError(t, scanned.Scan(value))
	assert.Equal(t, original, scanned)
}

func TestScriptList_ScanEmpty(t *testing.T) {
	var scanned ScriptList
	assert.NoError(t, scanned.Scan([]byte{}))
	assert.Nil(t, scanned)

	assert.NoError(t, scanned.Scan(nil))
	assert.Nil(t, scanned)
}

func TestProvision_EventsAndRequestsList(t *testing.T) {
	p := Provision{Events: " 0 BOOTSTRAP , 1 BOOT ", Requests: "Inform"}

	assert.Equal(t, []string{"0 BOOTSTRAP", "1 BOOT"}, p.EventsList())
	assert.Equal(t, []string{"Inform"}, p.RequestsList())

	empty := Provision{}
	assert.Nil(t, empty.EventsList())
	assert.Nil(t, empty.RequestsList())
}
