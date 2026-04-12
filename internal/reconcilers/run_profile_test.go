// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"errors"

	watermillMsg "github.com/ThreeDotsLabs/watermill/message"
	df "github.com/mindersec/minder/database/mock/fixtures"
	"github.com/mindersec/minder/internal/db"
	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	stubeventer "github.com/mindersec/minder/internal/events/stubs"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

var (
	testReconcileProjectID = uuid.New()
	simulateFailUUID       = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

func init() {
	toMessage = func(entRefresh *entityMessage.HandleEntityAndDoMessage, m *watermillMsg.Message) error {
		if entRefresh.Entity.EntityID == simulateFailUUID {
			return errors.New("mock marshal error")
		}
		return entRefresh.ToMessage(m)
	}
}

func Test_handleProfileInitEvent(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name         string
		setupDbMocks func() df.MockStoreBuilder
		numPublish   int
		expectedErr  bool
	}{
		{
			name: "valid event",
			setupDbMocks: func() df.MockStoreBuilder {
				retEnts := []db.EntityInstance{
					{
						EntityType: db.EntitiesArtifact,
						ID:         uuid.New(),
					},
					{
						EntityType: db.EntitiesPullRequest,
						ID:         uuid.New(),
					},
					{
						EntityType: db.EntitiesRepository,
						ID:         uuid.New(),
					},
				}
				return df.NewMockStore(
					df.WithSuccessfulGetEntitiesByProjectHierarchy(retEnts, []uuid.UUID{testReconcileProjectID}),
				)
			},
			expectedErr: false,
			numPublish:  3,
		},
		{
			name: "error getting entities",
			setupDbMocks: func() df.MockStoreBuilder {
				return df.NewMockStore(
					df.WithFailedGetEntitiesByProjectHierarchy(sql.ErrNoRows),
				)
			},
			expectedErr: true,
			numPublish:  0,
		},
		{
			name: "marshal failure skips but continues",
			setupDbMocks: func() df.MockStoreBuilder {
				retEnts := []db.EntityInstance{
					{
						EntityType: db.EntitiesArtifact,
						ID:         uuid.New(),
					},
					{
						EntityType: db.EntitiesPullRequest,
						ID:         simulateFailUUID, // this will fail to marshal in test hook
					},
					{
						EntityType: db.EntitiesRepository,
						ID:         uuid.New(),
					},
				}
				return df.NewMockStore(
					df.WithSuccessfulGetEntitiesByProjectHierarchy(retEnts, []uuid.UUID{testReconcileProjectID}),
				)
			},
			expectedErr: false,
			numPublish:  2,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg, err := NewProfileInitMessage(projectID)
			require.NoError(t, err)
			require.NotNil(t, msg)

			stubEventer := &stubeventer.StubEventer{}
			mockStore := scenario.setupDbMocks()(ctrl)

			reconciler, err := NewReconciler(mockStore, stubEventer, nil, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, reconciler)

			err = reconciler.publishProfileInitEvents(context.Background(), testReconcileProjectID)
			if scenario.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, scenario.numPublish, len(stubEventer.Sent))
			if scenario.numPublish > 0 {
				require.Contains(t, stubEventer.Topics, constants.TopicQueueRefreshEntityByIDAndEvaluate)
			}
		})
	}
}
