// Package context holds cache key constants for the ephemeral, per-device
// ACS state (on-demand parameter lookup, one-shot forced reprovisioning)
// that goacs-php kept in Redis under App\ACS\Context. Here it's backed by
// lib/cache instead of Redis, keyed by CPE serial number.
package context

const (
	LookupParamsPrefix        = "LOOKUP_PARAMS_"
	LookupParamsEnabledPrefix = "LOOKUP_PARAMS_ENABLED_"
	ProvisionPrefix           = "PROVISION_"
)

func KeyFor(prefix, serial string) string {
	return prefix + serial
}
