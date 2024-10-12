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
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	df "github.com/stacklok/minder/database/mock/fixtures"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	mockrepo "github.com/stacklok/minder/internal/repositories/mock"
	rf "github.com/stacklok/minder/internal/repositories/mock/fixtures"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	projectID    = uuid.New()
	providerID   = uuid.New()
	repositoryID = uuid.New()
)

type testCase struct {
	name          string
	mockStoreFunc df.MockStoreBuilder
	mockReposFunc rf.RepoMockBuilder
	messageFunc   func(*testing.T) *message.Message
	err           bool
}

func TestHandleEntityDelete(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name:          "happy path",
			mockStoreFunc: nil,
			mockReposFunc: rf.NewRepoService(
				rf.WithSuccessfulDeleteByIDDetailed(
					repositoryID,
					projectID,
				),
			),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithEntityID(repositoryID)

				err := eiw.ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
		},
		{
			name:          "ignore entity not found - expect no error",
			mockStoreFunc: nil,
			mockReposFunc: rf.NewRepoService(
				rf.WithFailedDeleteByID(
					service.ErrEntityNotFound,
				),
			),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithEntityID(repositoryID)

				err := eiw.ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
			err: false,
		},
		{
			name:          "db failure",
			mockStoreFunc: nil,
			mockReposFunc: rf.NewRepoService(
				rf.WithFailedDeleteByID(
					errors.New("oops"),
				),
			),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithEntityID(repositoryID)

				err := eiw.ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
			err: true,
		},
		{
			name:          "bad message",
			mockStoreFunc: nil,
			messageFunc: func(_ *testing.T) *message.Message {
				t.Helper()
				return message.NewMessage(uuid.New().String(), nil)
			},
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			reconciler := setUp(t, tt, ctrl)
			m := tt.messageFunc(t)

			// when
			err := reconciler.handleEntityDeleteEvent(m)

			// then
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func setUp(t *testing.T, tt testCase, ctrl *gomock.Controller) *Reconciler {
	t.Helper()

	mockStore := mockdb.NewMockStore(ctrl)
	if tt.mockStoreFunc != nil {
		mockStore = tt.mockStoreFunc(ctrl)
	}

	repoService := mockrepo.NewMockRepositoryService(ctrl)
	if tt.mockReposFunc != nil {
		repoService = tt.mockReposFunc(ctrl)
	}

	evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	reconciler, err := NewReconciler(
		mockStore,
		evt,
		nil, // crypto.Engine not used in these tests
		nil, // manager.ProviderManager not used in these tests
		repoService,
	)
	require.NoError(t, err)

	return reconciler
}
