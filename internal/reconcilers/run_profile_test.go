// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcilers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	df "github.com/stacklok/minder/database/mock/fixtures"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
)

var (
	testReconcileProjectID = uuid.New()
)

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

			reconciler, err := NewReconciler(mockStore, stubEventer, nil, nil, nil, nil)
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
				require.Contains(t, stubEventer.Topics, events.TopicQueueRefreshEntityByIDAndEvaluate)
			}
		})
	}
}
