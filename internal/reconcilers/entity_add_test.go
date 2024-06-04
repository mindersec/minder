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
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	df "github.com/stacklok/minder/database/mock/fixtures"
	db "github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	rf "github.com/stacklok/minder/internal/repositories/github/mock/fixtures"
	// "github.com/stacklok/minder/internal/util/testqueue"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	repoOwner = "stacklok"
	repoName  = "minder"
)

func TestHandleEntityAdd(t *testing.T) {
	t.Parallel()

	tests := []testCase{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:   providerID,
						Name: "providerName",
					},
					providerID,
				),
			),
			mockReposFunc: rf.NewRepoService(
				rf.WithSuccessfulCreate(
					projectID,
					repoOwner,
					repoName,
					&pb.Repository{},
				),
			),
			//nolint:thelper
			messageFunc: func(t *testing.T) *message.Message {
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := entities.NewEntityInfoWrapper().
					WithActionEvent("added").
					WithProjectID(projectID).
					WithProviderID(providerID).
					WithRepositoryName(repoName).
					WithRepositoryOwner(repoOwner).
					WithRepository(&pb.Repository{})
				err := eiw.ToMessage(m)
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
			//nolint:thelper
			messageFunc: func(t *testing.T) *message.Message {
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := entities.NewEntityInfoWrapper().
					WithActionEvent("added").
					WithProjectID(projectID).
					WithProviderID(providerID).
					WithRepositoryName(repoName).
					WithRepositoryOwner(repoOwner).
					WithRepository(&pb.Repository{})
				err := eiw.ToMessage(m)
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
						ID:   providerID,
						Name: "providerName",
					},
					providerID,
				),
			),
			mockReposFunc: rf.NewRepoService(
				rf.WithFailedCreate(
					errors.New("oops"),
					projectID,
					repoOwner,
					repoName,
				),
			),
			//nolint:thelper
			messageFunc: func(t *testing.T) *message.Message {
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := entities.NewEntityInfoWrapper().
					WithActionEvent("added").
					WithProjectID(projectID).
					WithProviderID(providerID).
					WithRepositoryName(repoName).
					WithRepositoryOwner(repoOwner).
					WithRepository(&pb.Repository{})
				err := eiw.ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
			err: true,
		},
		{
			name:          "bad message",
			mockStoreFunc: nil,
			//nolint:thelper
			messageFunc: func(_ *testing.T) *message.Message {
				return message.NewMessage(uuid.New().String(), nil)
			},
			err: true,
		},
		{
			name:          "not a repository",
			mockStoreFunc: nil,
			//nolint:thelper
			messageFunc: func(t *testing.T) *message.Message {
				m := message.NewMessage(uuid.New().String(), nil)
				eiw := entities.NewEntityInfoWrapper().
					WithActionEvent("added").
					WithProjectID(projectID).
					WithProviderID(providerID).
					WithArtifact(&pb.Artifact{}).
					WithArtifactID(uuid.New())
				err := eiw.ToMessage(m)
				require.NoError(t, err, "invalid message")
				return m
			},
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
			err := reconciler.handleEntityAddEvent(m)

			// then
			if tt.err {
				require.Error(t, err)
			}
		})
	}
}
