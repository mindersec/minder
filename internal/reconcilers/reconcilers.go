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
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/manager"
)

// Reconciler is a helper that reconciles entities
type Reconciler struct {
	store           db.Store
	evt             events.Publisher
	crypteng        crypto.Engine
	providerManager manager.ProviderManager
}

// NewReconciler creates a new reconciler object
func NewReconciler(
	store db.Store,
	evt events.Publisher,
	cryptoEngine crypto.Engine,
	providerManager manager.ProviderManager,
) (*Reconciler, error) {
	return &Reconciler{
		store:           store,
		evt:             evt,
		crypteng:        cryptoEngine,
		providerManager: providerManager,
	}, nil
}

// Register implements the Consumer interface.
func (r *Reconciler) Register(reg events.Registrar) {
	reg.Register(events.TopicQueueReconcileRepoInit, r.handleRepoReconcilerEvent)
	reg.Register(events.TopicQueueReconcileProfileInit, r.handleProfileInitEvent)
	reg.Register(events.TopicQueueReconcileEntityDelete, r.handleEntityDeleteEvent)
}
