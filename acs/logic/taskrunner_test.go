package logic

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunScriptMaxCount_FallsBackWithoutAWorkingConnection covers the path reachable
// without a live database: a connection that can never be established makes
// ConfigRepository.GetValue fail, so runScriptMaxCount must fall back to
// DefaultRunScriptMaxCount instead of propagating the error or panicking. The "a real
// config row overrides the default" path needs a live MySQL instance and is exercised
// via integration testing, same convention as the DB-touching Lua functions documented
// in acs/scripts/functions_test.go.
func TestRunScriptMaxCount_FallsBackWithoutAWorkingConnection(t *testing.T) {
	// sqlx.Open is lazy - it only fails once a query actually runs against this
	// deliberately unreachable address.
	db, err := sqlx.Open("mysql", "nobody:nothing@tcp(127.0.0.1:1)/nosuchdb")
	require.NoError(t, err)

	assert.Equal(t, DefaultRunScriptMaxCount, runScriptMaxCount(db))
}
