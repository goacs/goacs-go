package controllers

import (
	"goacs/acs/types"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiagnosticsPrefix(t *testing.T) {
	assert.Equal(t, "Device.IP.Diagnostics.", diagnosticsPrefix("Device"))
	assert.Equal(t, "InternetGatewayDevice.", diagnosticsPrefix("InternetGatewayDevice"))
}

func TestSelfHostedSpeedtestURL(t *testing.T) {
	req := httptest.NewRequest("GET", "http://acs.example/anything", nil)

	assert.Equal(t, "http://acs.example/speedtest/upload", selfHostedSpeedtestURL(req, "speedtest/upload", nil))

	query := url.Values{"bytes": []string{"1048576"}}
	assert.Equal(t, "http://acs.example/speedtest/download?bytes=1048576", selfHostedSpeedtestURL(req, "speedtest/download", query))
}

func TestBuildDiagnosticsResult_ComputesThroughputForCompleteRecentRun(t *testing.T) {
	eom := time.Now().Add(-time.Minute)
	bom := eom.Add(-10 * time.Second)
	prefix := "Device.IP.Diagnostics.DownloadDiagnostics."

	parameters := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Complete"}},
		{Name: prefix + "BOMTime", ValueStruct: types.ValueStruct{Value: bom.Format(time.RFC3339)}},
		{Name: prefix + "EOMTime", ValueStruct: types.ValueStruct{Value: eom.Format(time.RFC3339)}},
		{Name: prefix + "TestBytesReceived", ValueStruct: types.ValueStruct{Value: "125000000"}},
	}

	result := buildDiagnosticsResult(parameters, prefix, "TestBytesReceived")

	if assert.NotNil(t, result) {
		assert.Equal(t, "Complete", result.State)
		assert.Equal(t, int64(125000000), result.Bytes)
		assert.InDelta(t, 10.0, result.DurationSeconds, 0.01)
		// 125,000,000 bytes * 8 / 10s / 1e6 = 100 Mbps
		assert.InDelta(t, 100.0, result.ThroughputMbps, 0.5)
	}
}

func TestBuildDiagnosticsResult_ExcludesIncompleteState(t *testing.T) {
	prefix := "Device.IP.Diagnostics.DownloadDiagnostics."
	parameters := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Requested"}},
	}

	assert.Nil(t, buildDiagnosticsResult(parameters, prefix, "TestBytesReceived"))
}

func TestBuildDiagnosticsResult_ExcludesResultOlderThan24h(t *testing.T) {
	eom := time.Now().Add(-25 * time.Hour)
	bom := eom.Add(-10 * time.Second)
	prefix := "Device.IP.Diagnostics.UploadDiagnostics."

	parameters := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Complete"}},
		{Name: prefix + "BOMTime", ValueStruct: types.ValueStruct{Value: bom.Format(time.RFC3339)}},
		{Name: prefix + "EOMTime", ValueStruct: types.ValueStruct{Value: eom.Format(time.RFC3339)}},
		{Name: prefix + "TestBytesSent", ValueStruct: types.ValueStruct{Value: "1000"}},
	}

	assert.Nil(t, buildDiagnosticsResult(parameters, prefix, "TestBytesSent"))
}

func TestBuildDiagnosticsResult_MissingTimestampsExcluded(t *testing.T) {
	prefix := "Device.IP.Diagnostics.DownloadDiagnostics."
	parameters := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Complete"}},
	}

	assert.Nil(t, buildDiagnosticsResult(parameters, prefix, "TestBytesReceived"))
}
