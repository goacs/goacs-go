package auth

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func runHandler(t *testing.T, req *http.Request, h gin.HandlerFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	called := false
	h(c)
	if !c.IsAborted() {
		called = true
	}

	return w, called
}

func TestBasicAuth_ValidCredentials(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/acs", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("acs:secret")))

	w, nextCalled := runHandler(t, req, func(c *gin.Context) { basicAuth(c, "acs", "secret") })

	assert.True(t, nextCalled)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

func TestBasicAuth_WrongPassword(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/acs", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("acs:wrong")))

	w, nextCalled := runHandler(t, req, func(c *gin.Context) { basicAuth(c, "acs", "secret") })

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm=")
}

func TestBasicAuth_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/acs", nil)

	w, nextCalled := runHandler(t, req, func(c *gin.Context) { basicAuth(c, "acs", "secret") })

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDigestAuth_NoHeaderIssuesChallenge(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/acs", nil)

	w, nextCalled := runHandler(t, req, func(c *gin.Context) { digestAuth(c, "acs", "secret") })

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), `Digest realm=`)
	assert.Contains(t, w.Header().Get("WWW-Authenticate"), `nonce=`)
}

func TestDigestAuth_ValidResponse(t *testing.T) {
	// First request: get challenged and capture the server-issued nonce.
	challengeReq := httptest.NewRequest(http.MethodGet, "/acs", nil)
	w, _ := runHandler(t, challengeReq, func(c *gin.Context) { digestAuth(c, "acs", "secret") })
	nonce := extractParam(t, w.Header().Get("WWW-Authenticate"), "nonce")
	assert.NotEmpty(t, nonce)

	// Second request: compute a correct RFC2617 digest response using that nonce.
	const uri = "/acs"
	const cnonce = "clientnonce123"
	const nc = "00000001"
	ha1 := md5hex(fmt.Sprintf("%s:%s:%s", "acs", digestRealm, "secret"))
	ha2 := md5hex(fmt.Sprintf("%s:%s", http.MethodGet, uri))
	resp := md5hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, "auth", ha2))

	authHeader := fmt.Sprintf(
		`Digest username="acs", realm="%s", nonce="%s", uri="%s", qop=auth, nc=%s, cnonce="%s", response="%s"`,
		digestRealm, nonce, uri, nc, cnonce, resp,
	)

	req := httptest.NewRequest(http.MethodGet, uri, nil)
	req.Header.Set("Authorization", authHeader)

	w2, nextCalled := runHandler(t, req, func(c *gin.Context) { digestAuth(c, "acs", "secret") })

	assert.True(t, nextCalled)
	assert.NotEqual(t, http.StatusUnauthorized, w2.Code)
}

func TestDigestAuth_WrongPasswordRejected(t *testing.T) {
	challengeReq := httptest.NewRequest(http.MethodGet, "/acs", nil)
	w, _ := runHandler(t, challengeReq, func(c *gin.Context) { digestAuth(c, "acs", "secret") })
	nonce := extractParam(t, w.Header().Get("WWW-Authenticate"), "nonce")

	const uri = "/acs"
	const cnonce = "clientnonce123"
	const nc = "00000001"
	// Computed against the WRONG password.
	ha1 := md5hex(fmt.Sprintf("%s:%s:%s", "acs", digestRealm, "totally-wrong"))
	ha2 := md5hex(fmt.Sprintf("%s:%s", http.MethodGet, uri))
	resp := md5hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, "auth", ha2))

	authHeader := fmt.Sprintf(
		`Digest username="acs", realm="%s", nonce="%s", uri="%s", qop=auth, nc=%s, cnonce="%s", response="%s"`,
		digestRealm, nonce, uri, nc, cnonce, resp,
	)

	req := httptest.NewRequest(http.MethodGet, uri, nil)
	req.Header.Set("Authorization", authHeader)

	w2, nextCalled := runHandler(t, req, func(c *gin.Context) { digestAuth(c, "acs", "secret") })

	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestNonceExpiry(t *testing.T) {
	nonce := newNonce()
	assert.True(t, nonceValid(nonce))

	nonces.mu.Lock()
	nonces.items[nonce] = nonces.items[nonce].Add(-digestNonceTTL * 2)
	nonces.mu.Unlock()

	assert.False(t, nonceValid(nonce))
}

// extractParam pulls a quoted or bare param value out of a WWW-Authenticate header, e.g.
// extractParam(`Digest realm="x", nonce="abc"`, "nonce") -> "abc".
func extractParam(t *testing.T, header, key string) string {
	t.Helper()

	params := parseDigestParams(header[len("Digest "):])
	return params[key]
}

func TestMD5Hex(t *testing.T) {
	sum := md5.Sum([]byte("abc"))
	assert.Equal(t, hex.EncodeToString(sum[:]), md5hex("abc"))
}
