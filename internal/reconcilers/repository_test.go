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

	"github.com/mindersec/minder/internal/events"
	stubeventer "github.com/mindersec/minder/internal/events/stubs"
	"github.com/mindersec/minder/internal/reconcilers/messages"
)

var (
	testProviderID = uuid.New()
	testProjectID  = uuid.New()
	testRepoID     = uuid.New()
)

func Test_handleRepoReconcilerEvent(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name            string
		expectedPublish bool
		expectedErr     bool
		entityID        uuid.UUID
		topic           string
	}{
		{
			name:            "valid event",
			topic:           events.TopicQueueRefreshEntityByIDAndEvaluate,
			entityID:        testRepoID,
			expectedPublish: true,
			expectedErr:     false,
		},
		{
			// this is the case for gitlab. We test here that the event is published for the repo, but no errors occur
			// in this case the current code will issue the reconcile for the repo, but stop without a fatal error
			// just before reconciling artifacts - we verify that because if we hit the artifacts path, we would have
			// a bunch of other mocks to call
			name:            "event with string as upstream ID does publish",
			topic:           events.TopicQueueRefreshEntityByIDAndEvaluate,
			entityID:        testRepoID,
			expectedPublish: true,
			expectedErr:     false,
		},
		{
			name:            "event with no upstream ID",
			entityID:        uuid.Nil,
			expectedPublish: false,
			expectedErr:     false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			msg, err := messages.NewRepoReconcilerMessage(testProviderID, scenario.entityID, testProjectID)
			require.NoError(t, err)
			require.NotNil(t, msg)

			stubEventer := &stubeventer.StubEventer{}

			reconciler, err := NewReconciler(nil, stubEventer, nil, nil, nil)
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
