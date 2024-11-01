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

var (
	// ErrQuerierMissing is returned when the querier is not initialized
	ErrQuerierMissing = fmt.Errorf("querier is missing, possibly due to closed or committed transaction")
	// ErrProfileSvcMissing is returned when the profile service is not initialized
	ErrProfileSvcMissing = fmt.Errorf("profile service is missing")
	// ErrRuleSvcMissing is returned when the rule service is not initialized
	ErrRuleSvcMissing = fmt.Errorf("rule service is missing")
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
// Calling Commit() or Cancel() is necessary after using the querier.
// Note that they act as a destructor for the transaction
// so any further calls using the querier will result in an error.
type Querier interface {
	ProjectHandlers
	RuleTypeHandlers
	ProfileHandlers
	BundleHandlers
	Commit() error
	Cancel() error
}

// querierType represents the database querier
type querierType struct {
	store      db.Store
	tx         *sql.Tx
	querier    db.ExtendQuerier
	ruleSvc    ruletypes.RuleTypeService
	profileSvc profiles.ProfileService
}

// Closer is a function that closes the database connection
type Closer func()

// Ensure Type implements the Querier interface
var _ Querier = (*querierType)(nil)

// New creates a new instance of the querier
func New(ctx context.Context, config *server.Config) (Store, Closer, error) {
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
	return &querierType{
		store:      store,
		querier:    store, // use store by default
		ruleSvc:    ruletypes.NewRuleTypeService(),
		profileSvc: profiles.NewProfileService(evt, selectors.NewEnv()),
	}, dbCloser, nil
}

// BeginTx begins a new transaction
func (q *querierType) BeginTx() (Querier, error) {
	tx, err := q.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	return &querierType{
		store:      q.store,
		querier:    q.store.GetQuerierWithTransaction(tx),
		ruleSvc:    q.ruleSvc,
		profileSvc: q.profileSvc,
		tx:         tx,
	}, nil
}

// Commit commits the transaction
func (q *querierType) Commit() error {
	if q.tx != nil {
		err := q.store.Commit(q.tx)
		// Clear the transaction and the querier
		q.tx = nil
		q.querier = nil
		return err
	}
	return fmt.Errorf("no transaction to commit")
}

// Cancel cancels the transaction
func (q *querierType) Cancel() error {
	if q.tx != nil {
		err := q.store.Rollback(q.tx)
		// Clear the transaction and the querier
		q.tx = nil
		q.querier = nil
		return err
	}
	return fmt.Errorf("no transaction to cancel")
}

// initDatabase function initializes the database connection
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
