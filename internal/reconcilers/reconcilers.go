// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reconcilers contains the reconcilers for the various types of
// entities in minder.
package reconcilers

import (
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/repositories"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// Reconciler is a helper that reconciles entities
type Reconciler struct {
	store           db.Store
	evt             interfaces.Publisher
	crypteng        crypto.Engine
	providerManager manager.ProviderManager
	repos           repositories.RepositoryService
}

// NewReconciler creates a new reconciler object
func NewReconciler(
	store db.Store,
	evt interfaces.Publisher,
	cryptoEngine crypto.Engine,
	providerManager manager.ProviderManager,
	repositoryService repositories.RepositoryService,
) (*Reconciler, error) {
	return &Reconciler{
		store:           store,
		evt:             evt,
		crypteng:        cryptoEngine,
		providerManager: providerManager,
		repos:           repositoryService,
	}, nil
}

// Register implements the Consumer interface.
func (r *Reconciler) Register(reg interfaces.Registrar) {
	reg.Register(events.TopicQueueReconcileRepoInit, r.handleRepoReconcilerEvent)
	reg.Register(events.TopicQueueReconcileProfileInit, r.handleProfileInitEvent)
	reg.Register(events.TopicQueueReconcileEntityDelete, r.handleEntityDeleteEvent)
	reg.Register(events.TopicQueueReconcileEntityAdd, r.handleEntityAddEvent)
}
