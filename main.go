package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"goacs/acs"
	acshttp "goacs/http"
	"goacs/lib"
	"goacs/repository"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var Configuration Config

var Env *lib.Env

// version, commit and date are overridden at build time via
// -ldflags "-X main.version=... -X main.commit=... -X main.date=..."
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type Config struct {
	ACSHttpPort int
}

func init() {
	fmt.Println("Initializing app...")
	err := godotenv.Load()

	if err != nil {
		exitApp("Unable to load .env file", 1)
	}

	Env = new(lib.Env)

	Configuration.ACSHttpPort, err = strconv.Atoi(Env.Get("HTTP_PORT", "8085"))

	if err != nil {
		exitApp("Invalid HTTP_PORT", 1)
	}
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		fmt.Printf("goacs %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrateCommand()
		return
	}

	fmt.Println("Starting server...")
	repository.InitConnection()
	acs.StartSession()
	go PrintMemUsage()

	acshttp.Start()

}

// runMigrateCommand applies any contrib/database/*.sql file not yet recorded in the
// schema_migrations table - this is now the ONLY way schema changes happen, on a fresh
// database as much as an existing one (MariaDB's docker-compose service no longer
// bulk-applies contrib/database/*.sql on container init - see docker-compose.yml).
//
// Usage:
//
//	go run main.go migrate                                   # apply every not-yet-tracked file for real
//	go run main.go migrate --baseline file1.sql file2.sql     # one-time: record exactly these
//	                                                          # filenames as already applied,
//	                                                          # WITHOUT running them - for a
//	                                                          # database that already has this
//	                                                          # schema from before this tool
//	                                                          # existed (e.g. via the old
//	                                                          # docker-entrypoint bulk-init)
//
// The migrations directory defaults to contrib/database (relative to the working
// directory) and can be overridden with -dir=<path> before any --baseline filenames.
func runMigrateCommand() {
	args := os.Args[2:]

	dir := "contrib/database"
	if len(args) > 0 && strings.HasPrefix(args[0], "-dir=") {
		dir = strings.TrimPrefix(args[0], "-dir=")
		args = args[1:]
	}

	var baselineFiles []string
	if len(args) > 0 && args[0] == "--baseline" {
		baselineFiles = args[1:]
		if len(baselineFiles) == 0 {
			exitApp("migrate: --baseline requires at least one filename", 1)
		}
	}

	if err := repository.RunMigrations(dir, baselineFiles); err != nil {
		exitApp(fmt.Sprintf("migrate: %v", err), 1)
	}
}

func exitApp(msg string, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

func PrintMemUsage() {
	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		// For info on each, see: https://golang.org/pkg/runtime/#MemStats
		fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
		fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
		fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
		fmt.Printf("\tNumGC = %v\n", m.NumGC)
		time.Sleep(time.Second * 5)
	}
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
