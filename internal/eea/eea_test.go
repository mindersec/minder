// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package eea_test

import (
	"context"
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

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	"github.com/mindersec/minder/internal/eea"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/entities/properties"
	psvc "github.com/mindersec/minder/internal/entities/properties/service"
	propsvcmock "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/events"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

const (
	providerName = "test-provider"
)

var (
	providerID = uuid.New()
)

func TestAggregator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testQueries, td, err := embedded.GetFakeStore()
	require.NoError(t, err, "expected no error when creating embedded store")
	t.Cleanup(td)

	var concurrentEvents int64 = 100

	projectID, repoID := createNeededEntities(ctx, t, testQueries)

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
	},
		// we don't need the properties service for this test. These are only needed
		// for flushing all
		nil, nil)

	rateLimitedMessages := newTestPubSub()
	flushedMessages := newTestPubSub()

	rateLimitedMessageTopic := t.Name()

	// This tests that the middleware works as expected
	evt.Register(rateLimitedMessageTopic, rateLimitedMessages.Add, aggr.AggregateMiddleware)

	// This tests that flushing works as expected
	aggr.Register(evt)

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.TopicQueueEntityEvaluate, flushedMessages.Add, aggr.AggregateMiddleware)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()
	defer evt.Close()

	inf := entities.NewEntityInfoWrapper().
		WithRepository(&minderv1.Repository{}).
		WithID(repoID).
		WithProjectID(projectID).
		WithProviderID(providerID)
	msg, err := inf.BuildMessage()
	require.NoError(t, err, "expected no error when building message")

	<-evt.Running()

	t.Log("Publishing events")
	var wg sync.WaitGroup
	for i := 0; i < int(concurrentEvents); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := evt.Publish(rateLimitedMessageTopic, msg.Copy())
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

			err = evt.Publish(events.TopicQueueEntityFlush, msg.Copy())
			require.NoError(t, err, "expected no error when publishing message")
		}()
	}

	flushWg.Wait()
	flushedMessages.Wait()

	// flushing should only happen once
	assert.Equal(t, int32(1), flushedMessages.count.Load(), "expected only one message to be published")
}

func createNeededEntities(ctx context.Context, t *testing.T, testQueries db.Store) (projID uuid.UUID, repoID uuid.UUID) {
	t.Helper()

	// setup project
	proj, err := testQueries.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project",
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "expected no error when creating project")

	// setup provider
	prov, err := testQueries.CreateProvider(ctx, db.CreateProviderParams{
		Name:       providerName,
		ProjectID:  proj.ID,
		Class:      db.ProviderClassGithub,
		Implements: []db.ProviderType{db.ProviderTypeRest},
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
		Definition: json.RawMessage(`{}`),
	})
	require.NoError(t, err, "expected no error when creating provider")

	// setup repo
	repo, err := testQueries.CreateEntity(ctx, db.CreateEntityParams{
		EntityType: db.EntitiesRepository,
		Name:       "test-repo",
		ProjectID:  proj.ID,
		ProviderID: prov.ID,
	})
	require.NoError(t, err, "expected no error when creating repo")

	return proj.ID, repo.ID
}

