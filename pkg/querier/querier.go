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
	Close()
	Commit() error
}

// dbData struct stores the database connection and transaction details
type dbData struct {
	store         db.Store
	tx            *sql.Tx
	querier       db.ExtendQuerier
	querierCloser func()
}

// Type struct
type Type struct {
	db         *dbData
	ruleSvc    ruletypes.RuleTypeService
	profileSvc profiles.ProfileService
}

// New returns a new Type struct. All db operations are done through this struct and applied to the database directly.
// If you want to execute db operations with a transaction, use NewWithTransaction instead.
func New(ctx context.Context, config *server.Config) (*Type, func(), error) {
	return newQuerier(ctx, config, false)
}

// NewWithTransaction returns a new Type struct. All db operations are done through this struct with a transaction
func NewWithTransaction(ctx context.Context, config *server.Config) (*Type, func(), error) {
	return newQuerier(ctx, config, true)
}

// newQuerier returns a new Type struct
func newQuerier(ctx context.Context, config *server.Config, useTransaction bool) (*Type, func(), error) {
	var err error
	ret := &Type{}
	// Initialize the database
	ret.db, err = initDb(ctx, config, useTransaction)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup database: %w", err)
	}
	// Get a watermill event handler
	evt, err := eventer.New(ctx, nil, &config.Events)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup eventer: %w", err)
	}
	// Create profile and rule type services
	ret.ruleSvc = ruletypes.NewRuleTypeService()
	ret.profileSvc = profiles.NewProfileService(evt, selectors.NewEnv())
	// Return the new Type
	return ret, ret.Close, nil
}

// Commit function
func (t *Type) Commit() error {
	if t.db.tx != nil {
		return t.db.store.Commit(t.db.tx)
	}
	return nil
}

// Close function
func (t *Type) Close() {
	t.db.querierCloser()
}

// initDb function initializes the database connection and transaction details, if needed
func initDb(ctx context.Context, cfg *server.Config, useTransaction bool) (*dbData, error) {
	ret := &dbData{}
	dbConn, _, err := cfg.Database.GetDBConnection(ctx)
	if err != nil {
		return nil, err
	}
	storeCloser := func() {
		err := dbConn.Close()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error closing database connection")
		}
	}
	ret.store = db.NewStore(dbConn)
	// Determine if we should use a transaction or not.
	// If we are using a transaction, we will create a new transaction and use that for all queries.
	// If we are not using a transaction, we will use the store directly.
	// In either case, we will close the store connection when we are done.
	if useTransaction {
		// Begin a transaction
		tx, err := ret.store.BeginTransaction()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to begin transaction")
		}
		// Store it as the querier
		ret.querier = ret.store.GetQuerierWithTransaction(tx)
		ret.querierCloser = func() {
			_ = ret.store.Rollback(tx)
			storeCloser()
		}
		ret.tx = tx
	} else {
		ret.querier = ret.store
		ret.querierCloser = storeCloser
	}
	return ret, nil
}
