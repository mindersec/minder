// Copyright 2023 Stacklok, Inc
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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	propSvc "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/entities/properties/service/mock/fixtures"
	"github.com/stacklok/minder/internal/events"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
	"github.com/stacklok/minder/internal/reconcilers/messages"
)

var (
	testProviderID = uuid.New()
	testProjectID  = uuid.New()
	testRepoID     = uuid.New()
)

func Test_handleRepoReconcilerEvent(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name              string
		setupPropSvcMocks func() fixtures.MockPropertyServiceBuilder
		expectedPublish   bool
		expectedErr       bool
		topic             string
	}{
		{
			name: "valid event",
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				// this just shortcuts the function at the point we will refactor
				// soon
				return fixtures.NewMockPropertiesService(
					fixtures.WithFailedGetEntityWithPropertiesByID(propSvc.ErrEntityNotFound),
				)
			},
			topic:           events.TopicQueueRefreshEntityByIDAndEvaluate,
			expectedPublish: true,
			expectedErr:     false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg, err := messages.NewRepoReconcilerMessage(testProviderID, testRepoID, testProjectID)
			require.NoError(t, err)
			require.NotNil(t, msg)

			stubEventer := &stubeventer.StubEventer{}
			mockPropSvc := scenario.setupPropSvcMocks()(ctrl)

			reconciler, err := NewReconciler(nil, stubEventer, nil, nil, nil, mockPropSvc)
			require.NoError(t, err)
			require.NotNil(t, reconciler)

			err = reconciler.handleRepoReconcilerEvent(msg)
			if scenario.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if scenario.expectedPublish {
				require.Equal(t, 1, len(stubEventer.Sent))
				require.Contains(t, stubEventer.Topics, scenario.topic)
			} else {
				require.Equal(t, 0, len(stubEventer.Sent))
			}
		})
	}
}
