// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQueries_LockIfThresholdNotExceeded(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	project := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, project.ID)
	repo := createRandomRepository(t, project.ID, prov)

	threshold := 1
	concurrentCalls := 10

	// waitgroup
	var wg sync.WaitGroup
	var queueCount atomic.Int32
	var effectiveFlush atomic.Int32

	for i := 0; i < concurrentCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testQueries.LockIfThresholdNotExceeded(context.Background(), LockIfThresholdNotExceededParams{
				Entity:           EntitiesRepository,
				Interval:         fmt.Sprintf("%d", threshold),
				ProjectID:        project.ID,
				EntityInstanceID: repo.ID,
			})

			if err != nil && errors.Is(err, sql.ErrNoRows) {
				t.Log("lock had been acquired. adding to queue")
				// count the number of times we've been queued
				queueCount.Add(1)

				_, err := testQueries.EnqueueFlush(context.Background(), EnqueueFlushParams{
					Entity:           EntitiesRepository,
					ProjectID:        project.ID,
					EntityInstanceID: repo.ID,
				})
				if err == nil {
					effectiveFlush.Add(1)
				}
			} else if err != nil {
				assert.NoError(t, err, "expected no error")
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, concurrentCalls-1, int(queueCount.Load()), "expected all but one to be queued")
	assert.Equal(t, 1, int(effectiveFlush.Load()), "expected only one flush to be queued")

	t.Log("sleeping for threshold")
	time.Sleep(time.Duration(threshold) * time.Second)

	t.Log("Attempting to acquire lock now that threshold has passed")

	_, err := testQueries.LockIfThresholdNotExceeded(context.Background(), LockIfThresholdNotExceededParams{
		Entity:           EntitiesRepository,
		Interval:         fmt.Sprintf("%d", threshold),
		EntityInstanceID: repo.ID,
	})

	assert.NoError(t, err, "expected no error")
}
