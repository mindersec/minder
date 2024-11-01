// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/engine/selectors"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/profiles"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// Store interface provides functions to execute db queries and transactions
type Store interface {
	ProjectHandlers
	RuleTypeHandlers
	ProfileHandlers
	BundleHandlers
	BeginTx() (Querier, error)
}

// Querier interface provides functions to interact with the Minder database using transactions
type Querier interface {
	ProjectHandlers
	RuleTypeHandlers
	ProfileHandlers
	BundleHandlers
	CommitTx() error
	CancelTx() error
}

// Type represents the querier type
type Type struct {
	store      db.Store
	tx         *sql.Tx
	querier    db.ExtendQuerier
	ruleSvc    ruletypes.RuleTypeService
	profileSvc profiles.ProfileService
}

// Ensure Type implements the Querier interface
var _ Querier = (*Type)(nil)

// New creates a new instance of the querier
func New(ctx context.Context, config *server.Config) (Store, func(), error) {
	// Initialize the database
	store, dbCloser, err := initDatabase(ctx, config)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup database: %w", err)
	}
	// Get a watermill event handler
	evt, err := eventer.New(ctx, nil, &config.Events)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup eventer: %w", err)
	}
	// Return the new Type
	return &Type{
		store:      store,
		querier:    store, // use store by default
		ruleSvc:    ruletypes.NewRuleTypeService(),
		profileSvc: profiles.NewProfileService(evt, selectors.NewEnv()),
	}, dbCloser, nil
}

// BeginTx begins a new transaction
func (t *Type) BeginTx() (Querier, error) {
	tx, err := t.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	return &Type{
		store:      t.store,
		querier:    t.store.GetQuerierWithTransaction(tx),
		ruleSvc:    t.ruleSvc,
		profileSvc: t.profileSvc,
		tx:         tx,
	}, nil
}

// CommitTx commits the transaction
func (t *Type) CommitTx() error {
	if t.tx != nil {
		err := t.store.Commit(t.tx)
		// Clear the transaction and reset the querier to the store
		t.tx = nil
		t.querier = t.store
		return err
	}
	return fmt.Errorf("no transaction to commit")
}

// CancelTx cancels the transaction
func (t *Type) CancelTx() error {
	if t.tx != nil {
		err := t.store.Rollback(t.tx)
		// Clear the transaction and reset the querier to the store
		t.tx = nil
		t.querier = t.store
		return err
	}
	return fmt.Errorf("no transaction to cancel")
}

// initDb function initializes the database connection and transaction details, if needed
func initDatabase(ctx context.Context, cfg *server.Config) (db.Store, func(), error) {
	zerolog.Ctx(ctx).Debug().
		Str("name", cfg.Database.Name).
		Str("host", cfg.Database.Host).
		Str("user", cfg.Database.User).
		Str("ssl_mode", cfg.Database.SSLMode).
		Int("port", cfg.Database.Port).
		Msg("connecting to minder database")
	// Get a database connection
	dbConn, _, err := cfg.Database.GetDBConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	closer := func() {
		err := dbConn.Close()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error closing database connection")
		}
	}
	return db.NewStore(dbConn), closer, nil
}