func TestFlushAll(t *testing.T) {
	t.Parallel()

	repoID := uuid.New()
	artID := uuid.New()
	prID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()

	tests := []struct {
		name             string
		mockDBSetup      func(context.Context, *mockdb.MockStore)
		mockPropSvcSetup func(*propsvcmock.MockPropertiesService)
	}{
		{
			name: "flushes one repo",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:               uuid.New(),
							Entity:           db.EntitiesRepository,
							QueuedAt:         time.Now(),
							ProjectID:        projectID,
							EntityInstanceID: repoID,
						},
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
			mockPropSvcSetup: func(mockPropSvc *propsvcmock.MockPropertiesService) {
				mockPropSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), gomock.Eq(repoID), gomock.Nil()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         repoID,
							Type:       minderv1.Entity_ENTITY_REPOSITORIES,
							ProjectID:  projectID,
							ProviderID: providerID,
						},
						Properties: &properties.Properties{},
					}, nil)

				mockPropSvc.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&minderv1.Repository{}, nil)
			},
		},
		{
			name: "flushes one artifact with repo",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:               uuid.New(),
							Entity:           db.EntitiesArtifact,
							QueuedAt:         time.Now(),
							ProjectID:        projectID,
							EntityInstanceID: artID,
						},
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
			mockPropSvcSetup: func(mockPropSvc *propsvcmock.MockPropertiesService) {
				mockPropSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), gomock.Eq(artID), gomock.Nil()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:             artID,
							Type:           minderv1.Entity_ENTITY_ARTIFACTS,
							ProjectID:      projectID,
							ProviderID:     providerID,
							OriginatedFrom: repoID,
						},
						Properties: &properties.Properties{},
					}, nil)

				mockPropSvc.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&minderv1.Artifact{}, nil)
			},
		},
		{
			name: "flushes one artifact with no repo",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:               uuid.New(),
							Entity:           db.EntitiesArtifact,
							QueuedAt:         time.Now(),
							ProjectID:        projectID,
							EntityInstanceID: artID,
						},
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
			mockPropSvcSetup: func(mockPropSvc *propsvcmock.MockPropertiesService) {
				mockPropSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), gomock.Eq(artID), gomock.Nil()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         artID,
							Type:       minderv1.Entity_ENTITY_ARTIFACTS,
							ProjectID:  projectID,
							ProviderID: providerID,
						},
						Properties: &properties.Properties{},
					}, nil)

				mockPropSvc.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&minderv1.Artifact{}, nil)
			},
		},
		{
			name: "flushes one PR",
			mockDBSetup: func(ctx context.Context, mockStore *mockdb.MockStore) {
				mockStore.EXPECT().ListFlushCache(ctx).
					Return([]db.FlushCache{
						{
							ID:               uuid.New(),
							Entity:           db.EntitiesPullRequest,
							ProjectID:        projectID,
							EntityInstanceID: prID,
							QueuedAt:         time.Now(),
						},
					}, nil)

				// There should be one flush in the end
				mockStore.EXPECT().FlushCache(ctx, gomock.Any()).Times(1)
			},
			mockPropSvcSetup: func(mockPropSvc *propsvcmock.MockPropertiesService) {
				mockPropSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), gomock.Eq(prID), gomock.Nil()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:             prID,
							Type:           minderv1.Entity_ENTITY_PULL_REQUESTS,
							ProjectID:      projectID,
							ProviderID:     providerID,
							OriginatedFrom: repoID,
						},
						Properties: &properties.Properties{},
					}, nil)

				mockPropSvc.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&minderv1.PullRequest{}, nil)
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

			propsvc := propsvcmock.NewMockPropertiesService(ctrl)
			provman := mockmanager.NewMockProviderManager(ctrl)

			evt, err := events.Setup(ctx, &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err)

			flushedMessages := newTestPubSub()
			evt.Register(events.TopicQueueEntityEvaluate, flushedMessages.Add)

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
			}, propsvc, provman)

			tt.mockDBSetup(ctx, mockStore)
			tt.mockPropSvcSetup(propsvc)

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

	testQueries, td, err := embedded.GetFakeStore()
	require.NoError(t, err, "expected no error when creating embedded store")
	t.Cleanup(td)

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
	},
		// These are not used since we're not flushing any entities
		nil, nil)

	flushedMessages := newTestPubSub()

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.TopicQueueEntityEvaluate, flushedMessages.Add, aggr.AggregateMiddleware)

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
	evt.Register(events.TopicQueueEntityEvaluate, flushedMessages.Add)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()

	// we'll wait 1 second for the lock to be available
	var eventThreshold int64 = 1
	aggr := eea.NewEEA(mockStore, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	},
		// These are not used since we're not flushing any entities
		nil, nil)

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
	propsvc := propsvcmock.NewMockPropertiesService(ctrl)
	provman := mockmanager.NewMockProviderManager(ctrl)

	flushedMessages := newTestPubSub()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	// This tests that flushing sends messages to the executor engine
	evt.Register(events.TopicQueueEntityEvaluate, flushedMessages.Add)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()

	// we'll wait 1 second for the lock to be available
	var eventThreshold int64 = 1
	aggr := eea.NewEEA(mockStore, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	}, propsvc, provman)

	repoID := uuid.New()
	projID := uuid.New()

	// initial list flush
	mockStore.EXPECT().ListFlushCache(ctx).
		Return([]db.FlushCache{
			{
				ID:               uuid.New(),
				Entity:           db.EntitiesRepository,
				ProjectID:        projID,
				EntityInstanceID: repoID,
				QueuedAt:         time.Now(),
			},
		}, nil)

	propsvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), gomock.Eq(repoID), gomock.Nil()).
		Return(nil, psvc.ErrEntityNotFound)

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
