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
	Get(owner, token string, provider db.ProviderTrait) (provinfv1.REST, bool)
	Set(owner, token string, provider db.ProviderTrait, rest provinfv1.REST)

	// Close stops the eviction routine and disallows setting new entries
	// cache is not cleared, getting existing entries is still allowed
	Close()
}

type restClientCache struct {
	cache          *xsyncv3.MapOf[string, cacheEntry]
	evictionTime   time.Duration
	ctx            context.Context
	evictionTicker *time.Ticker
	closeChan      chan struct{}
	closeOnce      sync.Once
}

type cacheEntry struct {
	value      provinfv1.REST
	expiration time.Time
}

var _ RestClientCache = (*restClientCache)(nil)

// NewRestClientCache creates a new REST client cache
func NewRestClientCache(ctx context.Context) RestClientCache {
	// We check for expired entries every half of the eviction time
	evictionDuration := defaultCacheExpiration / 2
	c := &restClientCache{
		cache:          xsyncv3.NewMapOf[string, cacheEntry](),
		evictionTime:   defaultCacheExpiration,
		ctx:            ctx,
		evictionTicker: time.NewTicker(evictionDuration),
		closeChan:      make(chan struct{}),
	}

	go c.evictExpiredEntriesRoutine(ctx)
	return c
}

func (r *restClientCache) Get(owner, token string, provider db.ProviderTrait) (provinfv1.REST, bool) {
	key := r.getKey(owner, token, provider)
	entry, ok := r.cache.Load(key)
	if !ok || time.Now().After(entry.expiration) {
		// Entry doesn't exist or has expired
		return nil, false
	}
	return entry.value, true
}

func (r *restClientCache) Set(owner, token string, provider db.ProviderTrait, rest provinfv1.REST) {
	select {
	case <-r.ctx.Done():
		return
	case <-r.closeChan:
		return
	default:
	}

	key := r.getKey(owner, token, provider)
	r.cache.Store(key, cacheEntry{
		value:      rest,
		expiration: time.Now().Add(r.evictionTime), // Set expiration time
	})
}

func (r *restClientCache) Close() {
	r.closeOnce.Do(func() {
		defer r.evictionTicker.Stop()
		close(r.closeChan)
	})
}

func (_ *restClientCache) getKey(owner, token string, provider db.ProviderTrait) string {
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
	defer r.Close()
	for {
		select {
		case <-ctx.Done():
			return
		// If ctx is not done but the ticker has expired, this would allow the goroutine to stop
		// Stopping the ticker does not close the C channel, so we need to check for the closeChan
		case <-r.closeChan:
			return
		case <-r.evictionTicker.C:
			r.evictExpiredEntries()
		}
	}
}
