package repository

import (
	"fmt"
	"goacs/lib"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// RunMigrations applies every *.sql file in dir, in filename order, that hasn't been
// recorded in the schema_migrations table yet - the schema_migrations table is the
// single source of truth for what's applied, same approach as Laravel/most migration
// tools. There is no attempt to guess "this file's effects already exist" from error
// codes: that only works reliably for CREATE-TABLE-style additive DDL and silently falls
// apart for anything else (a column MODIFY or a plain INSERT without a unique
// constraint doesn't error on a second run at all, so guessing from errors would let it
// re-apply - and for an UPDATE/INSERT that's actual data corruption, not a false
// positive worth ignoring). So: any error here is a real error, full stop, and the
// docker-compose MariaDB service does NOT bulk-apply contrib/database/*.sql on
// container init (see docker-compose.yml) - every environment, fresh or existing, goes
// through this same tracked path from the very first migration.
//
// baselineFiles is the one-time escape hatch for a database that already has schema
// from before this tool existed (e.g. MariaDB's docker-entrypoint used to bulk-apply
// contrib/database/*.sql on first container init): when non-empty, ONLY those exact
// filenames are recorded as applied, WITHOUT being run - every other not-yet-tracked
// file in dir is left untouched for a later, ordinary migrate run to apply for real.
// This is deliberately an explicit list, not "baseline everything currently in dir":
// an existing deployment is rarely caught up to the latest file (e.g. 01-03 already
// applied by the old docker-entrypoint mechanism, but a just-added 04 genuinely isn't
// yet), and baselining a file whose effects are NOT actually present would leave the
// database silently behind with no error to signal it. Run this once for exactly the
// historical files that are already known to be applied, then use an ordinary
// (baselineFiles == nil) run for anything after.
//
// It opens its own short-lived connection (separate from the shared pool from
// InitConnection/GetConnection) with multiStatements enabled, since a migration file
// may contain several DDL/DML statements one after another. That flag is deliberately
// NOT enabled on the main application connection - it would widen the blast radius of
// any raw-SQL-concatenation bug elsewhere in the app into stacked-query injection ("one
// bad string turns one statement into several"), which is an acceptable trade solely for
// this trusted, operator-invoked, local-file command.
//
// Caveat: MySQL's DDL (CREATE/ALTER TABLE, ...) auto-commits per statement regardless of
// any transaction, so if a file with several statements fails partway through, whichever
// statements ran before the failing one are NOT rolled back even though the file itself
// is correctly left unmarked (so a retry re-attempts the whole file from the top, and
// may then fail again on an early "already exists" for what did get created). This is a
// property of MySQL DDL, not something this tool can paper over - keep individual
// migration files small and internally consistent, and inspect the schema by hand if a
// migration ever fails partway through before fixing and re-running it.
func RunMigrations(dir string, baselineFiles []string) error {
	env := new(lib.Env)
	connectionString := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true&multiStatements=true",
		env.Get("MYSQL_USER", ""),
		env.Get("MYSQL_PASSWORD", ""),
		env.Get("MYSQL_HOST", ""),
		env.Get("MYSQL_PORT", "3306"),
		env.Get("MYSQL_DATABASE", ""),
	)

	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		return fmt.Errorf("migrate: connect: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("migrate: ping: %w", err)
	}

	if _, err := db.Exec(`
		create table if not exists schema_migrations (
			filename varchar(255) not null primary key,
			applied_at datetime default current_timestamp not null
		)
	`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("migrate: list %s: %w", dir, err)
	}
	sort.Strings(files)

	if len(baselineFiles) > 0 {
		return baselineMigrations(db, dir, files, applied, baselineFiles)
	}

	applyCount := 0
	for _, path := range files {
		filename := filepath.Base(path)
		if applied[filename] {
			fmt.Printf("migrate: %s already applied, skipping\n", filename)
			continue
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", filename, err)
		}

		fmt.Printf("migrate: applying %s...\n", filename)
		if _, err := db.Exec(string(contents)); err != nil {
			return fmt.Errorf("migrate: %s: %w", filename, err)
		}

		if _, err := db.Exec("insert into schema_migrations (filename) values (?)", filename); err != nil {
			return fmt.Errorf("migrate: record %s: %w", filename, err)
		}

		applyCount++
	}

	fmt.Printf("migrate: done, %d file(s) newly applied\n", applyCount)
	return nil
}

// baselineMigrations records exactly the requested filenames as applied, without
// running any of them, after checking each one actually exists in dir (protects against
// a typo silently no-opping) and isn't already tracked.
func baselineMigrations(db *sqlx.DB, dir string, files []string, applied map[string]bool, baselineFiles []string) error {
	onDisk := map[string]bool{}
	for _, path := range files {
		onDisk[filepath.Base(path)] = true
	}

	for _, filename := range baselineFiles {
		if !onDisk[filename] {
			return fmt.Errorf("migrate: --baseline %s: no such file in %s", filename, dir)
		}
	}

	baselineCount := 0
	for _, filename := range baselineFiles {
		if applied[filename] {
			fmt.Printf("migrate: %s already applied, skipping\n", filename)
			continue
		}

		fmt.Printf("migrate: baselining %s (not executed, only recorded)\n", filename)
		if _, err := db.Exec("insert into schema_migrations (filename) values (?)", filename); err != nil {
			return fmt.Errorf("migrate: record %s: %w", filename, err)
		}

		baselineCount++
	}

	fmt.Printf("migrate: done, %d file(s) newly baselined\n", baselineCount)
	return nil
}

func appliedMigrations(db *sqlx.DB) (map[string]bool, error) {
	applied := map[string]bool{}

	rows, err := db.Query("select filename from schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: read schema_migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("migrate: scan schema_migrations: %w", err)
		}
		applied[filename] = true
	}

	return applied, nil
}
