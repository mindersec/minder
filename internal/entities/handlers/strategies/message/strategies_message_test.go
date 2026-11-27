// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package message_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/engine/entities"
	strategiesMsg "github.com/mindersec/minder/internal/entities/handlers/strategies/message"
	"github.com/mindersec/minder/internal/entities/models"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mockProvMgr "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestToMinderEntityStrategy(t *testing.T) {
	t.Parallel()

	strat := strategiesMsg.NewToMinderEntity()
	require.NotNil(t, strat)

	t.Run("nil entity returns error", func(t *testing.T) {
		t.Parallel()

		msg, err := strat.CreateMessage(context.Background(), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entity with properties is nil")
		require.Nil(t, msg)
	})

	testCases := []struct {
		name       string
		entityType minderv1.Entity
		entityName string
	}{
		{
			name:       "repository entity",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entityName: "test-owner/test-repo",
		},
		{
			name:       "pull request entity",
			entityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
			entityName: "test-owner/test-repo/pull/1",
		},
		{
			name:       "artifact entity",
			entityType: minderv1.Entity_ENTITY_ARTIFACTS,
			entityName: "test-owner/test-repo/container",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			projID := uuid.New()
			provID := uuid.New()
			entID := uuid.New()

			ewp := &models.EntityWithProperties{
				Entity: models.EntityInstance{
					ID:         entID,
					Type:       tc.entityType,
					Name:       tc.entityName,
					ProjectID:  projID,
					ProviderID: provID,
				},
			}

			msg, err := strat.CreateMessage(context.Background(), ewp)
			require.NoError(t, err)
			require.NotNil(t, msg)
			require.NotEmpty(t, msg.UUID)

			require.Equal(t, provID.String(), msg.Metadata.Get("providerID"))
			require.Equal(t, projID.String(), msg.Metadata.Get("projectID"))

			var parsedEv messages.MinderEvent
			err = json.Unmarshal(msg.Payload, &parsedEv)
			require.NoError(t, err)
			require.Equal(t, projID, parsedEv.ProjectID)
			require.Equal(t, provID, parsedEv.ProviderID)
			require.Equal(t, entID, parsedEv.EntityID)
			require.Equal(t, tc.entityType, parsedEv.EntityType)
		})
	}
}

func TestToEntityInfoWrapperStrategy(t *testing.T) {
	t.Parallel()

	t.Run("nil entity returns error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		s := strategiesMsg.NewToEntityInfoWrapper(
			mockdb.NewMockStore(ctrl),
			mockPropSvc.NewMockPropertiesService(ctrl),
			mockProvMgr.NewMockProviderManager(ctrl),
		)

		msg, err := s.CreateMessage(context.Background(), nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "entity with properties is nil")
		require.Nil(t, msg)
	})

	t.Run("conversion to proto error propagates cleanly", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockP := mockPropSvc.NewMockPropertiesService(ctrl)
		mockM := mockProvMgr.NewMockProviderManager(ctrl)
		mockS := mockdb.NewMockStore(ctrl)

		s := strategiesMsg.NewToEntityInfoWrapper(mockS, mockP, mockM)

		ewp := &models.EntityWithProperties{
			Entity: models.EntityInstance{
				ID:         uuid.New(),
				Type:       minderv1.Entity_ENTITY_REPOSITORIES,
				Name:       "owner/repo",
				ProjectID:  uuid.New(),
				ProviderID: uuid.New(),
			},
		}

		expectedErr := errors.New("upstream protobuf conversion error")
		mockP.EXPECT().
			EntityWithPropertiesAsProto(gomock.Any(), ewp, mockM).
			Return(nil, expectedErr)

		msg, err := s.CreateMessage(context.Background(), ewp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error converting entity to protobuf")
		require.ErrorIs(t, err, expectedErr)
		require.Nil(t, msg)
	})

	t.Run("successful message creation with protobuf wrapper metadata", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(ctrl.Finish)

		mockP := mockPropSvc.NewMockPropertiesService(ctrl)
		mockM := mockProvMgr.NewMockProviderManager(ctrl)
		mockS := mockdb.NewMockStore(ctrl)

		s := strategiesMsg.NewToEntityInfoWrapper(mockS, mockP, mockM)

		projID := uuid.New()
		provID := uuid.New()
		entID := uuid.New()

		ewp := &models.EntityWithProperties{
			Entity: models.EntityInstance{
				ID:         entID,
				Type:       minderv1.Entity_ENTITY_REPOSITORIES,
				Name:       "owner/repo",
				ProjectID:  projID,
				ProviderID: provID,
			},
		}

		pbRepo := &minderv1.Repository{
			Name:      "repo",
			Owner:     "owner",
			RepoId:    12345,
			IsFork:    false,
			IsPrivate: false,
		}

		mockP.EXPECT().
			EntityWithPropertiesAsProto(gomock.Any(), ewp, mockM).
			Return(pbRepo, nil)

		msg, err := s.CreateMessage(context.Background(), ewp)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.NotEmpty(t, msg.UUID)

		require.Equal(t, projID.String(), msg.Metadata.Get(entities.ProjectIDEventKey))
		require.Equal(t, provID.String(), msg.Metadata.Get(entities.ProviderIDEventKey))
		require.Equal(t, entID.String(), msg.Metadata.Get(entities.EntityIDEventKey))
		require.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES.ToString(), msg.Metadata.Get(entities.EntityTypeEventKey))

		eiw, err := entities.ParseEntityEvent(msg)
		require.NoError(t, err)
		require.Equal(t, projID, eiw.ProjectID)
		require.Equal(t, provID, eiw.ProviderID)
		parsedID, err := eiw.GetID()
		require.NoError(t, err)
		require.Equal(t, entID, parsedID)
		require.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, eiw.Type)
	})
}
