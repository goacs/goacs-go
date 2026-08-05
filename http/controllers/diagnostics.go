package controllers

import (
	"github.com/gin-gonic/gin"
	acshttp "goacs/acs/http"
	"goacs/acs/types"
	"goacs/http/request"
	"goacs/http/response"
	"goacs/models/cpe"
	"goacs/repository"
	"goacs/repository/mysql"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// TR-143 nests the diagnostics objects differently per data model: TR-181 (Device:2)
// under Device.IP.Diagnostics., legacy TR-098 directly under InternetGatewayDevice. -
// see cpe.DetermineDeviceTreeRootPath, which every session already resolves.
func diagnosticsPrefix(root string) string {
	if root == "Device" {
		return root + ".IP.Diagnostics."
	}
	return root + "."
}

// selfHostedSpeedtestURL builds a URL pointing back at this GoACS instance's own
// /speedtest/download or /speedtest/upload endpoint (see speedtest.go), using the same
// request.Host-based convention lib.GetFileUrl already uses for firmware URLs.
func selfHostedSpeedtestURL(request *http.Request, path string, query url.Values) string {
	u := url.URL{Scheme: "http", Host: request.Host, Path: path}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

type DownloadDiagnosticsRequest struct {
	URL                 string `json:"url" validate:"omitempty,url"`
	Bytes               int    `json:"bytes" validate:"omitempty,min=1"`
	NumberOfConnections int    `json:"number_of_connections" validate:"omitempty,min=1"`
}

type UploadDiagnosticsRequest struct {
	URL                 string `json:"url" validate:"omitempty,url"`
	TestFileLength      int    `json:"test_file_length" validate:"required,min=1"`
	NumberOfConnections int    `json:"number_of_connections" validate:"omitempty,min=1"`
}

func RunDownloadDiagnostics(ctx *gin.Context) {
	var req DownloadDiagnosticsRequest
	_ = ctx.ShouldBind(&req)

	validator := request.NewApiValidator(ctx, req)
	if err := validator.Validate(); err != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	parameters, err := cperepository.GetCPEParameters(cpeModel)
	if err != nil {
		response.Response500(ctx, err.Error(), "")
		return
	}

	prefix := diagnosticsPrefix(cpe.DetermineDeviceTreeRootPath(parameters)) + "DownloadDiagnostics."

	downloadURL := req.URL
	if downloadURL == "" {
		query := url.Values{}
		if req.Bytes > 0 {
			query.Set("bytes", strconv.Itoa(req.Bytes))
		}
		downloadURL = selfHostedSpeedtestURL(ctx.Request, "speedtest/download", query)
	}

	params := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Requested", Type: "xsd:string"}},
		{Name: prefix + "DownloadURL", ValueStruct: types.ValueStruct{Value: downloadURL, Type: "xsd:string"}},
	}
	if req.NumberOfConnections > 0 {
		params = append(params, types.ParameterValueStruct{
			Name:        prefix + "NumberOfConnections",
			ValueStruct: types.ValueStruct{Value: strconv.Itoa(req.NumberOfConnections), Type: "xsd:unsignedInt"},
		})
	}

	acshttp.NewACSRequest(cpeModel).RunDiagnostics(params)
	response.ResponseData(ctx, "")
}

func RunUploadDiagnostics(ctx *gin.Context) {
	var req UploadDiagnosticsRequest
	_ = ctx.ShouldBind(&req)

	validator := request.NewApiValidator(ctx, req)
	if err := validator.Validate(); err != nil {
		response.ResponseValidationErrors(ctx, validator)
		return
	}

	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	parameters, err := cperepository.GetCPEParameters(cpeModel)
	if err != nil {
		response.Response500(ctx, err.Error(), "")
		return
	}

	prefix := diagnosticsPrefix(cpe.DetermineDeviceTreeRootPath(parameters)) + "UploadDiagnostics."

	uploadURL := req.URL
	if uploadURL == "" {
		uploadURL = selfHostedSpeedtestURL(ctx.Request, "speedtest/upload", nil)
	}

	params := []types.ParameterValueStruct{
		{Name: prefix + "DiagnosticsState", ValueStruct: types.ValueStruct{Value: "Requested", Type: "xsd:string"}},
		{Name: prefix + "UploadURL", ValueStruct: types.ValueStruct{Value: uploadURL, Type: "xsd:string"}},
		{Name: prefix + "TestFileLength", ValueStruct: types.ValueStruct{Value: strconv.Itoa(req.TestFileLength), Type: "xsd:unsignedInt"}},
	}
	if req.NumberOfConnections > 0 {
		params = append(params, types.ParameterValueStruct{
			Name:        prefix + "NumberOfConnections",
			ValueStruct: types.ValueStruct{Value: strconv.Itoa(req.NumberOfConnections), Type: "xsd:unsignedInt"},
		})
	}

	acshttp.NewACSRequest(cpeModel).RunDiagnostics(params)
	response.ResponseData(ctx, "")
}

type DiagnosticsResult struct {
	State           string    `json:"state"`
	Bytes           int64     `json:"bytes"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	DurationSeconds float64   `json:"duration_seconds"`
	ThroughputMbps  float64   `json:"throughput_mbps"`
}

type DiagnosticsReport struct {
	Download *DiagnosticsResult `json:"download"`
	Upload   *DiagnosticsResult `json:"upload"`
}

// diagnosticsResultTTL is how long after a test finishes (per the CPE's own EOMTime,
// not our DB's updated_at) the report endpoint still reports it - older results are
// silently dropped rather than surfaced as stale.
const diagnosticsResultTTL = 24 * time.Hour

func GetDiagnosticsReport(ctx *gin.Context) {
	cperepository := mysql.NewCPERepository(repository.GetConnection())
	cpeModel, err := getCPEFromContext(ctx, cperepository)
	if err != nil {
		return
	}

	parameters, err := cperepository.GetCPEParameters(cpeModel)
	if err != nil {
		response.Response500(ctx, err.Error(), "")
		return
	}

	root := cpe.DetermineDeviceTreeRootPath(parameters)

	report := DiagnosticsReport{
		Download: buildDiagnosticsResult(parameters, diagnosticsPrefix(root)+"DownloadDiagnostics.", "TestBytesReceived"),
		Upload:   buildDiagnosticsResult(parameters, diagnosticsPrefix(root)+"UploadDiagnostics.", "TestBytesSent"),
	}

	response.ResponseData(ctx, report)
}

func findParameterValue(parameters []types.ParameterValueStruct, name string) (string, bool) {
	for _, p := range parameters {
		if p.Name == name {
			return p.ValueStruct.Value, true
		}
	}
	return "", false
}

// buildDiagnosticsResult returns nil (omitted from the report) unless the CPE reports
// DiagnosticsState "Complete" with a parseable BOMTime/EOMTime pair and EOMTime falls
// within diagnosticsResultTTL - callers never see an incomplete or stale result.
func buildDiagnosticsResult(parameters []types.ParameterValueStruct, prefix string, bytesParamName string) *DiagnosticsResult {
	state, ok := findParameterValue(parameters, prefix+"DiagnosticsState")
	if !ok || state != "Complete" {
		return nil
	}

	bomRaw, ok := findParameterValue(parameters, prefix+"BOMTime")
	if !ok {
		return nil
	}
	eomRaw, ok := findParameterValue(parameters, prefix+"EOMTime")
	if !ok {
		return nil
	}

	bom, err := parseXsdDateTime(bomRaw)
	if err != nil {
		return nil
	}
	eom, err := parseXsdDateTime(eomRaw)
	if err != nil {
		return nil
	}

	if time.Since(eom) > diagnosticsResultTTL {
		return nil
	}

	bytesRaw, _ := findParameterValue(parameters, prefix+bytesParamName)
	bytesValue, _ := strconv.ParseInt(bytesRaw, 10, 64)

	duration := eom.Sub(bom).Seconds()
	var throughputMbps float64
	if duration > 0 {
		throughputMbps = float64(bytesValue) * 8 / duration / 1e6
	}

	return &DiagnosticsResult{
		State:           state,
		Bytes:           bytesValue,
		StartTime:       bom,
		EndTime:         eom,
		DurationSeconds: duration,
		ThroughputMbps:  throughputMbps,
	}
}

func parseXsdDateTime(value string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	var err error
	for _, layout := range layouts {
		var t time.Time
		if t, err = time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}
