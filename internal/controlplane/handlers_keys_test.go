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

package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/db"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

func TestKeysHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projectID := uuid.New()

	request := &pb.CreateKeyPairRequest{
		Passphrase: "test",
		ProjectId:  projectID.String(),
	}

	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	// Set the expectations for the CreateSigningKey method
	mockStore.EXPECT().CreateSigningKey(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, params db.CreateSigningKeyParams) (db.SigningKey, error) {
			// Validate the generated key pair
			assert.NotEmpty(t, params.PrivateKey, "Private key should not be empty")
			assert.NotEmpty(t, params.PublicKey, "Public key should not be empty")

			// Return a mock SigningKey with the expected properties
			return db.SigningKey{
				ID:            1,
				ProjectID:     params.ProjectID,
				PrivateKey:    params.PrivateKey,
				PublicKey:     params.PublicKey,
				Passphrase:    params.Passphrase,
				KeyIdentifier: "test",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}, nil
		},
	)

	s := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
	}

	response, err := s.CreateKeyPair(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response.PublicKey)
	assert.NotNil(t, response.KeyIdentifier)
	// Add more assertions as needed
}

func TestKeysHandler_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

	testCases := []struct {
		name               string
		request            *pb.CreateKeyPairRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, response *pb.CreateKeyPairResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "success",
			request: &pb.CreateKeyPairRequest{
				Passphrase: "test",
				ProjectId:  projectID.String(),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateSigningKey(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreateSigningKeyParams) (db.SigningKey, error) {
						return db.SigningKey{
							ID:            1,
							ProjectID:     params.ProjectID,
							PrivateKey:    params.PrivateKey,
							PublicKey:     params.PublicKey,
							Passphrase:    params.Passphrase,
							KeyIdentifier: "test",
							CreatedAt:     time.Now(),
							UpdatedAt:     time.Now(),
						}, nil
					},
				)
			},
			checkResponse: func(t *testing.T, response *pb.CreateKeyPairResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, response.PublicKey)
				assert.NotNil(t, response.KeyIdentifier)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "failure",
			request: &pb.CreateKeyPairRequest{
				Passphrase: "test",
				ProjectId:  projectID.String(),
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateSigningKey(gomock.Any(), gomock.Any()).Return(db.SigningKey{}, assert.AnError)
			},
			checkResponse: func(t *testing.T, response *pb.CreateKeyPairResponse, err error) {
				t.Helper()

				assert.Error(t, err)
			},
			expectedStatusCode: codes.Internal,
		},
	}
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := &Server{
				store: mockStore,
				cfg: &config.Config{
					Salt: config.DefaultConfigForTest().Salt,
				},
			}

			response, err := server.CreateKeyPair(ctx, tc.request)
			tc.checkResponse(t, response, err)
		})
	}
}
