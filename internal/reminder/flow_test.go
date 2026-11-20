// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	df "github.com/mindersec/minder/database/mock/fixtures"
	"github.com/mindersec/minder/internal/db"
	engentities "github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/handlers"
	"github.com/mindersec/minder/internal/entities/models"
	mockpropsvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/providers/manager"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/reconcilers"
	"github.com/mindersec/minder/internal/reminderprocessor"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	reminderconfig "github.com/mindersec/minder/pkg/config/reminder"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// syncBus delivers each published message synchronously to the handlers
// registered for its topic, and records messages on topics nobody handles.
type syncBus struct {
	interfaces.Interface
	t        *testing.T
	handlers map[string][]interfaces.Handler
	sent     map[string][]*message.Message
}

func newSyncBus(t *testing.T) *syncBus {
	t.Helper()
	return &syncBus{t: t, handlers: map[string][]interfaces.Handler{}, sent: map[string][]*message.Message{}}
}

func (b *syncBus) Register(topic string, handler interfaces.Handler, _ ...message.HandlerMiddleware) {
	b.handlers[topic] = append(b.handlers[topic], handler)
}

func (b *syncBus) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		b.sent[topic] = append(b.sent[topic], msg)
		for _, handler := range b.handlers[topic] {
			require.NoError(b.t, handler(msg), "handler for %s failed", topic)
		}
	}
	return nil
}

func (*syncBus) Close() error {
	return nil
}

func TestArtifactReminderFlowsToRefreshAndEvaluate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	propSvc := mockpropsvc.NewMockPropertiesService(ctrl)
	provMgr := mockmanager.NewMockProviderManager(ctrl)

	artifact := db.EntityInstance{
		ID:         uuid.New(),
		EntityType: db.EntitiesArtifact,
		Name:       "library/alpine:3.20",
		ProviderID: uuid.New(),
		ProjectID:  uuid.New(),
	}

	store.EXPECT().ListEntitiesAfterID(gomock.Any(), db.ListEntitiesAfterIDParams{
		EntityType: db.EntitiesArtifact, ID: uuid.Nil, Limit: 5,
	}).Return([]db.EntityInstance{artifact}, nil)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), []uuid.UUID{artifact.ID}).
		Return([]db.ListOldestRuleEvaluationsByEntityIDRow{
			{EntityInstanceID: artifact.ID, OldestLastUpdated: time.Now().Add(-12 * time.Hour)},
		}, nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil)
	df.WithTransaction()(store)

	staleProps, err := oci.NewArtifactProperties(artifact.Name, "docker.io", "index.docker.io", "sha256:old")
	require.NoError(t, err)
	freshProps, err := oci.NewArtifactProperties(artifact.Name, "docker.io", "index.docker.io", "sha256:new")
	require.NoError(t, err)
	ewp := models.NewEntityWithProperties(artifact, staleProps)

	propSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), artifact.ID, gomock.Any()).Return(ewp, nil)
	propSvc.EXPECT().RetrieveAllPropertiesForEntity(gomock.Any(), ewp, manager.ProviderManager(provMgr), gomock.Any()).
		DoAndReturn(func(_ context.Context, e *models.EntityWithProperties, _ manager.ProviderManager, _ any) error {
			e.UpdateProperties(freshProps)
			return nil
		})
	propSvc.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), ewp, gomock.Any()).
		DoAndReturn(func(_ context.Context, e *models.EntityWithProperties, _ manager.ProviderManager) (any, error) {
			require.Equal(t, "sha256:new",
				e.Properties.GetProperty(oci.ArtifactPropertyDigest).GetString(), "evaluated with refreshed digest")
			return oci.ArtifactV1FromProperties(e.Properties, "docker.io")
		})

	bus := newSyncBus(t)
	reminderprocessor.NewReminderProcessor(bus).Register(bus)
	reconciler, err := reconcilers.NewReconciler(store, bus, nil, nil, nil, nil)
	require.NoError(t, err)
	reconciler.Register(bus)
	handlers.NewRefreshByIDAndEvaluateHandler(bus, store, propSvc, provMgr).Register(bus)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: time.Hour, BatchSize: 100, MinElapsed: time.Hour,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true), BatchSize: ptr(5), Interval: ptr(6 * time.Hour)},
			},
		},
	})
	r.eventPublisher = bus

	require.NoError(t, entityReminderFor(t, r, db.EntitiesArtifact).sendReminders(context.Background()))

	require.Len(t, bus.sent[constants.TopicQueueRepoReminder], 1)
	require.Len(t, bus.sent[constants.TopicQueueReconcileRepoInit], 1)
	require.Len(t, bus.sent[constants.TopicQueueRefreshEntityByIDAndEvaluate], 1)
	require.Len(t, bus.sent[constants.TopicQueueEntityEvaluate], 1)

	eiw, err := engentities.ParseEntityEvent(bus.sent[constants.TopicQueueEntityEvaluate][0])
	require.NoError(t, err)
	require.Equal(t, minderv1.Entity_ENTITY_ARTIFACTS, eiw.Type)
	require.Equal(t, artifact.ID, eiw.EntityID)
	require.Equal(t, artifact.ProjectID, eiw.ProjectID)
	require.Equal(t, artifact.ProviderID, eiw.ProviderID)
}
