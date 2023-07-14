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
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestKeysHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateKeyPairRequest{
		Passphrase: "test",
		GroupId:    1,
	}

	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
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
				GroupID:       params.GroupID,
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
			Salt: config.GetCryptoConfigWithDefaults(),
		},
	}

	response, err := s.CreateKeyPair(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response.PublicKey)
	assert.NotNil(t, response.KeyIdentifier)
	// Add more assertions as needed
}

func TestKeysHandler_gRPC(t *testing.T) {
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
				GroupId:    1,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateSigningKey(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, params db.CreateSigningKeyParams) (db.SigningKey, error) {
						return db.SigningKey{
							ID:            1,
							GroupID:       params.GroupID,
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
				GroupId:    1,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateSigningKey(gomock.Any(), gomock.Any()).Return(db.SigningKey{}, assert.AnError)
			},
			checkResponse: func(t *testing.T, response *pb.CreateKeyPairResponse, err error) {
				assert.Error(t, err)
			},
			expectedStatusCode: codes.Internal,
		},
	}
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:              1,
		OrganizationId:      1,
		GroupIds:            []int32{1},
		NeedsPasswordChange: false,
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := &Server{
				store: mockStore,
				cfg: &config.Config{
					Salt: config.GetCryptoConfigWithDefaults(),
				},
			}

			response, err := server.CreateKeyPair(ctx, tc.request)
			tc.checkResponse(t, response, err)
		})
	}
}
