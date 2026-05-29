package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func dbHost() string {
	if host := os.Getenv("DB_HOST"); host != "" {
		return host
	}
	return "localhost"
}

func dbPassword() string {
	if pwd := os.Getenv("PGPASSWORD"); pwd != "" {
		return pwd
	}
	return os.Getenv("DB_PASSWORD")
}

func adminDSN(dbUser string) string {
	pwd := dbPassword()
	if pwd != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:5432/postgres?sslmode=disable", dbUser, pwd, dbHost())
	}
	return fmt.Sprintf("postgres://%s@%s:5432/postgres?sslmode=disable", dbUser, dbHost())
}

func dbDSN(dbUser, dbName string) string {
	pwd := dbPassword()
	if pwd != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", dbUser, pwd, dbHost(), dbName)
	}
	return fmt.Sprintf("postgres://%s@%s:5432/%s?sslmode=disable", dbUser, dbHost(), dbName)
}

// ResolveDBName returns the effective database name for isolated test runs.
func ResolveDBName(dbName, pkg string) string {
	if os.Getenv("TEST_PARALLEL_MODE") != "" && pkg != "" {
		return dbName + "_" + pkg
	}
	return dbName
}

// BootstrapLocalDatabaseWithPkg creates the test database and applies migrations.
func BootstrapLocalDatabaseWithPkg(dbUser, dbName, pkg, migrationsDir string) *sqlx.DB {
	return bootstrapDB(dbUser, ResolveDBName(dbName, pkg), migrationsDir)
}

func bootstrapDB(dbUser, dbName, migrationsDir string) *sqlx.DB {
	admin := sqlx.MustConnect("pgx", adminDSN(dbUser))
	defer admin.Close()

	var exists bool
	if err := admin.Get(&exists, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName); err != nil {
		panic(err)
	}
	if !exists {
		if _, err := admin.Exec(fmt.Sprintf(`CREATE DATABASE %q`, dbName)); err != nil {
			panic(err)
		}
	}

	db := sqlx.MustConnect("pgx", dbDSN(dbUser, dbName))
	if err := applyMigrations(db, migrationsDir); err != nil {
		panic(err)
	}
	return db
}

func applyMigrations(db *sqlx.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", file, err)
		}
	}
	return nil
}

// DropDatabaseWithoutExitingWithPkg drops the test database.
func DropDatabaseWithoutExitingWithPkg(dbUser, dbName, pkg string) {
	dropDB(dbUser, ResolveDBName(dbName, pkg))
}

func dropDB(dbUser, dbName string) {
	admin := sqlx.MustConnect("pgx", adminDSN(dbUser))
	defer admin.Close()

	_, _ = admin.Exec(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, dbName)
	_, _ = admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %q`, dbName))
}
