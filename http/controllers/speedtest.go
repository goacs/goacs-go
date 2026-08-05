package controllers

import (
	"github.com/gin-gonic/gin"
	"goacs/http/response"
	"goacs/lib"
	"io"
	"net/http"
	"strconv"
)

// defaultSpeedtestBytes is used when the CPE-supplied ?bytes= is missing/invalid.
const defaultSpeedtestBytes = 20 * 1024 * 1024 // 20MiB

// defaultSpeedtestMaxBytes caps both endpoints so an unauthenticated, internet-facing
// route can't be turned into a bandwidth/memory amplification vector. Overridable via
// SPEEDTEST_MAX_BYTES, same env-knob convention as FILESTORE_PATH (lib/filehelper.go).
const defaultSpeedtestMaxBytes = 500 * 1024 * 1024 // 500MiB

const speedtestChunkSize = 32 * 1024

func speedtestMaxBytes() int64 {
	env := lib.Env{}
	if v, err := strconv.ParseInt(env.Get("SPEEDTEST_MAX_BYTES", ""), 10, 64); err == nil && v > 0 {
		return v
	}
	return defaultSpeedtestMaxBytes
}

// SpeedtestDownload streams exactly ?bytes=N filler octets so a CPE's TR-143
// DownloadDiagnostics can measure throughput against GoACS itself, without needing any
// file pre-staged in FILESTORE_PATH. Content is generated from a small reused buffer, not
// read from disk, so arbitrarily large transfers stay cheap for the server.
func SpeedtestDownload(ctx *gin.Context) {
	maxBytes := speedtestMaxBytes()

	size := int64(defaultSpeedtestBytes)
	if q := ctx.Query("bytes"); q != "" {
		if parsed, err := strconv.ParseInt(q, 10, 64); err == nil && parsed > 0 {
			size = parsed
		}
	}
	if size > maxBytes {
		size = maxBytes
	}

	ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
	ctx.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	ctx.Writer.WriteHeader(http.StatusOK)

	buf := make([]byte, speedtestChunkSize)
	for remaining := size; remaining > 0; {
		chunk := int64(len(buf))
		if remaining < chunk {
			chunk = remaining
		}
		written, err := ctx.Writer.Write(buf[:chunk])
		if err != nil {
			return
		}
		remaining -= int64(written)
	}
}

// SpeedtestUpload is the counterpart target for TR-143 UploadDiagnostics: it reads and
// discards whatever the CPE sends, capped at SPEEDTEST_MAX_BYTES, and reports how many
// bytes it actually received.
func SpeedtestUpload(ctx *gin.Context) {
	maxBytes := speedtestMaxBytes()

	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBytes)
	written, err := io.Copy(io.Discard, ctx.Request.Body)
	if err != nil {
		response.ResponseError(ctx, http.StatusRequestEntityTooLarge, "upload too large", "")
		return
	}

	response.ResponseData(ctx, gin.H{"bytes_received": written})
}
