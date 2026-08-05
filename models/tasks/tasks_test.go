package tasks

import (
	"goacs/acs/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsDiagnostics_RoundTripsThroughPayloadValueScan(t *testing.T) {
	task := NewCPETask("cpe-uuid")
	task.AsDiagnostics([]types.ParameterValueStruct{
		{Name: "Device.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Requested", Type: "xsd:string"}},
		{Name: "Device.IP.Diagnostics.DownloadDiagnostics.DownloadURL", ValueStruct: types.ValueStruct{Value: "http://acs.example/speedtest/download", Type: "xsd:string"}},
	})

	assert.Equal(t, RunDiagnostics, task.Task)

	// Simulate the actual DB round-trip: TaskPayload.Value() marshals to JSON for the
	// `payload` column, TaskPayload.Scan() decodes it back on load - this is how a
	// RunDiagnostics task actually reaches the CPE's later check-in, unlike the
	// in-memory-only ParameterValues field.
	raw, err := task.Payload.Value()
	assert.NoError(t, err)

	var reloaded TaskPayload
	assert.NoError(t, reloaded.Scan(raw))

	params, ok := reloaded["parameters"].([]interface{})
	if assert.True(t, ok) && assert.Len(t, params, 2) {
		first, ok := params[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "Device.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState", first["name"])
		assert.Equal(t, "Requested", first["value"])
		assert.Equal(t, "xsd:string", first["type"])
	}
}
