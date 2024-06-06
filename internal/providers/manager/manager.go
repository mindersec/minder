// Copyright 2024 Stacklok, Inc.
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

// Package manager contains logic for creating Provider instances
package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"dario.cat/mergo"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ProviderManager encapsulates operations for manipulating Provider instances
type ProviderManager interface {
	// CreateFromConfig creates a new Provider instance in the database with a given configuration or the provider default
	CreateFromConfig(
		ctx context.Context, providerClass db.ProviderClass, projectID uuid.UUID, name string, config json.RawMessage,
	) (*db.Provider, error)
	// InstantiateFromID creates the provider from the Provider's UUID
	InstantiateFromID(ctx context.Context, providerID uuid.UUID) (v1.Provider, error)
	// InstantiateFromNameProject creates the provider using the provider's name and
	// project hierarchy.
	InstantiateFromNameProject(ctx context.Context, name string, projectID uuid.UUID) (v1.Provider, error)
	// BulkInstantiateByTrait instantiates multiple providers in the
	// project hierarchy. Providers are filtered by trait, and optionally by
	// name (empty name string means no filter by name).
	// To preserve compatibility with behaviour expected by the API, if a
	// provider cannot be instantiated, it will not cause the method to error
	// out, instead a list of failed provider names will be returned.
	BulkInstantiateByTrait(
		ctx context.Context,
		projectID uuid.UUID,
		trait db.ProviderType,
		name string,
	) (map[string]v1.Provider, []string, error)
	// DeleteByID deletes the specified instance of the Provider, and
	// carries out any cleanup needed.
	DeleteByID(ctx context.Context, providerID uuid.UUID, projectID uuid.UUID) error
	// DeleteByName deletes the specified instance of the Provider, and
	// carries out any cleanup needed.
	// Deletion will only occur if the provider is in the specified project -
	// it will not attempt to find a provider elsewhere in the hierarchy.
	DeleteByName(ctx context.Context, name string, projectID uuid.UUID) error
	// PatchProviderConfig updates the configuration of the specified provider with the specified patch.
	// All keys in the configMap will overwrite the fields in the provider config.
	PatchProviderConfig(ctx context.Context, providerName string, projectID uuid.UUID, configPatch map[string]any) error
}

// ProviderClassManager describes an interface for creating instances of a
// specific Provider class. The idea is that ProviderManager determines the
// class of the Provider, and delegates to the appropraite ProviderClassManager
type ProviderClassManager interface {
	providerClassAuthManager

	GetConfig(ctx context.Context, class db.ProviderClass, userConfig json.RawMessage) (json.RawMessage, error)
	// MarshallConfig validates the config and marshalls it into a format that can be stored in the database
	MarshallConfig(ctx context.Context, class db.ProviderClass, config json.RawMessage) (json.RawMessage, error)
	// Build creates an instance of Provider based on the config in the DB
	Build(ctx context.Context, config *db.Provider) (v1.Provider, error)
	// Delete deletes an instance of this provider
	Delete(ctx context.Context, config *db.Provider) error
	// GetSupportedClasses lists the types of Provider class which this manager
	// can produce.
	GetSupportedClasses() []db.ProviderClass
}

type classTracker struct {
	classManagers map[db.ProviderClass]ProviderClassManager
}

func (p *classTracker) getClassManager(class db.ProviderClass) (ProviderClassManager, error) {
	manager, ok := p.classManagers[class]
	if !ok {
		return nil, fmt.Errorf("unexpected provider class: %s", class)
	}
	return manager, nil
}

func newClassTracker(
	classManagers ...ProviderClassManager,
) (*classTracker, error) {
	classes := make(map[db.ProviderClass]ProviderClassManager)

	for _, factory := range classManagers {
		supportedClasses := factory.GetSupportedClasses()
		// Sanity check: make sure we don't inadvertently register the same
		// class to two different factories, and that the manager has at least
		// one type registered
		if len(supportedClasses) == 0 {
			return nil, errors.New("provider class manager has no registered classes")
		}
		for _, class := range supportedClasses {
			_, ok := classes[class]
			if ok {
				return nil, fmt.Errorf("attempted to register class %s more than once", class)
			}
			classes[class] = factory
		}
	}

	return &classTracker{
		classManagers: classes,
	}, nil
}

type providerManager struct {
	classTracker
	store providers.ProviderStore
}

