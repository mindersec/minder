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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/internal/util"
)

func TestLogin_gRPC(t *testing.T) {
	seed := time.Now().UnixNano()
	password := util.RandomPassword(8, seed)
	cryptcfg := config.GetCryptoConfigWithDefaults()
	hash, err := mcrypto.GeneratePasswordHash(password, &cryptcfg)
	if err != nil {
		t.Fatalf("Error generating password hash: %v", err)
	}

	user := db.User{
		ID:       1,
		Username: "test",
		Password: hash,
	}

	// prepare keys for signing tokens
	viper.SetDefault("auth.access_token_private_key", "access_token_private.pem")
	viper.SetDefault("auth.refresh_token_private_key", "refresh_token_private.pem")
	err = util.RandomPrivateKeyFile(2048, "access_token_private.pem")
	if err != nil {
		t.Fatalf("Error generating access token private key: %v", err)
	}
	err = util.RandomPrivateKeyFile(2048, "refresh_token_private.pem")
	if err != nil {
		t.Fatalf("Error generating refresh token private key: %v", err)
	}

	testCases := []struct {
		name               string
		req                *pb.LogInRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.LogInResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.LogInRequest{
				Username: "test",
				Password: password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserGroups(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())
				store.EXPECT().
					GetUserByUserName(gomock.Any(), gomock.Any()).
					Times(1).Return(user, nil)
				store.EXPECT().CleanTokenIat(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.LogInResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
		},
		{
			name: "EmptyRequest",
			req: &pb.LogInRequest{
				Username: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.LogInResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.LogIn(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}

	_ = os.Remove(filepath.Join(".", "access_token_private.pem"))
	_ = os.Remove(filepath.Join(".", "refresh_token_private.pem"))
}

func TestLogout_gRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().RevokeUserToken(gomock.Any(), gomock.Any())

	server, err := NewServer(mockStore, &config.Config{})
	require.NoError(t, err, "failed to create test server")

	res, err := server.LogOut(ctx, &pb.LogOutRequest{})

	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestRevokeTokens_gRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().RevokeUsersTokens(gomock.Any(), gomock.Any())

	server, err := NewServer(mockStore, &config.Config{})
	require.NoError(t, err, "failed to create test server")

	res, err := server.RevokeTokens(ctx, &pb.RevokeTokensRequest{})

	assert.NoError(t, err)
	assert.NotNil(t, res)
}

func TestRefreshToken_gRPC(t *testing.T) {
	// prepare keys for signing tokens
	viper.SetDefault("auth.access_token_private_key", "access_token_private.pem")
	viper.SetDefault("auth.access_token_public_key", "access_token_public.pem")
	viper.SetDefault("auth.refresh_token_private_key", "refresh_token_private.pem")
	viper.SetDefault("auth.refresh_token_public_key", "refresh_token_public.pem")
	viper.SetDefault("auth.token_expiry", 3600)
	viper.SetDefault("auth.refresh_expiry", 86400)
	err := util.RandomKeypairFile(2048, viper.Get("auth.access_token_private_key").(string), viper.Get("auth.access_token_public_key").(string))
	if err != nil {
		t.Fatalf("Error generating access token private key: %v", err)
	}

	err = util.RandomKeypairFile(2048, viper.Get("auth.refresh_token_private_key").(string), viper.Get("auth.refresh_token_public_key").(string))
	if err != nil {
		t.Fatalf("Error generating refresh token private key: %v", err)
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStoreToken := mockdb.NewMockStore(ctrl)
	mockStore := mockdb.NewMockStore(ctrl)

	ctxToken := context.Background()
	// mocked calls
	mockStoreToken.EXPECT().GetUserByID(ctxToken, gomock.Any())
	mockStoreToken.EXPECT().GetUserGroups(ctxToken, gomock.Any())
	mockStoreToken.EXPECT().GetUserRoles(ctxToken, gomock.Any())
	// generate a token
	_, refreshToken, _, _, _, err := generateToken(ctxToken, mockStoreToken, 1)
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"refresh-token": refreshToken,
	})

	// Create a new context with added header metadata
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, md)
	server, err := NewServer(mockStore, &config.Config{})
	require.NoError(t, err, "failed to create test server")
	mockStore.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).Times(2)
	mockStore.EXPECT().GetUserGroups(gomock.Any(), gomock.Any())
	mockStore.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())

	// validate the status of the output
	res, err := server.RefreshToken(ctx, &pb.RefreshTokenRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotNil(t, res.AccessToken)

	_ = os.Remove(filepath.Join(".", viper.Get("auth.refresh_token_private_key").(string)))
	_ = os.Remove(filepath.Join(".", viper.Get("auth.refresh_token_public_key").(string)))
	_ = os.Remove(filepath.Join(".", viper.Get("auth.access_token_private_key").(string)))
	_ = os.Remove(filepath.Join(".", viper.Get("auth.access_token_public_key").(string)))
}
