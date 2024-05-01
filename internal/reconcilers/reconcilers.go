// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package reconcilers contains the reconcilers for the various types of
// entities in minder.
package reconcilers

import (
	gogithub "github.com/google/go-github/v61/github"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	providertelemetry "github.com/stacklok/minder/internal/providers/telemetry"
)

// Reconciler is a helper that reconciles entities
type Reconciler struct {
	store               db.Store
	evt                 events.Publisher
	crypteng            crypto.Engine
	restClientCache     ratecache.RestClientCache
	provCfg             *serverconfig.ProviderConfig
	provMt              providertelemetry.ProviderMetrics
	fallbackTokenClient *gogithub.Client
}

// ReconcilerOption is a function that modifies a reconciler
type ReconcilerOption func(*Reconciler)

// WithProviderMetrics sets the provider metrics for the reconciler
func WithProviderMetrics(mt providertelemetry.ProviderMetrics) ReconcilerOption {
	return func(r *Reconciler) {
		r.provMt = mt
	}
}

// NewReconciler creates a new reconciler object
func NewReconciler(
	store db.Store,
	evt events.Publisher,
	authCfg *serverconfig.AuthConfig,
	provCfg *serverconfig.ProviderConfig,
	restClientCache ratecache.RestClientCache,
	opts ...ReconcilerOption,
) (*Reconciler, error) {
	crypteng, err := crypto.EngineFromAuthConfig(authCfg)
	if err != nil {
		return nil, err
	}

	fallbackTokenClient := github.NewFallbackTokenClient(*provCfg)

	r := &Reconciler{
		store:               store,
		evt:                 evt,
		crypteng:            crypteng,
		provCfg:             provCfg,
		provMt:              providertelemetry.NewNoopMetrics(),
		fallbackTokenClient: fallbackTokenClient,
		restClientCache:     restClientCache,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

// Register implements the Consumer interface.
func (r *Reconciler) Register(reg events.Registrar) {
	reg.Register(events.TopicQueueReconcileRepoInit, r.handleRepoReconcilerEvent)
	reg.Register(events.TopicQueueReconcileProfileInit, r.handleProfileInitEvent)
	reg.Register(events.TopicQueueReconcileEntityDelete, r.handleEntityDeleteEvent)
}
