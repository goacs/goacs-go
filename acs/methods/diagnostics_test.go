package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasEventCode(t *testing.T) {
	codes := []string{"1 BOOT", diagnosticsCompleteEventCode}

	assert.True(t, hasEventCode(codes, diagnosticsCompleteEventCode))
	assert.False(t, hasEventCode(codes, "2 PERIODIC"))
	assert.False(t, hasEventCode(nil, diagnosticsCompleteEventCode))
}

func TestDiagnosticsResultParameterInfo_TR181Root(t *testing.T) {
	info := diagnosticsResultParameterInfo("Device")

	names := make([]string, len(info))
	for i, p := range info {
		names[i] = p.Name
	}

	assert.Contains(t, names, "Device.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState")
	assert.Contains(t, names, "Device.IP.Diagnostics.DownloadDiagnostics.TestBytesReceived")
	assert.Contains(t, names, "Device.IP.Diagnostics.UploadDiagnostics.TestBytesSent")
}

func TestDiagnosticsResultParameterInfo_TR098Root(t *testing.T) {
	info := diagnosticsResultParameterInfo("InternetGatewayDevice")

	names := make([]string, len(info))
	for i, p := range info {
		names[i] = p.Name
	}

	// TR-098 has no ".IP." segment - DownloadDiagnostics/UploadDiagnostics hang directly
	// off the InternetGatewayDevice root.
	assert.Contains(t, names, "InternetGatewayDevice.DownloadDiagnostics.DiagnosticsState")
	assert.Contains(t, names, "InternetGatewayDevice.UploadDiagnostics.TestBytesSent")
	assert.NotContains(t, names, "InternetGatewayDevice.IP.Diagnostics.DownloadDiagnostics.DiagnosticsState")
}
