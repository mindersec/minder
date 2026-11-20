// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminderprocessor

import (
	"encoding/json"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	reconcilermessages "github.com/mindersec/minder/internal/reconcilers/messages"
	remindermessages "github.com/mindersec/minder/internal/reminder/messages"
	"github.com/mindersec/minder/pkg/eventer/constants"
	eventermock "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
)

func TestReminderProcessor_Register(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvt := eventermock.NewMockInterface(ctrl)
	rp := NewReminderProcessor(mockEvt)

	// Since interfaces.Registrar is implemented by interfaces.Interface
	// we can just use our mockEvt as the Registrar.
	mockEvt.EXPECT().Register(constants.TopicQueueRepoReminder, gomock.Any(), gomock.Any()).Return()

	rp.Register(mockEvt)
}

func TestReminderProcessor_reminderMessageHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvt := eventermock.NewMockInterface(ctrl)
	rp := NewReminderProcessor(mockEvt)

	projectID := uuid.New()
	providerID := uuid.New()
	entityID := uuid.New()

	msg, err := remindermessages.NewEntityReminderMessage(providerID, entityID, projectID)
	require.NoError(t, err)

	// Expect the publish of the reconciler message to happen
	mockEvt.EXPECT().Publish(constants.TopicQueueReconcileRepoInit, gomock.Any()).Return(nil)

	err = rp.reminderMessageHandler(msg)
	require.NoError(t, err)
}

func TestReminderProcessor_reminderMessageHandler_InvalidMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEvt := eventermock.NewMockInterface(ctrl)
	rp := NewReminderProcessor(mockEvt)

	// Create an invalid message
	msg := message.NewMessage(uuid.New().String(), []byte("invalid json"))

	err := rp.reminderMessageHandler(msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error unmarshalling reminder event")
}

func TestReminderProcessor_reminderMessageHandler_ArtifactEntity(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockEvt := eventermock.NewMockInterface(ctrl)
	rp := NewReminderProcessor(mockEvt)

	projectID := uuid.New()
	providerID := uuid.New()
	artifactID := uuid.New()

	msg, err := remindermessages.NewEntityReminderMessage(providerID, artifactID, projectID)
	require.NoError(t, err)

	var published *message.Message
	mockEvt.EXPECT().Publish(constants.TopicQueueReconcileRepoInit, gomock.Any()).
		DoAndReturn(func(_ string, msgs ...*message.Message) error {
			require.Len(t, msgs, 1)
			published = msgs[0]
			return nil
		})

	require.NoError(t, rp.reminderMessageHandler(msg))
	require.NotNil(t, published)

	var evt reconcilermessages.RepoReconcilerEvent
	require.NoError(t, json.Unmarshal(published.Payload, &evt))
	require.Equal(t, reconcilermessages.RepoReconcilerEvent{
		Project: projectID, Provider: providerID, EntityID: artifactID,
	}, evt)
}
