package mwsql

// ^ uh, not to be confused with mysql

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/hypirion/go-mw"
)

// TODO: Circuit breaking
// Also TODO: sqlx when it has ported over context

// WrapParams is the set of input parameters
type WrapParams struct {
	// DB is the database to connect to. Has to be nonnil.
	DB *sql.DB
	// Options when starting a transactional
	DBOpts *sql.TxOptions
	// Index is the index of this database. This argument is optional and is not
	// necessary unless you need to connect to multiple different SQL databases.
	Index int

	// lazy me just using WrapParams as value in functions to be able to use a
	// single struct instead of two
	wrapped mw.Handler
}

// WrapSQL returns a middleware which provides the context with a transaction
// and a database.
func WrapSQL(params WrapParams) mw.Middleware {
	return func(h mw.Handler) mw.Handler {
		params.wrapped = h // pass by value makes this ok
		return params.handle
	}
}

func (wp WrapParams) handle(req *http.Request) (resp *mw.Response, err error) {
	val := contextValue{db: wp.DB, dbopts: wp.DBOpts}
	paniced := true
	req = mw.WithContextValue(req, contextKey(wp.Index), &val)
	defer func() {
		if val.tx != nil {
			// did we error, or did we panic?
			if paniced || err != nil {
				val.tx.Tx.Rollback()
				// TODO: Support error message handling in case rollback fails (?)
			} else {
				err = val.tx.Tx.Commit()
			}
		}
	}()
	resp, err = wp.wrapped(req)
	paniced = false
	return
}

// To avoid accidental overwrites
type contextKey int

type contextValue struct {
	db     *sql.DB
	dbopts *sql.TxOptions
	tx     *Tx
}

// Tx is like sql.Tx, but rollbacks/commits are not provided, and the context is
// implicitly provided in all calls. You can grab the underlying Tx if
// necessary, but it is not recommended and it may be nil. You can also grab the
// context and modify it if necessary (Its parent context should always be the
// original ctx).
type Tx struct {
	Ctx       context.Context
	Tx        *sql.Tx
	initError error
}

type unusableResult struct {
	err error
}

func (ur unusableResult) LastInsertId() (int64, error) {
	return 0, ur.err
}

func (ur unusableResult) RowsAffected() (int64, error) {
	return 0, ur.err
}

func (ur unusableResult) Scan(dest ...interface{}) error {
	return ur.err
}

// InitErr returns the initialisation error of this transaction, if there are
// any.
func (tx *Tx) InitErr() error {
	return tx.initError
}

// Exec executes a query that doesn't return rows. For example: an INSERT and
// UPDATE.
func (tx *Tx) Exec(query string, args ...interface{}) (sql.Result, error) {
	if tx.initError != nil {
		return unusableResult{tx.initError}, tx.initError
	}
	return tx.Tx.ExecContext(tx.Ctx, query, args...)
}

// Prepare creates a prepared statement for use within a transaction.
//
// The returned statement operates within the transaction and can no longer be
// used once the transaction has been committed or rolled back.
//
// To use an existing prepared statement on this transaction, see Tx.Stmt.
func (tx *Tx) Prepare(query string) (*sql.Stmt, error) {
	if tx.initError != nil {
		return nil, tx.initError
	}
	return tx.Tx.PrepareContext(tx.Ctx, query)
}

// Query executes a query that returns rows, typically a SELECT.
func (tx *Tx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if tx.initError != nil {
		return nil, tx.initError
	}
	return tx.Tx.QueryContext(tx.Ctx, query, args...)
}

// Row is like *sql.Row, but is instead an interface
type Row interface {
	Scan(dest ...interface{}) error
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan
// method is called.
func (tx *Tx) QueryRow(query string, args ...interface{}) Row {
	if tx.initError != nil {
		return &unusableResult{tx.initError}
	}
	return tx.Tx.QueryRowContext(tx.Ctx, query, args...)
}

// Stmt returns a transaction-specific prepared statement from an existing
// statement. This will only error if the transaction has trouble with
// initialisation.
func (tx *Tx) Stmt(stmt *sql.Stmt) (*sql.Stmt, error) {
	if tx.initError != nil {
		return nil, tx.initError
	}
	return tx.Tx.StmtContext(tx.Ctx, stmt), nil
}

// GetRawDB returns the raw database from the provided context, or nil if it
// does not exist. Prefer GetTx when you can.
func GetRawDB(ctx context.Context, index int) *sql.DB {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil
	}
	return val.(*contextValue).db
}

// GetTx returns a transaction for the provided context, or nil if the context
// is not attached to any database. If the transaction initialisation failed,
// then all database calls will fail. Successive calls will return the same
// transaction, unless the transaction initialisation failed.
func GetTx(ctx context.Context) *Tx {
	return GetIndexedTx(ctx, 0)
}

// GetIndexedTx works like GetTx, except that it provides the option to specify
// which database to get a transaction from.
func GetIndexedTx(ctx context.Context, index int) *Tx {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil
	}
	ctxval := val.(*contextValue)
	if ctxval.tx != nil {
		return ctxval.tx
	}
	tx, err := ctxval.db.BeginTx(ctx, ctxval.dbopts)
	ctxval.tx = &Tx{ctx, tx, err}
	return ctxval.tx
}
