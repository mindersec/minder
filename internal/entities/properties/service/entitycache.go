// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"

	"github.com/mindersec/minder/internal/entities/models"
)

// This uses a persistent entity cache, that is, a cache that is not cleared.
// This is useful for a one-shot caching mechanism, where the cache is only
// cleared once the cache instance goes out of scope.
type propertyServiceWithPersistentEntityCache struct {
	// Embeds a PropertyService to provide the actual service implementation.
	PropertiesService

	entCache *xsync.MapOf[uuid.UUID, *models.EntityWithProperties]
}

// WithEntityCache wraps a PropertiesService with a persistent entity cache.
func WithEntityCache(ps PropertiesService, sizeHint int) (PropertiesService, error) {
	if ps == nil {
		return nil, fmt.Errorf("properties service is nil")
	}

	return newPropertyServiceWithPersistentEntityCache(ps, sizeHint), nil
}

func newPropertyServiceWithPersistentEntityCache(
	ps PropertiesService,
	sizeHint int,
) *propertyServiceWithPersistentEntityCache {
	opts := make([]func(*xsync.MapConfig), 0)
	if sizeHint > 0 {
		opts = append(opts, xsync.WithPresize(sizeHint))
	}
	return &propertyServiceWithPersistentEntityCache{
		PropertiesService: ps,
		entCache:          xsync.NewMapOf[uuid.UUID, *models.EntityWithProperties](opts...),
	}
}

func (ps *propertyServiceWithPersistentEntityCache) EntityWithPropertiesByID(
	ctx context.Context, entityID uuid.UUID,
	opts *CallOptions,
) (*models.EntityWithProperties, error) {
	// Check the cache first.
	if ent, ok := ps.entCache.Load(entityID); ok {
		return ent, nil
	}

	// If not in the cache, call the underlying service.
	ent, err := ps.PropertiesService.EntityWithPropertiesByID(ctx, entityID, opts)
	if err != nil {
		return nil, err
	}

	// Store the entity in the cache.
	ps.entCache.Store(entityID, ent)

	return ent, nil
}
