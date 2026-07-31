package auth

import (
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"goacs/repository"
	"goacs/repository/mysql"
)

const (
	authTypeBasic  = "basic"
	authTypeDigest = "digest"

	configKeyAuthType = "acs_auth_type"
	configKeyUsername = "acs_auth_username"
	configKeyPassword = "acs_auth_password"
)

// ACSAuthMiddleware authenticates incoming CPE requests on /acs according to the
// acs_auth_type value stored in the config table (unset/"none", "basic" or "digest").
// The config is re-read on every request (not cached) so changing it from the admin
// panel takes effect immediately without a restart.
func ACSAuthMiddleware() gin.HandlerFunc {
	var warnOnce sync.Once

	return func(c *gin.Context) {
		configRepository := mysql.NewConfigRepository(repository.GetConnection())
		authType, _ := configRepository.GetValue(configKeyAuthType)
		username, _ := configRepository.GetValue(configKeyUsername)
		password, _ := configRepository.GetValue(configKeyPassword)

		switch authType {
		case authTypeBasic:
			basicAuth(c, username, password)
		case authTypeDigest:
			digestAuth(c, username, password)
		default:
			warnOnce.Do(func() {
				log.Println("WARNING: /acs does not require authentication (acs_auth_type is unset or 'none'). " +
					"Set config keys acs_auth_type=basic|digest and acs_auth_username/acs_auth_password to secure CPE connections.")
			})
			c.Next()
		}
	}
}
