// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ratecache provides a cache for the REST clients
package ratecache

import (
	"context"
	"sync"
	"time"

	xsyncv3 "github.com/puzpuzpuz/xsync/v3"

	"github.com/stacklok/minder/internal/db"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	defaultCacheExpiration = 30 * time.Minute
)

// RestClientCache is the interface for the REST client cache
type RestClientCache interface {
	Get(owner, token string, provider db.ProviderType) (provinfv1.REST, bool)
	Set(owner, token string, provider db.ProviderType, rest provinfv1.REST)

	// Wait waits for the background eviction routine to finish
	Wait()
}

type restClientCache struct {
	cache                *xsyncv3.MapOf[string, cacheEntry]
	evictionTime         time.Duration
	ctx                  context.Context
	wgBackgroundEviction sync.WaitGroup
}

type cacheEntry struct {
	value      provinfv1.REST
	expiration time.Time
}

var _ RestClientCache = (*restClientCache)(nil)

// NewRestClientCache creates a new REST client cache
func NewRestClientCache(ctx context.Context) RestClientCache {
	c := &restClientCache{
		cache:        xsyncv3.NewMapOf[string, cacheEntry](),
		evictionTime: defaultCacheExpiration,
		ctx:          ctx,
	}

	c.wgBackgroundEviction.Add(1)
	go c.evictExpiredEntriesRoutine(ctx)
	return c
}

func (r *restClientCache) Get(owner, token string, provider db.ProviderType) (provinfv1.REST, bool) {
	key := r.getKey(owner, token, provider)
	entry, ok := r.cache.Load(key)
	if !ok || time.Now().After(entry.expiration) {
		// Entry doesn't exist or has expired
		return nil, false
	}
	return entry.value, true
}

func (r *restClientCache) Set(owner, token string, provider db.ProviderType, rest provinfv1.REST) {
	// If the context has been cancelled, don't allow setting new entries
	if r.ctx.Err() != nil {
		return
	}

	key := r.getKey(owner, token, provider)
	r.cache.Store(key, cacheEntry{
		value:      rest,
		expiration: time.Now().Add(r.evictionTime), // Set expiration time
	})
}

func (r *restClientCache) Wait() {
	r.wgBackgroundEviction.Wait()
}

func (_ *restClientCache) getKey(owner, token string, provider db.ProviderType) string {
	return owner + token + string(provider)
}

func (r *restClientCache) evictExpiredEntries() {
	now := time.Now()
	r.cache.Range(func(key string, entry cacheEntry) bool {
		if now.After(entry.expiration) {
			r.cache.Delete(key)
		}
		return true
	})
}

func (r *restClientCache) evictExpiredEntriesRoutine(ctx context.Context) {
	defer r.wgBackgroundEviction.Done()

	// We check for expired entries every half of the eviction time
	evictionDuration := r.evictionTime / 2
	ticker := time.NewTicker(evictionDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.evictExpiredEntries()
		}
	}
}
