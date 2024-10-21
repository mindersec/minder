// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	df "github.com/mindersec/minder/database/mock/fixtures"
	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	mockrepo "github.com/mindersec/minder/internal/repositories/mock"
	rf "github.com/mindersec/minder/internal/repositories/mock/fixtures"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

	evt, err := events.Setup(context.Background(), nil, &serverconfig.EventConfig{
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
