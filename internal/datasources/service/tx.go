// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"database/sql"

	"github.com/mindersec/minder/internal/db"
)

// serviceTX is an interface that defines the methods for a service transaction.
//
// This service may be used with a pre-built transaction or without one.
// thus, we need to be able to handle both cases.
type serviceTX interface {
	Q() db.ExtendQuerier
	Commit() error
	Rollback() error
}

// This is a handy helper function to optionally begin a transaction if we've already
// got one.
func beginTx(d *dataSourceService, opts txGetter) (serviceTX, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.getTransaction() != nil {
		return &externalTX{q: opts.getTransaction()}, nil
	}

	return beginInternalTx(d, opts)
}

// builds a new transaction for the service
func beginInternalTx(d *dataSourceService, opts txGetter) (serviceTX, error) {
	tx, err := d.store.BeginTransaction()
	if err != nil {
		return nil, err
	}

	// If this is a read-only operation we can set the transaction to read-only
	// We can know this by casting to the *ReadOptions struct
	if _, ok := opts.(*ReadOptions); ok {
		if _, err := tx.Query("SET TRANSACTION READ ONLY"); err != nil {
			return nil, err
		}
	}

	return &internalTX{tx: tx, q: d.store.GetQuerierWithTransaction(tx)}, nil
}

type externalTX struct {
	q db.ExtendQuerier
}

func (e *externalTX) Q() db.ExtendQuerier {
	return e.q
}

func (_ *externalTX) Commit() error {
	return nil
}

func (_ *externalTX) Rollback() error {
	return nil
}

type internalTX struct {
	tx *sql.Tx
	q  db.ExtendQuerier
}

func (i *internalTX) Q() db.ExtendQuerier {
	return i.q
}

func (i *internalTX) Commit() error {
	return i.tx.Commit()
}

func (i *internalTX) Rollback() error {
	return i.tx.Rollback()
}
