package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const basicRealm = "GoACS Authorization"

func basicAuth(c *gin.Context, expectedUser, expectedPass string) {
	user, pass, ok := parseBasicAuthHeader(c.Request.Header.Get("Authorization"))

	if !ok || !constantTimeEqual(user, expectedUser) || !constantTimeEqual(pass, expectedPass) {
		c.Writer.Header().Set("WWW-Authenticate", `Basic realm="`+basicRealm+`"`)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Next()
}

func parseBasicAuthHeader(header string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(header[len(prefix):])
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
