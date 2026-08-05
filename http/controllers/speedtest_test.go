package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSpeedtestDownload_DefaultSize(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/speedtest/download", nil)

	SpeedtestDownload(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Len(t, w.Body.Bytes(), defaultSpeedtestBytes)
}

func TestSpeedtestDownload_HonorsBytesQueryParam(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/speedtest/download?bytes=12345", nil)

	SpeedtestDownload(ctx)

	assert.Equal(t, "12345", w.Header().Get("Content-Length"))
	assert.Len(t, w.Body.Bytes(), 12345)
}

func TestSpeedtestDownload_CapsAtMaxBytesEnv(t *testing.T) {
	t.Setenv("SPEEDTEST_MAX_BYTES", "1000")

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/speedtest/download?bytes=999999", nil)

	SpeedtestDownload(ctx)

	assert.Len(t, w.Body.Bytes(), 1000)
}

func TestSpeedtestUpload_ReportsBytesReceived(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 4096)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/speedtest/upload", bytes.NewReader(body))

	SpeedtestUpload(ctx)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"bytes_received":4096`)
}

func TestSpeedtestUpload_RejectsBodyOverCap(t *testing.T) {
	t.Setenv("SPEEDTEST_MAX_BYTES", "100")

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/speedtest/upload", strings.NewReader(strings.Repeat("x", 1000)))

	SpeedtestUpload(ctx)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}
