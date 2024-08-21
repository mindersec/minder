//
// Copyright 2023 Stacklok, Inc.
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

	"github.com/google/uuid"
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
				RepositoryID:     uuid.NullUUID{UUID: repo.ID, Valid: true},
				ArtifactID:       uuid.NullUUID{},
				PullRequestID:    uuid.NullUUID{},
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
					RepositoryID:     uuid.NullUUID{UUID: repo.ID, Valid: true},
					ArtifactID:       uuid.NullUUID{},
					PullRequestID:    uuid.NullUUID{},
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
		RepositoryID:     uuid.NullUUID{UUID: repo.ID, Valid: true},
		ArtifactID:       uuid.NullUUID{},
		PullRequestID:    uuid.NullUUID{},
		Interval:         fmt.Sprintf("%d", threshold),
		EntityInstanceID: repo.ID,
	})

	assert.NoError(t, err, "expected no error")
}
