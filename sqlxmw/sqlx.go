// Copyright 2017 Jean Niklas L'orange.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlxmw

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/hypirion/go-mw"

	"github.com/jmoiron/sqlx"
)

// WrapParams is the set of input parameters
type WrapParams struct {
	// DB is the database to connect to. Has to be nonnil.
	DB *sqlx.DB
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
		p := params
		p.wrapped = h // pass by value makes this ok
		return p.handle
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
	db     *sqlx.DB
	dbopts *sql.TxOptions
	tx     *sqlx.Tx
}

// GetRawDB returns the raw database from the provided context, or nil if it
// does not exist. Prefer GetTx when you can.
func GetRawDB(ctx context.Context, index int) (*sqlx.DB, error) {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil, &mw.ErrMissingContextValue{fmt.Sprintf("go-mw/sqlx.DB[%d]", index)}
	}
	return val.(*contextValue).db, nil
}

// GetTx returns a transaction for the provided context. Successive calls will
// return the same transaction, unless the transaction initialisation failed.
func GetTx(ctx context.Context) (*sqlx.Tx, error) {
	return GetIndexedTx(ctx, 0)
}

// GetIndexedTx works like GetTx, except that it provides the option to specify
// which database to get a transaction from.
func GetIndexedTx(ctx context.Context, index int) (*sqlx.Tx, error) {
	val := ctx.Value(contextKey(index))
	if val == nil {
		return nil, &mw.ErrMissingContextValue{fmt.Sprintf("go-mw/sqlx.Tx[%d]", index)}
	}
	ctxval := val.(*contextValue)
	if ctxval.tx != nil {
		return ctxval.tx, nil
	}
	tx, err := ctxval.db.BeginTxx(ctx, ctxval.dbopts)
	if err == nil {
		ctxval.tx = tx
	}
	return tx, err
}
