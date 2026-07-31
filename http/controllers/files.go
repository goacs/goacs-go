package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"goacs/http/response"
	"goacs/lib"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxUploadSize = 512 * 1024 * 1024 // 512MB, generous enough for firmware images

type FileInfoResponse struct {
	Size     int64     `json:"size"`
	Filename string    `json:"filename"`
	IsDir    bool      `json:"is_dir"`
	ModTime  time.Time `json:"mod_time"`
}

func ListFiles(ctx *gin.Context) {
	env := lib.Env{}

	fileDir := env.Get("FILESTORE_PATH", "./storage")
	absPath, _ := filepath.Abs(fileDir)
	files, err := ioutil.ReadDir(absPath)

	if err != nil {
		response.ResponseError(ctx, http.StatusInternalServerError, "File list error", err)
		return
	}

	var fileResponse []FileInfoResponse

	for _, file := range files {
		fileResponse = append(fileResponse, FileInfoResponse{
			Size:     file.Size(),
			Filename: file.Name(),
			IsDir:    file.IsDir(),
			ModTime:  file.ModTime(),
		})
	}
	response.ResponseData(ctx, fileResponse)
}

func UploadFile(ctx *gin.Context) {
	env := lib.Env{}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, "", err)
		return
	}

	if fileHeader.Size > maxUploadSize {
		response.ResponseError(ctx, http.StatusBadRequest, "File too large", nil)
		return
	}

	filePath, safeName, err := resolveStoragePath(env, fileHeader.Filename)
	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()), err)
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()), err)
		return
	}
	defer src.Close()

	// O_EXCL guarantees the create-if-absent check above and this write are atomic,
	// closing the TOCTOU window a separate fileExists() check followed by SaveUploadedFile left open.
	dst, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			response.ResponseError(ctx, http.StatusBadRequest, fmt.Sprintf("File %s exists", safeName), err)
			return
		}
		response.ResponseError(ctx, http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()), err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(filePath)
		response.ResponseError(ctx, http.StatusInternalServerError, fmt.Sprintf("upload file err: %s", err.Error()), err)
		return
	}

	response.ResponseData(ctx, FileInfoResponse{Size: fileHeader.Size, Filename: safeName})
}

func DownloadFile(ctx *gin.Context) {
	env := lib.Env{}

	filePath, safeName, err := resolveStoragePath(env, ctx.Param("filename"))
	if err != nil {
		response.ResponseError(ctx, http.StatusBadRequest, "invalid filename", err)
		return
	}

	ctx.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", safeName))
	ctx.Writer.Header().Add("Content-Type", "application/octet-stream")
	ctx.File(filePath)
}

// resolveStoragePath rejects path traversal / separators in the requested filename and
// resolves it against the storage root, with a containment check as defense in depth.
func resolveStoragePath(env lib.Env, requestedName string) (fullPath string, safeName string, err error) {
	if requestedName == "" || strings.ContainsAny(requestedName, "/\\") {
		return "", "", errors.New("invalid filename")
	}

	safeName = filepath.Base(requestedName)
	if safeName != requestedName || safeName == "." || safeName == ".." {
		return "", "", errors.New("invalid filename")
	}

	fileDir := env.Get("FILESTORE_PATH", "./storage")
	storageAbs, err := filepath.Abs(fileDir)
	if err != nil {
		return "", "", err
	}

	fullPath = filepath.Join(storageAbs, safeName)
	if !strings.HasPrefix(fullPath, storageAbs+string(os.PathSeparator)) {
		return "", "", errors.New("path traversal detected")
	}

	return fullPath, safeName, nil
}
