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

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	mockidentity "github.com/stacklok/minder/internal/auth/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/projects"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetInviteDetails(t *testing.T) {
	t.Parallel()

	projectDisplayName := "project1"
	identityName := "User"
	projectMetadata, err := json.Marshal(
		projects.Metadata{Public: projects.PublicMetadataV1{DisplayName: projectDisplayName}},
	)
	require.NoError(t, err)

	scenarios := []struct {
		name           string
		code           string
		setup          func(store *mockdb.MockStore, idClient *mockidentity.MockResolver)
		expectedError  string
		expectedResult *pb.GetInviteDetailsResponse
	}{
		{
			name:          "missing code",
			code:          "",
			expectedError: "code is required",
		},
		{
			name: "invitation not found",
			code: "code",
			setup: func(store *mockdb.MockStore, _ *mockidentity.MockResolver) {
				store.EXPECT().GetInvitationByCode(gomock.Any(), "code").Return(db.GetInvitationByCodeRow{}, sql.ErrNoRows)
			},
			expectedError: "invitation not found",
		},
		{
			name: "success",
			code: "code",
			setup: func(store *mockdb.MockStore, idClient *mockidentity.MockResolver) {
				projectID := uuid.New()
				identitySubject := "user1"
				store.EXPECT().GetInvitationByCode(gomock.Any(), "code").Return(db.GetInvitationByCodeRow{
					Project:         projectID,
					IdentitySubject: identitySubject,
				}, nil)
				store.EXPECT().GetProjectByID(gomock.Any(), projectID).Return(db.Project{
					Name:     "project1",
					Metadata: projectMetadata,
				}, nil)
				idClient.EXPECT().Resolve(gomock.Any(), identitySubject).Return(&auth.Identity{
					UserID:    identitySubject,
					HumanName: identityName,
				}, nil)
			},
			expectedResult: &pb.GetInviteDetailsResponse{
				ProjectDisplay: projectDisplayName,
				SponsorDisplay: identityName,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockIdClient := mockidentity.NewMockResolver(ctrl)
			if scenario.setup != nil {
				scenario.setup(mockStore, mockIdClient)
			}

			server := &Server{
				store:    mockStore,
				idClient: mockIdClient,
			}

			invite, err := server.GetInviteDetails(context.Background(), &pb.GetInviteDetailsRequest{
				Code: scenario.code,
			})

			if scenario.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, scenario.expectedResult.GetProjectDisplay(), invite.GetProjectDisplay())
			require.Equal(t, scenario.expectedResult.GetSponsorDisplay(), invite.GetSponsorDisplay())
		})
	}
}
