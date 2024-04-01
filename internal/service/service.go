//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package service contains the business logic for the minder services.
package service

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/stacklok/minder/internal/auth"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/reconcilers"
)

// AllInOneServerService is a helper function that starts the gRPC and HTTP servers,
// the eventer, aggregator, the executor, and the reconciler.
func AllInOneServerService(
	ctx context.Context,
	cfg *serverconfig.Config,
	store db.Store,
	vldtr auth.JwtValidator,
	serverOpts []controlplane.ServerOption,
	executorOpts []engine.ExecutorOption,
	reconcilerOpts []reconcilers.ReconcilerOption,

) error {
	errg, ctx := errgroup.WithContext(ctx)

	evt, err := events.Setup(ctx, &cfg.Events)
	if err != nil {
		return fmt.Errorf("unable to setup eventer: %w", err)
	}

	s, err := controlplane.NewServer(
		store, evt, cfg, vldtr,
		serverOpts...,
	)
	if err != nil {
		return fmt.Errorf("unable to create server: %w", err)
	}

	aggr := eea.NewEEA(store, evt, &cfg.Events.Aggregator)

	// consume flush-all events
	evt.ConsumeEvents(aggr)

	// prepend the aggregator to the executor options
	executorOpts = append([]engine.ExecutorOption{engine.WithMiddleware(aggr.AggregateMiddleware)},
		executorOpts...)

	exec, err := engine.NewExecutor(ctx, store, &cfg.Auth, &cfg.Provider, evt, executorOpts...)
	if err != nil {
		return fmt.Errorf("unable to create executor: %w", err)
	}

	evt.ConsumeEvents(exec)

	rec, err := reconcilers.NewReconciler(store, evt, &cfg.Auth, &cfg.Provider, reconcilerOpts...)
	if err != nil {
		return fmt.Errorf("unable to create reconciler: %w", err)
	}

	evt.ConsumeEvents(rec)

	// Start the gRPC and HTTP server in separate goroutines
	errg.Go(func() error {
		return s.StartGRPCServer(ctx)
	})

	errg.Go(func() error {
		return s.StartHTTPServer(ctx)
	})

	errg.Go(func() error {
		defer evt.Close()
		return evt.Run(ctx)
	})

	// Wait for event handlers to start running
	<-evt.Running()

	// Flush all cache
	if err := aggr.FlushAll(ctx); err != nil {
		return fmt.Errorf("error flushing cache: %w", err)
	}

	// Wait for all entity events to be executed
	exec.Wait()

	return errg.Wait()
}
