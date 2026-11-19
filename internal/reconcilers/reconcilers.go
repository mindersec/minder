// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reconcilers contains the reconcilers for the various types of
// entities in minder.
package reconcilers

import (
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	entityService "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/repositories"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// Reconciler is a helper that reconciles entities
type Reconciler struct {
	store           db.Store
	evt             interfaces.Publisher
	crypteng        crypto.Engine
	providerManager manager.ProviderManager
	repos           repositories.RepositoryService
	entities        entityService.EntityService
}

// NewReconciler creates a new reconciler object
func NewReconciler(
	store db.Store,
	evt interfaces.Publisher,
	cryptoEngine crypto.Engine,
	providerManager manager.ProviderManager,
	repositoryService repositories.RepositoryService,
	entities entityService.EntityService,
) (*Reconciler, error) {
	return &Reconciler{
		store:           store,
		evt:             evt,
		crypteng:        cryptoEngine,
		providerManager: providerManager,
		repos:           repositoryService,
		entities:        entities,
	}, nil
}

// Register implements the Consumer interface.
func (r *Reconciler) Register(reg interfaces.Registrar) {
	reg.Register(constants.TopicQueueReconcileRepoInit, r.handleRepoReconcilerEvent)
	reg.Register(constants.TopicQueueReconcileProfileInit, r.handleProfileInitEvent)
	reg.Register(constants.TopicQueueReconcileEntityDelete, r.handleEntityDeleteEvent)
	reg.Register(constants.TopicQueueReconcileEntityAdd, r.handleEntityAddEvent)
}
