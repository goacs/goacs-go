package auth

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	digestRealm      = "GoACS Authorization"
	digestNonceTTL   = 5 * time.Minute
	digestSweepEvery = time.Minute
)

// nonceStore tracks nonces issued to CPEs. TR-069 devices commonly reuse a nonce for the
// duration of a session, so we only enforce expiry rather than strict single-use replay
// protection (a full nonce-count ledger is not worth the complexity for this use case).
var nonces = struct {
	mu    sync.Mutex
	items map[string]time.Time
}{items: make(map[string]time.Time)}

func init() {
	go func() {
		for {
			time.Sleep(digestSweepEvery)
			now := time.Now()
			nonces.mu.Lock()
			for n, issued := range nonces.items {
				if now.Sub(issued) > digestNonceTTL {
					delete(nonces.items, n)
				}
			}
			nonces.mu.Unlock()
		}
	}()
}

func newNonce() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	nonce := hex.EncodeToString(buf)

	nonces.mu.Lock()
	nonces.items[nonce] = time.Now()
	nonces.mu.Unlock()

	return nonce
}

func nonceValid(nonce string) bool {
	nonces.mu.Lock()
	defer nonces.mu.Unlock()

	issued, ok := nonces.items[nonce]
	if !ok {
		return false
	}

	return time.Since(issued) <= digestNonceTTL
}

func challengeDigest(c *gin.Context) {
	nonce := newNonce()
	header := fmt.Sprintf(`Digest realm="%s", qop="auth", nonce="%s", opaque="%s"`,
		digestRealm, nonce, md5hex(digestRealm))
	c.Writer.Header().Set("WWW-Authenticate", header)
	c.AbortWithStatus(http.StatusUnauthorized)
}

func digestAuth(c *gin.Context, expectedUser, expectedPass string) {
	header := c.Request.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Digest ") {
		challengeDigest(c)
		return
	}

	params := parseDigestParams(header[len("Digest "):])
	if params["username"] != expectedUser || !nonceValid(params["nonce"]) {
		challengeDigest(c)
		return
	}

	ha1 := md5hex(fmt.Sprintf("%s:%s:%s", expectedUser, digestRealm, expectedPass))
	ha2 := md5hex(fmt.Sprintf("%s:%s", c.Request.Method, params["uri"]))

	var expectedResponse string
	if params["qop"] == "auth" {
		expectedResponse = md5hex(fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			ha1, params["nonce"], params["nc"], params["cnonce"], params["qop"], ha2))
	} else {
		expectedResponse = md5hex(fmt.Sprintf("%s:%s:%s", ha1, params["nonce"], ha2))
	}

	if !constantTimeEqual(expectedResponse, params["response"]) {
		challengeDigest(c)
		return
	}

	c.Next()
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func parseDigestParams(raw string) map[string]string {
	params := make(map[string]string)

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		params[strings.TrimSpace(kv[0])] = strings.Trim(strings.TrimSpace(kv[1]), `"`)
	}

	return params
}
