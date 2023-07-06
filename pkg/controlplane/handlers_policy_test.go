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

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreatePolicyDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreatePolicyRequest{
		Provider:         "github",
		GroupId:          1,
		Type:             pb.PolicyType_POLICY_TYPE_BRANCH_PROTECTION,
		PolicyDefinition: "key: value",
	}

	expectedPolicy := db.Policy{
		ID:               1,
		Provider:         "github",
		GroupID:          1,
		PolicyType:       db.PolicyTypePOLICYTYPEBRANCHPROTECTION,
		PolicyDefinition: "key: value",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().
		CreatePolicy(ctx, gomock.Any()).
		Return(expectedPolicy, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreatePolicy(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedPolicy.ID, response.Policy.Id)
	assert.Equal(t, expectedPolicy.GroupID, response.Policy.GroupId)
	assert.Equal(t, expectedPolicy.Provider, response.Policy.Provider)
	assert.Equal(t, expectedPolicy.PolicyType, db.PolicyType("POLICY_TYPE_BRANCH_PROTECTION"))
	assert.Equal(t, expectedPolicy.PolicyDefinition, response.Policy.PolicyDefinition)
	expectedCreatedAt := expectedPolicy.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Policy.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedPolicy.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Policy.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreatePolicy_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.CreatePolicyRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreatePolicyResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreatePolicyRequest{
				Provider:         "github",
				GroupId:          1,
				Type:             pb.PolicyType_POLICY_TYPE_BRANCH_PROTECTION,
				PolicyDefinition: "key: value",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreatePolicy(gomock.Any(), gomock.Any()).
					Return(db.Policy{
						ID:               1,
						Provider:         "github",
						GroupID:          1,
						PolicyType:       db.PolicyTypePOLICYTYPEBRANCHPROTECTION,
						PolicyDefinition: "key: value",
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreatePolicyResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Policy.Id)
				assert.Equal(t, int32(1), res.Policy.GroupId)
				assert.Equal(t, "github", res.Policy.Provider)
				assert.Equal(t, pb.PolicyType_POLICY_TYPE_BRANCH_PROTECTION, res.Policy.Type)
				assert.Equal(t, "key: value", res.Policy.PolicyDefinition)
				assert.NotNil(t, res.Policy.CreatedAt)
				assert.NotNil(t, res.Policy.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req:  &pb.CreatePolicyRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.CreatePolicyResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
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

			server := NewServer(mockStore, &config.Config{})

			resp, err := server.CreatePolicy(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeletePolicyDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.DeletePolicyRequest{Id: 1}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetPolicyByID(ctx, gomock.Any())
	mockStore.EXPECT().
		DeletePolicy(ctx, gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeletePolicy(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeletePolicy_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.DeletePolicyRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeletePolicyResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeletePolicyRequest{
				Id: 1,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPolicyByID(gomock.Any(), gomock.Any()).Return(db.Policy{}, nil).Times(1)
				store.EXPECT().
					DeletePolicy(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeletePolicyResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeletePolicyResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeletePolicyRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.DeletePolicyResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
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

			server := NewServer(mockStore, &config.Config{})

			resp, err := server.DeletePolicy(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPoliciesDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPoliciesRequest{Provider: "github", GroupId: 1}

	expectedPolicies := []db.Policy{
		{
			ID:               1,
			Provider:         "github",
			GroupID:          1,
			PolicyType:       db.PolicyTypePOLICYTYPEBRANCHPROTECTION,
			PolicyDefinition: "key: value",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().ListPoliciesByGroupID(ctx, gomock.Any()).
		Return(expectedPolicies, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicies(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedPolicies), len(response.Policies))
	assert.Equal(t, expectedPolicies[0].ID, response.Policies[0].Id)
	assert.Equal(t, expectedPolicies[0].Provider, response.Policies[0].Provider)
	assert.Equal(t, expectedPolicies[0].GroupID, response.Policies[0].GroupId)
	assert.Equal(t, expectedPolicies[0].PolicyType, db.PolicyType("POLICY_TYPE_BRANCH_PROTECTION"))
	assert.Equal(t, expectedPolicies[0].PolicyDefinition, response.Policies[0].PolicyDefinition)
}

func TestGetPolicies_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPoliciesRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPoliciesResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPoliciesRequest{Provider: "github", GroupId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListPoliciesByGroupID(gomock.Any(), gomock.Any()).
					Return([]db.Policy{
						{
							ID:               1,
							Provider:         "github",
							GroupID:          1,
							PolicyType:       db.PolicyTypePOLICYTYPEBRANCHPROTECTION,
							PolicyDefinition: "key: value",
							CreatedAt:        time.Now(),
							UpdatedAt:        time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPoliciesResponse, err error) {
				expectedPolicies := []db.Policy{
					{
						ID:               1,
						Provider:         "github",
						GroupID:          1,
						PolicyType:       db.PolicyTypePOLICYTYPEBRANCHPROTECTION,
						PolicyDefinition: "key: value",
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedPolicies), len(res.Policies))
				assert.Equal(t, expectedPolicies[0].ID, res.Policies[0].Id)
				assert.Equal(t, expectedPolicies[0].Provider, res.Policies[0].Provider)
				assert.Equal(t, expectedPolicies[0].GroupID, res.Policies[0].GroupId)
				assert.Equal(t, expectedPolicies[0].PolicyType, db.PolicyType("POLICY_TYPE_BRANCH_PROTECTION"))
				assert.Equal(t, expectedPolicies[0].PolicyDefinition, res.Policies[0].PolicyDefinition)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
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

			server := NewServer(mockStore, &config.Config{})

			resp, err := server.GetPolicies(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
