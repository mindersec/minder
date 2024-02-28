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

package eea_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	providerName = "test-provider"
)

func TestAggregator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var concurrentEvents int64 = 100

	projectID, repoID := createNeededEntities(ctx, t)

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BufferSize:                     concurrentEvents,
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err)

	// we'll wait 2 seconds for the lock to be available
	var eventThreshold int64 = 2

	aggr := eea.NewEEA(testQueries, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	})

	rateLimitedMessages := newTestPubSub()
	flushedMessages := newTestPubSub()

	rateLimitedMessageTopic := t.Name()

	// This tests that the middleware works as expected
	evt.Register(rateLimitedMessageTopic, rateLimitedMessages.Add, aggr.AggregateMiddleware)

	// This tests that flushing works as expected
	aggr.Register(evt)

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.ExecuteEntityEventTopic, flushedMessages.Add, aggr.AggregateMiddleware)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()
	defer evt.Close()

	inf := entities.NewEntityInfoWrapper().
		WithRepository(&minderv1.Repository{}).
		WithRepositoryID(repoID).
		WithProjectID(projectID).
		WithProvider(providerName)

	<-evt.Running()

	t.Log("Publishing events")
	var wg sync.WaitGroup
	for i := 0; i < int(concurrentEvents); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := inf.BuildMessage()
			require.NoError(t, err, "expected no error when building message")
			err = evt.Publish(rateLimitedMessageTopic, msg.Copy())
			require.NoError(t, err, "expected no error when publishing message")
		}()
	}

	wg.Wait()
	rateLimitedMessages.Wait()

	assert.Equal(t, int32(1), rateLimitedMessages.count.Load(), "expected only one message to be published")

	t.Log("Waiting for lock to be available")
	time.Sleep(time.Duration(eventThreshold) * time.Second)

	t.Log("Publishing flush events")
	var flushWg sync.WaitGroup
	for i := 0; i < int(concurrentEvents); i++ {
		flushWg.Add(1)
		go func() {
			defer flushWg.Done()
			msg, err := inf.BuildMessage()
			require.NoError(t, err, "expected no error when building message")

			err = evt.Publish(events.FlushEntityEventTopic, msg.Copy())
			require.NoError(t, err, "expected no error when publishing message")
		}()
	}

	flushWg.Wait()
	flushedMessages.Wait()

	// flushing should only happen once
	assert.Equal(t, int32(1), flushedMessages.count.Load(), "expected only one message to be published")
}

func createNeededEntities(ctx context.Context, t *testing.T) (projID uuid.UUID, repoID uuid.UUID) {
	t.Helper()

	// setup project
	proj, err := testQueries.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project",
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "expected no error when creating project")

	// setup provider
	_, err = testQueries.CreateProvider(ctx, db.CreateProviderParams{
		Name:       providerName,
		ProjectID:  proj.ID,
		Implements: []db.ProviderType{db.ProviderTypeRest},
		Definition: json.RawMessage(`{}`),
	})
	require.NoError(t, err, "expected no error when creating provider")

	// setup repo
	repo, err := testQueries.CreateRepository(ctx, db.CreateRepositoryParams{
		ProjectID: proj.ID,
		Provider:  providerName,
		RepoName:  "test-repo",
		RepoOwner: "test-owner",
		RepoID:    123,
	})
	require.NoError(t, err, "expected no error when creating repo")

	return proj.ID, repo.ID
}

func TestFlushAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mockDBSetup func(context.Context, *mockdb.MockStore)
	}{
		{
			name: "flushes one repo",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				repoID := uuid.New()
				projectID := uuid.New()

				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:           uuid.New(),
							Entity:       db.EntitiesRepository,
							RepositoryID: repoID,
							QueuedAt:     time.Now(),
						},
					}, nil)

				// 1 - fetch repo info for repo
				// base repo info
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)
				// subsequent repo fetch for protobuf conversion
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
		},
		{
			name: "flushes one artifact",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				repoID := uuid.New()
				artID := uuid.New()
				projectID := uuid.New()

				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:           uuid.New(),
							Entity:       db.EntitiesArtifact,
							RepositoryID: repoID,
							ArtifactID: uuid.NullUUID{
								UUID:  artID,
								Valid: true,
							},
							QueuedAt: time.Now(),
						},
					}, nil)

				// 1 - fetch repo info for repo
				// base repo info
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)
				// subsequent artifact fetch for protobuf conversion
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)
				mockStore.EXPECT().GetArtifactByID(ctx, artID).
					Return(db.GetArtifactByIDRow{
						ID:        artID,
						ProjectID: projectID,
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
		},
		{
			name: "flushes one PR",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				repoID := uuid.New()
				prID := uuid.New()
				projectID := uuid.New()

				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:           uuid.New(),
							Entity:       db.EntitiesPullRequest,
							RepositoryID: repoID,
							PullRequestID: uuid.NullUUID{
								UUID:  prID,
								Valid: true,
							},
							QueuedAt: time.Now(),
						},
					}, nil)

				// 1 - fetch repo info for repo
				// base repo info
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)
				// subsequent artifact fetch for protobuf conversion
				mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
					Return(db.Repository{
						ID:        repoID,
						ProjectID: projectID,
						Provider:  providerName,
					}, nil)
				mockStore.EXPECT().GetPullRequestByID(ctx, prID).
					Return(db.PullRequest{
						ID: prID,
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)

			evt, err := events.Setup(ctx, &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err)

			flushedMessages := newTestPubSub()
			evt.Register(events.ExecuteEntityEventTopic, flushedMessages.Add)

			go func() {
				t.Log("Running eventer")
				err := evt.Run(ctx)
				assert.NoError(t, err, "expected no error when running eventer")
			}()

			<-evt.Running()

			// minimum wait
			var eventThreshold int64 = 1
			aggr := eea.NewEEA(mockStore, evt, &serverconfig.AggregatorConfig{
				LockInterval: eventThreshold,
			})

			tt.mockDBSetup(ctx, mockStore)

			go func() {
				t.Log("Flushing all")
				require.NoError(t, aggr.FlushAll(ctx), "expected no error")
			}()

			t.Log("Waiting for flush")
			flushedMessages.Wait()

			assert.Equal(t, int32(1), flushedMessages.count.Load(), "expected one message")
		})
	}
}

func TestFlushAllListFlushIsEmpty(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	// we'll wait 1 second for the lock to be available
	var eventThreshold int64 = 1
	// use in-memory postgres for this test
	aggr := eea.NewEEA(testQueries, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	})

	flushedMessages := newTestPubSub()

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.ExecuteEntityEventTopic, flushedMessages.Add, aggr.AggregateMiddleware)

	t.Log("Flushing all")
	require.NoError(t, aggr.FlushAll(ctx), "expected no error")

	assert.Equal(t, int32(0), flushedMessages.count.Load(), "expected no messages")
}

func TestFlushAllListFlushFails(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	flushedMessages := newTestPubSub()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.ExecuteEntityEventTopic, flushedMessages.Add)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()

	// we'll wait 1 second for the lock to be available
	var eventThreshold int64 = 1
	aggr := eea.NewEEA(mockStore, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	})

	mockStore.EXPECT().ListFlushCache(ctx).
		Return(nil, fmt.Errorf("expected error"))

	t.Log("Flushing all")
	require.ErrorContains(t, aggr.FlushAll(ctx), "expected error")

	assert.Equal(t, int32(0), flushedMessages.count.Load(), "expected no messages")
}

// This scenario represents the case where the `ListFlushCache` has
// returned a repository that has been deleted. This should not cause
// the flush to fail and should just skip the flush for that repository.
func TestFlushAllListFlushListsARepoThatGetsDeletedLater(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	flushedMessages := newTestPubSub()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.ExecuteEntityEventTopic, flushedMessages.Add)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()

	// we'll wait 1 second for the lock to be available
	var eventThreshold int64 = 1
	aggr := eea.NewEEA(mockStore, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	})

	repoID := uuid.New()

	// initial list flush
	mockStore.EXPECT().ListFlushCache(ctx).
		Return([]db.FlushCache{
			{
				ID:           uuid.New(),
				Entity:       db.EntitiesRepository,
				RepositoryID: repoID,
				QueuedAt:     time.Now(),
			},
		}, nil)

	// repo does not exist
	mockStore.EXPECT().GetRepositoryByID(ctx, repoID).
		Return(db.Repository{}, sql.ErrNoRows)

	t.Log("Flushing all")
	require.NoError(t, aggr.FlushAll(ctx), "expected no error")

	assert.Equal(t, int32(0), flushedMessages.count.Load(), "expected no messages")
}

type testPubSub struct {
	// counts the number of messages added
	count            *atomic.Int32
	firstMessageOnce *sync.Once
	// allows us to wait for the first message to be added
	firstMessage chan struct{}
}

func newTestPubSub() *testPubSub {
	var count atomic.Int32
	return &testPubSub{
		count:            &count,
		firstMessage:     make(chan struct{}),
		firstMessageOnce: &sync.Once{},
	}
}

func (t *testPubSub) Wait() {
	<-t.firstMessage
}

func (t *testPubSub) Add(_ *message.Message) error {
	t.count.Add(1)
	t.firstMessageOnce.Do(func() {
		t.firstMessage <- struct{}{}
	})
	return nil
}
