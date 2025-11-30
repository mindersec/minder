// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reconcilers

import (
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	df "github.com/mindersec/minder/database/mock/fixtures"
	db "github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	rf "github.com/mindersec/minder/internal/repositories/mock/fixtures"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

var (
	repoFullName = "mindersec/minder"
)

func repoProperties() *properties.Properties {
	props := properties.NewProperties(map[string]any{
		properties.PropertyName: repoFullName,
	})
	return props
}

func TestHandleEntityAdd(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:    providerID,
						Name:  "providerName",
						Class: db.ProviderClassGithub,
					},
					providerID,
				),
			),
			mockReposFunc: rf.NewRepoService(
				rf.WithSuccessfulCreate(
					projectID,
					&pb.Repository{},
				),
			),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				err := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithProperties(repoProperties()).
					ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
		},
		{
			name: "db failure",
			mockStoreFunc: df.NewMockStore(
				df.WithFailedGetProviderByID(
					errors.New("oops"),
				),
			),
			mockReposFunc: rf.NewRepoService(),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				err := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithProperties(repoProperties()).
					ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
			err: true,
		},
		{
			name: "repo service failure",
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:    providerID,
						Name:  "providerName",
						Class: db.ProviderClassGithubApp,
					},
					providerID,
				),
			),
			mockReposFunc: rf.NewRepoService(
				rf.WithFailedCreate(
					errors.New("oops"),
					projectID,
				),
			),
			messageFunc: func(t *testing.T) *message.Message {
				t.Helper()
				m := message.NewMessage(uuid.New().String(), nil)
				err := messages.NewMinderEvent().
					WithProviderID(providerID).
					WithProjectID(projectID).
					WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
					WithProperties(repoProperties()).
					ToMessage(m)
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

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// setup
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			reconciler := setUp(t, tt, ctrl)
			m := tt.messageFunc(t)

			// when
			err := reconciler.handleEntityAddEvent(m)

			// then
			if tt.err {
				require.Error(t, err)
			}
		})
	}
}
