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

package cache

import (
	"context"
	"sync"
	"time"

	xsyncv3 "github.com/puzpuzpuz/xsync/v3"
)

const (
	defaultCacheExpiration = 30 * time.Minute
)

// Expiring is an interface for an expiring cache entry
// It provides a method to get the expiration time and the value
type Expiring[T any] interface {
	Expiration() time.Time
	Value() T
}

type expiringImpl[T any] struct {
	expiration time.Time
	value      T
}

// NewExpiring creates a new Expiring entry
func NewExpiring[T any](value T, expiration time.Time) Expiring[T] {
	return &expiringImpl[T]{
		expiration: expiration,
		value:      value,
	}
}

// Expiration returns the expiration time of the entry
func (e *expiringImpl[T]) Expiration() time.Time {
	return e.expiration
}

// Value returns the value of the entry
func (e *expiringImpl[T]) Value() T {
	return e.value
}

// ExpiringCache is a cache that expires entries after a certain time
type ExpiringCache[T any] struct {
	cache          *xsyncv3.MapOf[string, Expiring[T]]
	evictionTime   time.Duration
	ctx            context.Context
	evictionTicker *time.Ticker
	closeChan      chan struct{}
	closeOnce      sync.Once
}

// ExpiringCacheConfig is the configuration for the ExpiringCache
type ExpiringCacheConfig struct {
	EvictionTime time.Duration
}

// NewExpiringCache creates a new ExpiringCache based on the configuration
// If the configuration is nil, the default eviction time is 30 minutes
func NewExpiringCache[T any](ctx context.Context, cfg *ExpiringCacheConfig) *ExpiringCache[T] {
	if cfg == nil {
		cfg = &ExpiringCacheConfig{
			EvictionTime: defaultCacheExpiration,
		}
	}

	if cfg.EvictionTime <= 0 {
		cfg.EvictionTime = defaultCacheExpiration
	}

	// We check for expired entries every half of the eviction time
	evictionDuration := cfg.EvictionTime / 2
	c := &ExpiringCache[T]{
		cache:          xsyncv3.NewMapOf[string, Expiring[T]](),
		evictionTime:   cfg.EvictionTime,
		ctx:            ctx,
		evictionTicker: time.NewTicker(evictionDuration),
		closeChan:      make(chan struct{}),
	}

	go c.evictExpiredEntriesRoutine(ctx)
	return c
}

// Get returns the value of the entry and a boolean indicating if the entry exists
func (ec *ExpiringCache[T]) Get(key string) (T, bool) {
	entry, ok := ec.cache.Load(key)
	if !ok || time.Now().After(entry.Expiration()) {
		// Entry doesn't exist or has expired
		var emptyT T
		return emptyT, false
	}
	return entry.Value(), true
}

// Set sets the value of the entry
func (ec *ExpiringCache[T]) Set(key string, value T) {
	select {
	case <-ec.ctx.Done():
		return
	case <-ec.closeChan:
		return
	default:
	}

	ec.cache.Store(key, NewExpiring(value, time.Now().Add(ec.evictionTime)))
}

// Delete deletes the entry from the cache
func (ec *ExpiringCache[T]) Delete(key string) {
	ec.cache.Delete(key)
}

// Close stops the eviction routine and disallows setting new entries
func (ec *ExpiringCache[T]) Close() {
	ec.closeOnce.Do(func() {
		defer ec.evictionTicker.Stop()
		close(ec.closeChan)
	})
}

// Size returns the number of entries in the cache. This is useful
// for testing purposes
func (ec *ExpiringCache[T]) Size() int {
	return ec.cache.Size()
}

func (ec *ExpiringCache[T]) evictExpiredEntries() {
	now := time.Now()
	ec.cache.Range(func(key string, entry Expiring[T]) bool {
		if now.After(entry.Expiration()) {
			ec.cache.Delete(key)
		}
		return true
	})
}

func (ec *ExpiringCache[T]) evictExpiredEntriesRoutine(ctx context.Context) {
	defer ec.Close()
	for {
		select {
		case <-ctx.Done():
			return
		// If ctx is not done but the ticker has expired, this would allow the goroutine to stop
		// Stopping the ticker does not close the C channel, so we need to check for the closeChan
		case <-ec.closeChan:
			return
		case <-ec.evictionTicker.C:
			ec.evictExpiredEntries()
		}
	}
}
