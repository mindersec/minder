// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package eea_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	"github.com/mindersec/minder/internal/eea"
	"github.com/mindersec/minder/internal/engine/entities"
	psvc "github.com/mindersec/minder/internal/entities/properties/service"
	mockgithub "github.com/mindersec/minder/internal/providers/github/mock"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// TestFlushAllWithQueuedRelease covers the case where a release evaluation
// event arrives while the release is locked by a running evaluation, so the
// aggregator parks it in flush_cache. If the server stops before that
// evaluation flushes the row, the next server start calls FlushAll
// (internal/service/service.go), which must not fail because of the row.
func TestFlushAllWithQueuedRelease(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testQueries, td, err := embedded.GetFakeStore()
	require.NoError(t, err, "expected no error when creating embedded store")
	t.Cleanup(td)

	projectID, repoID := createNeededEntities(ctx, t, testQueries)

	repo, err := testQueries.GetEntityByID(ctx, repoID)
	require.NoError(t, err, "expected no error when getting repo")

	// setup a release originating from the repo, with the properties the
	// GitHub provider stores for releases
	releaseProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: "123456",
		properties.PropertyName:       "test-owner/test-repo/v1.0.0",
		properties.ReleasePropertyTag: "v1.0.0",
		ghprop.ReleasePropertyOwner:   "test-owner",
		ghprop.ReleasePropertyRepo:    "test-repo",
	})

	release, err := testQueries.CreateEntity(ctx, db.CreateEntityParams{
		EntityType:     db.EntitiesRelease,
		Name:           "test-owner/test-repo/v1.0.0",
		ProjectID:      projectID,
		ProviderID:     repo.ProviderID,
		OriginatedFrom: uuid.NullUUID{UUID: repoID, Valid: true},
	})
	require.NoError(t, err, "expected no error when creating release")

	err = psvc.NewPropertiesService(testQueries).SaveAllProperties(ctx, release.ID, releaseProps, nil)
	require.NoError(t, err, "expected no error when saving release properties")

	evt, err := eventer.New(ctx, nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	// capture the evaluate events FlushAll re-publishes
	published := make(chan *message.Message, 10)
	evt.Register(constants.TopicQueueEntityEvaluate, func(msg *message.Message) error {
		published <- msg
		return nil
	})

	runErr := make(chan error, 1)
	go func() {
		runErr <- evt.Run(ctx)
	}()
	<-evt.Running()

	// FlushAll rebuilds the queued event through the provider, so use a
	// GitHub provider with the real release-to-proto conversion.
	ctrl := gomock.NewController(t)
	ghProvider := mockgithub.NewMockGitHub(ctrl)
	ghProvider.EXPECT().PropertiesToProtoMessage(minderv1.Entity_ENTITY_RELEASE, gomock.Any()).
		DoAndReturn(func(_ minderv1.Entity, props *properties.Properties) (protoreflect.ProtoMessage, error) {
			return ghprop.EntityInstanceV1FromReleaseProperties(props)
		}).AnyTimes()
	provMan := mockmanager.NewMockProviderManager(ctrl)
	provMan.EXPECT().InstantiateFromID(gomock.Any(), repo.ProviderID).Return(ghProvider, nil).AnyTimes()

	// a long lock interval, so the second event is queued instead of executed
	var eventThreshold int64 = 300
	aggr := eea.NewEEA(testQueries, evt, &serverconfig.AggregatorConfig{
		LockInterval: eventThreshold,
	}, psvc.NewPropertiesService(testQueries), provMan)

	// Build the evaluation message the same way the refresh-and-evaluate
	// handler does (toEntityInfoWrapper), using the GitHub provider's
	// release-to-proto conversion.
	newReleaseEvalMessage := func() *message.Message {
		t.Helper()
		pbRelease, err := ghprop.EntityInstanceV1FromReleaseProperties(releaseProps)
		require.NoError(t, err, "expected no error when converting release properties")

		msg := message.NewMessage(uuid.New().String(), nil)
		err = entities.NewEntityInfoWrapper().
			WithProjectID(projectID).
			WithProviderID(repo.ProviderID).
			WithProtoMessage(minderv1.Entity_ENTITY_RELEASE, pbRelease).
			WithID(release.ID).
			ToMessage(msg)
		require.NoError(t, err, "expected no error when building message")
		msg.SetContext(ctx)
		return msg
	}

	// counts the events the aggregator passes on for execution
	executed := 0
	handler := aggr.AggregateMiddleware(func(_ *message.Message) ([]*message.Message, error) {
		executed++
		return nil, nil
	})

	// The first event takes the lock and is passed on for execution.
	_, err = handler(newReleaseEvalMessage())
	require.NoError(t, err, "expected no error when aggregating first event")

	// The second event finds the lock held and is queued in flush_cache.
	_, err = handler(newReleaseEvalMessage())
	require.NoError(t, err, "expected no error when aggregating second event")

	require.Equal(t, 1, executed, "expected only the first event to be executed")

	cache, err := testQueries.ListFlushCache(ctx)
	require.NoError(t, err, "expected no error when listing flush cache")
	require.Len(t, cache, 1, "expected the second event to be queued in flush_cache")
	require.Equal(t, db.EntitiesRelease, cache[0].Entity)
	require.Equal(t, release.ID, cache[0].EntityInstanceID)

	// Simulate a server start: internal/service/service.go returns an error
	// (and the server exits) if FlushAll fails.
	require.NoError(t, aggr.FlushAll(ctx), "expected FlushAll to handle a queued release")

	// The queued release must be re-published for evaluation, not dropped.
	select {
	case msg := <-published:
		inf, err := entities.ParseEntityEvent(msg)
		require.NoError(t, err, "expected no error when parsing flushed event")
		require.Equal(t, minderv1.Entity_ENTITY_RELEASE, inf.Type)
		flushedID, err := inf.GetID()
		require.NoError(t, err, "expected flushed event to have an entity ID")
		require.Equal(t, release.ID, flushedID)
		require.Equal(t, projectID, inf.ProjectID)
	case <-time.After(10 * time.Second):
		t.Fatal("expected the queued release to be flushed as an evaluate event")
	}
	require.Empty(t, published, "expected exactly one flushed event")

	cache, err = testQueries.ListFlushCache(ctx)
	require.NoError(t, err, "expected no error when listing flush cache")
	require.Empty(t, cache, "expected flush_cache to be empty after FlushAll")

	cancel()
	require.NoError(t, <-runErr)
}
