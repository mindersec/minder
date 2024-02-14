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

package ratecache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/golang/mock/gomock"
	xsyncv3 "github.com/puzpuzpuz/xsync/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/db"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
)

func TestRestClientCache_GetSet(t *testing.T) {
	t.Parallel()

	restClient := mockgh.NewMockREST(gomock.NewController(t))
	cache := newTestRestClientCache(context.Background(), t, 10*time.Minute)

	cache.Set("owner", "token", db.ProviderTypeGithub, restClient)

	numOfGoroutines := 50
	var wg sync.WaitGroup

	for i := 0; i < numOfGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recdRestClient, ok := cache.Get("owner", "token", db.ProviderTypeGithub)
			assert.True(t, ok)
			assert.Equal(t, restClient, recdRestClient)
		}()
	}

	wg.Wait()
}

func TestRestClientCache_evictExpiredEntriesRoutine(t *testing.T) {
	t.Parallel()

	restClient := mockgh.NewMockREST(gomock.NewController(t))
	cache := newTestRestClientCache(context.Background(), t, 10*time.Millisecond)

	owner := "owner"
	token := "token"
	cache.Set(owner, token, db.ProviderTypeGithub, restClient)

	op := func() error {
		_, ok := cache.Get(owner, token, db.ProviderTypeGithub)
		if ok {
			return fmt.Errorf("entry not evicted")
		}
		return nil
	}

	err := backoffv4.Retry(op, getBackoffPolicy(t))
	assert.NoError(t, err)
}

func TestRestClientCache_evictMultipleExpiredEntries(t *testing.T) {
	t.Parallel()

	restClient := mockgh.NewMockREST(gomock.NewController(t))
	cache := newTestRestClientCache(context.Background(), t, 10*time.Millisecond)

	op := func() error {
		size := cache.cache.Size()
		if size != 0 {
			return fmt.Errorf("cache not empty")
		}
		return nil
	}

	owner := "owner"
	token := "token"
	cache.Set(owner, token, db.ProviderTypeGithub, restClient)
	require.NoError(t, backoffv4.Retry(op, getBackoffPolicy(t)))

	owner2 := "owner2"
	token2 := "token2"
	cache.Set(owner2, token2, db.ProviderTypeGithub, restClient)
	require.NoError(t, backoffv4.Retry(op, getBackoffPolicy(t)))
}

func TestRestClientCache_contextCancellation(t *testing.T) {
	t.Parallel()

	restClient := mockgh.NewMockREST(gomock.NewController(t))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := newTestRestClientCache(ctx, t, 10*time.Minute)

	owner := "owner-a"
	token := "token-a"
	cache.Set(owner, token, db.ProviderTypeGithub, restClient)
	_, ok := cache.Get(owner, token, db.ProviderTypeGithub)
	require.True(t, ok)

	// cancel the context, which would stop the eviction routine
	cancel()

	owner2 := "owner2-a"
	token2 := "token2-a"
	cache.Set(owner2, token2, db.ProviderTypeGithub, restClient)
	_, ok = cache.Get(owner2, token2, db.ProviderTypeGithub)
	require.False(t, ok)
	require.Equal(t, 1, cache.cache.Size())

	cache.Wait()
}

func newTestRestClientCache(ctx context.Context, t *testing.T, evictionTime time.Duration) *restClientCache {
	t.Helper()
	c := &restClientCache{
		cache:        xsyncv3.NewMapOf[string, cacheEntry](),
		evictionTime: evictionTime,
		ctx:          ctx,
	}

	c.wgBackgroundEviction.Add(1)
	go c.evictExpiredEntriesRoutine(ctx)
	return c
}

func getBackoffPolicy(t *testing.T) backoffv4.BackOff {
	t.Helper()

	constBackoff := backoffv4.NewConstantBackOff(2 * time.Second)
	return backoffv4.WithMaxRetries(constBackoff, 5)
}
