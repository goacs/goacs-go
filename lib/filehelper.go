package lib

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// GetFileUrl builds the public download URL for a file already sitting in FILESTORE_PATH
// (used to point CPEs at firmware/config files via the Download RPC). requestedName must
// be a bare filename - traversal attempts (separators, "..") are rejected rather than
// silently collapsed, since this ends up in a URL served back to untrusted devices.
func GetFileUrl(requestedName string, request *http.Request) (string, error) {
	libenv := Env{}

	if requestedName == "" || strings.ContainsAny(requestedName, "/\\") {
		return "", errors.New("invalid filename")
	}

	safeName := filepath.Base(requestedName)
	if safeName != requestedName || safeName == "." || safeName == ".." {
		return "", errors.New("invalid filename")
	}

	fileDir := libenv.Get("FILESTORE_PATH", "./storage")
	absPath, err := filepath.Abs(fileDir)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(absPath, safeName)
	if !strings.HasPrefix(fullPath, absPath+string(os.PathSeparator)) {
		return "", errors.New("path traversal detected")
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("File does not exist\n")
		return "", err
	}

	urlStruct := url.URL{
		Scheme: "http",
		Host:   request.Host,
		Path:   "file/" + safeName,
	}

	return urlStruct.String(), nil
}
