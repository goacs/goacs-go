//go:build scenario

package harness

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Server is a running goacs-go instance under test, pointed at its own
// disposable database.
type Server struct {
	BaseURL string

	dbName  string
	rootDSN func(dbName string) string
	cmd     *exec.Cmd
	output  *syncBuffer
}

var (
	buildServerOnce sync.Once
	serverBinary    string
	serverBuildErr  error
)

// buildServerBinary compiles the goacs-go binary once per test run and
// returns its path. Reused by every StartServer call so repeated scenarios
// don't each pay a full `go build`.
func buildServerBinary(t *testing.T) string {
	t.Helper()

	buildServerOnce.Do(func() {
		dir, err := os.MkdirTemp("", "goacs-scenario-server-*")
		if err != nil {
			serverBuildErr = err
			return
		}

		binPath := filepath.Join(dir, "goacs-go-scenario")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = RepoRoot()
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			serverBuildErr = fmt.Errorf("building goacs-go binary: %w\n%s", err, out.String())
			return
		}

		serverBinary = binPath
	})

	if serverBuildErr != nil {
		t.Fatalf("harness: %v", serverBuildErr)
	}

	return serverBinary
}

// devEnv reads the repo's own .env (the same one `go run main.go` picks up
// for normal dev use) without mutating the current process's environment, so
// the harness knows how to reach the same MariaDB instance `docker compose up
// -d goacs-db` starts.
func devEnv(t *testing.T) map[string]string {
	t.Helper()

	envPath := filepath.Join(RepoRoot(), ".env")
	env, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("harness: reading %s: %v (run `cp .env.example .env` first, per the dev workflow in AGENTS.md)", envPath, err)
	}
	return env
}

func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("harness: allocating a free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// StartServer builds (once) and starts a real goacs-go server backed by a
// freshly created, uniquely-named database, migrated from scratch. It
// registers t.Cleanup to stop the process and drop the database. Requires
// `docker compose up -d goacs-db` (or equivalent) already running per the
// MYSQL_HOST/MYSQL_PORT/MYSQL_ROOT_PASSWORD in the repo's .env.
//
// The server (and the migrate step) run authenticated as the MySQL root user
// against a disposable, uniquely-named schema - simplest way to get
// CREATE/DROP DATABASE rights on a schema the app's normal MYSQL_USER isn't
// otherwise granted. This is only ever pointed at a local/dev MariaDB
// instance, never a shared or production one.
func StartServer(t *testing.T) *Server {
	t.Helper()

	binPath := buildServerBinary(t)
	baseEnv := devEnv(t)

	dbName := fmt.Sprintf("goacs_scenario_%d", time.Now().UnixNano())
	rootDSN := func(name string) string {
		return fmt.Sprintf("root:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true",
			baseEnv["MYSQL_ROOT_PASSWORD"], baseEnv["MYSQL_HOST"], baseEnv["MYSQL_PORT"], name)
	}

	adminDB, err := sql.Open("mysql", rootDSN(""))
	if err != nil {
		t.Fatalf("harness: opening root MySQL connection: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("harness: creating test database %s: %v", dbName, err)
	}

	srv := &Server{dbName: dbName, rootDSN: rootDSN}
	t.Cleanup(srv.stop)

	childEnv := envSlice(baseEnv, map[string]string{
		"MYSQL_USER":     "root",
		"MYSQL_PASSWORD": baseEnv["MYSQL_ROOT_PASSWORD"],
		"MYSQL_DATABASE": dbName,
	})

	migrate := exec.Command(binPath, "migrate")
	migrate.Dir = RepoRoot()
	migrate.Env = childEnv
	var migrateOut bytes.Buffer
	migrate.Stdout = &migrateOut
	migrate.Stderr = &migrateOut
	if err := migrate.Run(); err != nil {
		t.Fatalf("harness: running migrations against %s: %v\n%s", dbName, err, migrateOut.String())
	}

	port := freePort(t)
	runEnv := envSlice(baseEnv, map[string]string{
		"MYSQL_USER":     "root",
		"MYSQL_PASSWORD": baseEnv["MYSQL_ROOT_PASSWORD"],
		"MYSQL_DATABASE": dbName,
		"HTTP_PORT":      fmt.Sprintf("%d", port),
	})

	cmd := exec.Command(binPath)
	cmd.Dir = RepoRoot()
	cmd.Env = runEnv
	out := &syncBuffer{}
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Start(); err != nil {
		t.Fatalf("harness: starting goacs-go server: %v", err)
	}
	srv.cmd = cmd
	srv.output = out
	srv.BaseURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	if err := waitForServer(srv.BaseURL, 20*time.Second); err != nil {
		t.Fatalf("harness: server at %s never became ready: %v\n--- server output ---\n%s", srv.BaseURL, err, out.String())
	}

	return srv
}

func waitForServer(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Post(baseURL+"/api/auth/login", "application/json", bytes.NewReader([]byte(`{}`)))
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(150 * time.Millisecond)
	}
	return lastErr
}

// Output returns everything the goacs-go server process has printed to
// stdout/stderr so far - notably where the Lua log() function's output ends
// up (log.Printf on the server process, not any DB table - see
// acs/scripts/functions.go luaLog).
func (s *Server) Output() string {
	return s.output.String()
}

func (s *Server) stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}

	if s.dbName == "" {
		return
	}
	adminDB, err := sql.Open("mysql", s.rootDSN(""))
	if err != nil {
		return
	}
	defer adminDB.Close()
	_, _ = adminDB.Exec("DROP DATABASE IF EXISTS `" + s.dbName + "`")
}

// envSlice builds a child process environment: everything from base (the
// repo's .env, read but not yet applied to any process), overridden by
// overrides, formatted as VAR=value strings. Values set here win over the
// child's own .env-file loading, since godotenv.Load only fills in variables
// not already present in the process environment.
func envSlice(base map[string]string, overrides map[string]string) []string {
	merged := make(map[string]string, len(base)+len(overrides))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}

	env := os.Environ()
	for k, v := range merged {
		env = append(env, k+"="+v)
	}
	return env
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
