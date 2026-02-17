package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// DefaultDBTimeout is the default timeout for database operations when no context is provided.
// Individual operations can override this by wrapping with DBTXWithContext.
var DefaultDBTimeout = 30 * time.Second

// DBTX is the common functions which are defined on both sqlx.DB and sqlx.Tx
type DBTX interface {
	BindNamed(query string, arg interface{}) (string, []interface{}, error)
	DriverName() string
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	MustExec(query string, args ...interface{}) sql.Result
	MustExecContext(ctx context.Context, query string, args ...interface{}) sql.Result
	NamedExec(query string, arg interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	PrepareNamed(query string) (*sqlx.NamedStmt, error)
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
	Preparex(query string) (*sqlx.Stmt, error)
	PreparexContext(ctx context.Context, query string) (*sqlx.Stmt, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryRowx(query string, args ...interface{}) *sqlx.Row
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	Rebind(query string) string
	Select(dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// DBTXWithContext wraps a DBTX with a context for timeout support.
type DBTXWithContext struct {
	DBTX
	Ctx context.Context
}

// GetDBContext extracts the context from a DBTX and returns it along with a cancel function
func GetDBContext(dbx DBTX) (context.Context, context.CancelFunc) {
	if dbxCtx, ok := dbx.(*DBTXWithContext); ok && dbxCtx.Ctx != nil {
		return dbxCtx.Ctx, func() {} // no-op cancel for already-managed context
	}
	return context.WithTimeout(context.Background(), DefaultDBTimeout)
}
