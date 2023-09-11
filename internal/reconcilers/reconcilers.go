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
// entities in mediator.
package reconcilers

import (
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/events"
)

const (
	// InternalReconcilerEventTopic is the topic for internal reconciler events
	InternalReconcilerEventTopic = "internal.repo.reconciler.event"
	// InternalPolicyInitEventTopic is the topic for internal init events
	InternalPolicyInitEventTopic = "internal.policy.init.event"
)

// Reconciler is a helper that reconciles entities
type Reconciler struct {
	store db.Store
	evt   *events.Eventer
}

// NewRecociler creates a new reconciler object
func NewRecociler(store db.Store, evt *events.Eventer) *Reconciler {
	return &Reconciler{
		store: store,
		evt:   evt,
	}
}

// Register implements the Consumer interface.
func (e *Reconciler) Register(r events.Registrar) {
	r.Register(InternalReconcilerEventTopic, e.handleRepoReconcilerEvent)
	r.Register(InternalPolicyInitEventTopic, e.handlePolicyInitEvent)
}
