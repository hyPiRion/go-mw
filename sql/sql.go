package mwsql

// ^ uh, not to be confused with mysql

import (
	"context"
	"database/sql"
	"fmt"
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

// WithDB returns a middleware which provides the context with a transaction
// and a database.
func WithDB(params WrapParams) mw.Middleware {
	return func(h mw.Handler) mw.Handler {
		params.wrapped = h // pass by value makes this ok
		return params.handle
	}
}

func (wp WrapParams) handle(resp *mw.Response, req *http.Request) (err error) {
	val := contextValue{db: wp.DB, dbopts: wp.DBOpts}
	paniced := true
	req = mw.WithContextValue(req, contextKey(wp.Index), &val)
	defer func() {
		if val.tx != nil {
			// did we error, or did we panic?
			if paniced || err != nil {
				val.tx.Rollback()
				// TODO: Support error message handling in case rollback fails (?)
			} else {
				verr := val.tx.Commit()
				if verr != sql.ErrTxDone {
					err = verr
				}
			}
		}
	}()
	err = wp.wrapped(resp, req)
	paniced = false
	return
}

// To avoid accidental overwrites
type contextKey int

type contextValue struct {
	db     *sql.DB
	dbopts *sql.TxOptions
	tx     *sql.Tx
}

// GetRawDB returns the raw database from the provided context, or nil if it
// does not exist. Prefer GetTx when you can.
func GetRawDB(ctx context.Context, index int) (*sql.DB, error) {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil, &mw.ErrMissingContextValue{fmt.Sprintf("go-mw/sql.DB[%d]", index)}
	}
	return val.(*contextValue).db, nil
}

// GetTx returns a transaction for the provided context. Successive calls will
// return the same transaction, unless the transaction initialisation failed.
func GetTx(ctx context.Context) (*sql.Tx, error) {
	return GetIndexedTx(ctx, 0)
}

// GetIndexedTx works like GetTx, except that it provides the option to specify
// which database to get a transaction from.
func GetIndexedTx(ctx context.Context, index int) (*sql.Tx, error) {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil, &mw.ErrMissingContextValue{fmt.Sprintf("go-mw/sql.Tx[%d]", index)}
	}
	ctxval := val.(*contextValue)
	if ctxval.tx != nil {
		return ctxval.tx, nil
	}
	tx, err := ctxval.db.BeginTx(ctx, ctxval.dbopts)
	if err == nil {
		ctxval.tx = tx
	}
	return tx, err
}
