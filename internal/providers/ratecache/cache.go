// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ratecache provides a cache for the REST clients
package ratecache

import (
	"context"
	"time"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util/cache"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

const (
	defaultCacheExpiration = 30 * time.Minute
)

// RestClientCache is the interface for the REST client cache
type RestClientCache interface {
	Get(owner, token string, provider db.ProviderType) (provinfv1.REST, bool)
	Set(owner, token string, provider db.ProviderType, rest provinfv1.REST)

	// Close stops the eviction routine and disallows setting new entries
	// cache is not cleared, getting existing entries is still allowed
	Close()
}

type restClientCache struct {
	*cache.ExpiringCache[provinfv1.REST]
}

var _ RestClientCache = (*restClientCache)(nil)

// NewRestClientCache creates a new REST client cache
func NewRestClientCache(ctx context.Context) RestClientCache {
	c := &restClientCache{
		ExpiringCache: cache.NewExpiringCache[provinfv1.REST](ctx, &cache.ExpiringCacheConfig{
			EvictionTime: defaultCacheExpiration,
		}),
	}

	return c
}

func (r *restClientCache) Get(owner, token string, provider db.ProviderType) (provinfv1.REST, bool) {
	key := r.getKey(owner, token, provider)
	return r.ExpiringCache.Get(key)
}

func (r *restClientCache) Set(owner, token string, provider db.ProviderType, rest provinfv1.REST) {
	key := r.getKey(owner, token, provider)
	r.ExpiringCache.Set(key, rest)
}

func (*restClientCache) getKey(owner, token string, provider db.ProviderType) string {
	return owner + token + string(provider)
}

// NoopRestClientCache is a no-op implementation of the interface used for testing
type NoopRestClientCache struct{}

// Get always returns nil, false
func (*NoopRestClientCache) Get(_, _ string, _ db.ProviderType) (provinfv1.REST, bool) {
	return nil, false
}

// Set does nothing
func (*NoopRestClientCache) Set(_, _ string, _ db.ProviderType, _ provinfv1.REST) {
	// no-op
}

// Close does nothing
func (*NoopRestClientCache) Close() {
	// no-op
}