// NewProviderManager creates a new instance of ProviderManager
func NewProviderManager(
	store providers.ProviderStore,
	classManagers ...ProviderClassManager,
) (ProviderManager, error) {
	classes, err := newClassTracker(classManagers...)
	if err != nil {
		return nil, fmt.Errorf("error creating class tracker: %w", err)
	}

	return &providerManager{
		classTracker: *classes,
		store:        store,
	}, nil
}

func (p *providerManager) CreateFromConfig(
	ctx context.Context, providerClass db.ProviderClass, projectID uuid.UUID, name string, config json.RawMessage,
) (*db.Provider, error) {
	manager, err := p.getClassManager(providerClass)
	if err != nil {
		return nil, fmt.Errorf("error getting class manager: %w", err)
	}

	provConfig, err := manager.GetConfig(ctx, providerClass, config)
	if err != nil {
		return nil, fmt.Errorf("error getting provider config: %w", err)
	}

	marshalledConfig, err := manager.MarshallConfig(ctx, providerClass, provConfig)
	if err != nil {
		return nil, providers.NewErrProviderInvalidConfig(err.Error())
	}

	return p.store.Create(ctx, providerClass, name, projectID, marshalledConfig)
}

func (p *providerManager) InstantiateFromID(ctx context.Context, providerID uuid.UUID) (v1.Provider, error) {
	config, err := p.store.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.buildFromDBRecord(ctx, config)
}

func (p *providerManager) InstantiateFromNameProject(ctx context.Context, name string, projectID uuid.UUID) (v1.Provider, error) {
	config, err := p.store.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.buildFromDBRecord(ctx, config)
}

func (p *providerManager) BulkInstantiateByTrait(
	ctx context.Context,
	projectID uuid.UUID,
	trait db.ProviderType,
	name string,
) (map[string]v1.Provider, []string, error) {
	providerConfigs, err := p.store.GetByTraitInHierarchy(ctx, projectID, name, trait)
	if err != nil {
		return nil, nil, fmt.Errorf("error retrieving db records: %w", err)
	}

	result := make(map[string]v1.Provider, len(providerConfigs))
	failedProviders := []string{}
	for _, config := range providerConfigs {
		provider, err := p.buildFromDBRecord(ctx, &config)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msgf("error while instantiating provider %s", config.ID)
			failedProviders = append(failedProviders, config.Name)
			continue
		}
		result[config.Name] = provider
	}

	return result, failedProviders, nil
}

func (p *providerManager) DeleteByID(ctx context.Context, providerID uuid.UUID, projectID uuid.UUID) error {
	config, err := p.store.GetByIDProject(ctx, providerID, projectID)
	if err != nil {
		return fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.deleteByRecord(ctx, config)
}

func (p *providerManager) DeleteByName(ctx context.Context, name string, projectID uuid.UUID) error {
	config, err := p.store.GetByNameInSpecificProject(ctx, projectID, name)
	if err != nil {
		return fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.deleteByRecord(ctx, config)
}

func (p *providerManager) PatchProviderConfig(
	ctx context.Context, providerName string, projectID uuid.UUID, configPatch map[string]any,
) error {
	dbProvider, err := p.store.GetByNameInSpecificProject(ctx, projectID, providerName)
	if err != nil {
		return fmt.Errorf("error retrieving db record: %w", err)
	}

	var originalConfig map[string]any

	if err := json.Unmarshal(dbProvider.Definition, &originalConfig); err != nil {
		return fmt.Errorf("error unmarshalling provider config: %w", err)
	}

	err = mergo.Map(&originalConfig, configPatch, mergo.WithOverride)
	if err != nil {
		return fmt.Errorf("error merging provider config: %w", err)
	}

	mergedJSON, err := json.Marshal(originalConfig)
	if err != nil {
		return fmt.Errorf("error marshalling provider config: %w", err)
	}

	return p.store.Update(ctx, dbProvider.ID, dbProvider.ProjectID, mergedJSON)
}

func (p *providerManager) deleteByRecord(ctx context.Context, config *db.Provider) error {
	manager, err := p.getClassManager(config.Class)
	if err != nil {
		return err
	}

	// carry out provider-specific cleanup
	if err = manager.Delete(ctx, config); err != nil {
		return fmt.Errorf("error while cleaning up provider: %w", err)
	}

	// finally: delete from the database
	if err = p.store.Delete(ctx, config.ID, config.ProjectID); err != nil {
		return fmt.Errorf("error while deleting provider from DB: %w", err)
	}
	return nil
}

func (p *providerManager) buildFromDBRecord(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	manager, err := p.getClassManager(config.Class)
	if err != nil {
		return nil, err
	}
	return manager.Build(ctx, config)
}
