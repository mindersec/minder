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
	"fmt"
	"sync"
	"testing"
	"time"

	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpirationCacheCache_GetSet(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})
	defer cache.Close()

	expect := 42
	cache.Set("foo", expect)

	numOfGoroutines := 50
	var wg sync.WaitGroup

	for i := 0; i < numOfGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, ok := cache.Get("foo")
			assert.True(t, ok)
			assert.Equal(t, expect, got)
		}()
	}

	wg.Wait()
}

func TestExpirationCacheCache_evictExpiredEntriesRoutine(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})
	defer cache.Close()

	value := 42
	cache.Set("foo", value)

	op := func() error {
		_, ok := cache.Get("foo")
		if ok {
			return fmt.Errorf("entry not evicted")
		}
		return nil
	}

	err := backoffv4.Retry(op, getBackoffPolicy(t))
	assert.NoError(t, err)
}

func TestExpirationCacheCache_evictMultipleExpiredEntries(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})
	defer cache.Close()

	op := func() error {
		size := cache.Size()
		if size != 0 {
			return fmt.Errorf("cache not empty")
		}
		return nil
	}

	cache.Set("foo", 42)
	require.NoError(t, backoffv4.Retry(op, getBackoffPolicy(t)))

	cache.Set("bar", 43)
	require.NoError(t, backoffv4.Retry(op, getBackoffPolicy(t)))
}

func TestExpirationCacheCache_contextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewExpiringCache[int](ctx, &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})

	cache.Set("foo", 42)
	_, ok := cache.Get("foo")
	require.True(t, ok)

	// cancel the context, which would stop the eviction routine and close the store
	cancel()

	cache.Set("bar", 43)
	_, ok = cache.Get("bar")
	require.False(t, ok)
	require.Equal(t, 1, cache.Size())
}

func TestExpirationCacheCache_Close(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Minute,
	})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cache.evictExpiredEntriesRoutine(cache.ctx)
	}()

	cache.Set("foo", 42)
	cache.Close()

	// Ensure that the eviction routine has stopped
	wg.Wait()

	cache.Set("bar", 43)
	_, ok := cache.Get("bar")

	// Assert that setting a value after the cache has been closed does not work (non context cancellation)
	require.False(t, ok)

	// Assert that the cache has not been cleared
	require.Equal(t, 1, cache.cache.Size())

	// Assert that the cache has been stopped (closeChan has been closed)
	require.Equal(t, struct{}{}, <-cache.closeChan)
}

func TestExpirationCache_Delete(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})
	defer cache.Close()

	key := "foo"
	value := 42
	cache.Set(key, value)

	got, ok := cache.Get(key)
	require.True(t, ok)
	require.Equal(t, value, got)
	require.Equal(t, 1, cache.Size())

	// Delete a non-existent key
	cache.Delete("bar")

	// Ensure that the key still exists
	got, ok = cache.Get(key)
	require.True(t, ok)
	require.Equal(t, value, got)
	require.Equal(t, 1, cache.Size())

	cache.Delete(key)

	_, ok = cache.Get(key)
	require.False(t, ok)

	require.Equal(t, 0, cache.Size())
}

func TestExpirationCache_Size(t *testing.T) {
	t.Parallel()

	cache := NewExpiringCache[int](context.Background(), &ExpiringCacheConfig{
		EvictionTime: 10 * time.Millisecond,
	})
	defer cache.Close()

	key := "foo"
	value := 42
	cache.Set(key, value)

	size := cache.Size()
	require.Equal(t, 1, size)
}

func getBackoffPolicy(t *testing.T) backoffv4.BackOff {
	t.Helper()

	constBackoff := backoffv4.NewConstantBackOff(2 * time.Second)
	return backoffv4.WithMaxRetries(constBackoff, 5)
}
