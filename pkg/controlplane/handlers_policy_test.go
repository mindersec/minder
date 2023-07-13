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
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	github "github.com/stacklok/mediator/pkg/providers/github"
)

const policyDefinitionJson = `{
  "branches": [
    {
      "name": "main",
      "rules": {
        "pull_request_reviews_enforcement_level": "everyone"
      }
    }
  ]
}`
const policyDefinition = `branches:
    - name: main
      rules:
        pull_request_reviews_enforcement_level: everyone
`

func TestCreatePolicyDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreatePolicyRequest{
		Provider:         "github",
		GroupId:          1,
		Type:             "branch_protection",
		PolicyDefinition: policyDefinitionJson,
	}

	expectedPolicy := db.Policy{
		ID:               1,
		Provider:         "github",
		GroupID:          1,
		PolicyType:       1,
		PolicyDefinition: json.RawMessage(policyDefinitionJson),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	policyTypes := []db.PolicyType{
		{ID: 1, Provider: github.Github, PolicyType: "branch_protection", Version: "1.0.0"},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetPolicyTypes(ctx, gomock.Any()).Return(policyTypes, nil)
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
	assert.Equal(t, response.Policy.Type, "branch_protection")
	assert.Equal(t, response.Policy.PolicyDefinition, policyDefinition)
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
				Type:             "branch_protection",
				PolicyDefinition: policyDefinitionJson,
			},
			buildStubs: func(store *mockdb.MockStore) {
				policyTypes := []db.PolicyType{
					{ID: 1, Provider: github.Github, PolicyType: "branch_protection", Version: "1.0.0"},
				}
				store.EXPECT().GetPolicyTypes(gomock.Any(), gomock.Any()).Return(policyTypes, nil)
				store.EXPECT().
					CreatePolicy(gomock.Any(), gomock.Any()).
					Return(db.Policy{
						ID:               1,
						Provider:         "github",
						GroupID:          1,
						PolicyType:       1,
						PolicyDefinition: json.RawMessage(policyDefinitionJson),
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
				assert.Equal(t, "branch_protection", res.Policy.Type)
				assert.Equal(t, policyDefinition, res.Policy.PolicyDefinition)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

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
				store.EXPECT().GetPolicyByID(gomock.Any(), gomock.Any())
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

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

	expectedPolicies := []db.ListPoliciesByGroupIDRow{
		{
			ID:               1,
			Provider:         "github",
			GroupID:          1,
			PolicyType:       1,
			PolicyDefinition: json.RawMessage(policyDefinitionJson),
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

	mockStore.EXPECT().ListPoliciesByGroupID(ctx, gomock.Any()).Return(expectedPolicies, nil)

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
	assert.Equal(t, response.Policies[0].PolicyDefinition, policyDefinition)
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
				expectedPolicies := []db.ListPoliciesByGroupIDRow{
					{
						ID:               1,
						Provider:         "github",
						GroupID:          1,
						PolicyType:       1,
						PolicyDefinition: json.RawMessage(policyDefinitionJson),
						CreatedAt:        time.Now(),
						UpdatedAt:        time.Now(),
					},
				}

				store.EXPECT().ListPoliciesByGroupID(gomock.Any(), gomock.Any()).
					Return(expectedPolicies, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPoliciesResponse, err error) {
				expectedPolicies := []db.ListPoliciesByGroupIDRow{
					{
						ID:               1,
						Provider:         "github",
						GroupID:          1,
						PolicyType:       1,
						PolicyDefinition: json.RawMessage(`{"key": "value"}`),
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
				assert.Equal(t, res.Policies[0].PolicyDefinition, policyDefinition)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicies(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPolicyStatusByIdDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPolicyStatusByIdRequest{PolicyId: 1}

	expectedStatus := []db.GetPolicyStatusByIdRow{
		{
			PolicyType:   "branch_protection",
			RepoID:       1,
			RepoOwner:    "foo",
			RepoName:     "bar",
			PolicyStatus: "success",
			LastUpdated:  time.Now(),
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

	mockStore.EXPECT().GetPolicyByID(ctx, gomock.Any())
	mockStore.EXPECT().GetPolicyStatusById(ctx, gomock.Any()).Return(expectedStatus, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicyStatusById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedStatus), len(response.PolicyRepoStatus))
	assert.Equal(t, expectedStatus[0].PolicyType, response.PolicyRepoStatus[0].PolicyType)
	assert.Equal(t, expectedStatus[0].RepoID, response.PolicyRepoStatus[0].RepoId)
	assert.Equal(t, expectedStatus[0].RepoOwner, response.PolicyRepoStatus[0].RepoOwner)
	assert.Equal(t, expectedStatus[0].RepoName, response.PolicyRepoStatus[0].RepoName)
	assert.Equal(t, "success", response.PolicyRepoStatus[0].PolicyStatus)
}

func TestGetPolicyStatusById_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPolicyStatusByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPolicyStatusByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPolicyStatusByIdRequest{PolicyId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				expectedStatus := []db.GetPolicyStatusByIdRow{
					{
						PolicyType:   "branch_protection",
						RepoID:       1,
						RepoOwner:    "foo",
						RepoName:     "bar",
						PolicyStatus: "success",
						LastUpdated:  time.Now(),
					},
				}

				store.EXPECT().GetPolicyByID(gomock.Any(), gomock.Any())
				store.EXPECT().GetPolicyStatusById(gomock.Any(), gomock.Any()).
					Return(expectedStatus, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPolicyStatusByIdResponse, err error) {
				expectedStatus := []db.GetPolicyStatusByIdRow{
					{
						PolicyType:   "branch_protection",
						RepoID:       1,
						RepoOwner:    "foo",
						RepoName:     "bar",
						PolicyStatus: db.PolicyStatusTypes("success"),
						LastUpdated:  time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedStatus), len(res.PolicyRepoStatus))
				assert.Equal(t, expectedStatus[0].PolicyType, res.PolicyRepoStatus[0].PolicyType)
				assert.Equal(t, expectedStatus[0].RepoID, res.PolicyRepoStatus[0].RepoId)
				assert.Equal(t, expectedStatus[0].RepoOwner, res.PolicyRepoStatus[0].RepoOwner)
				assert.Equal(t, expectedStatus[0].RepoName, res.PolicyRepoStatus[0].RepoName)
				assert.Equal(t, "success", res.PolicyRepoStatus[0].PolicyStatus)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicyStatusById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPolicyStatusByRepositoryIdDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPolicyStatusByRepositoryRequest{RepositoryId: 1}

	expectedStatus := []db.GetPolicyStatusByRepositoryIdRow{
		{
			PolicyType:   "branch_protection",
			RepoID:       1,
			RepoOwner:    "foo",
			RepoName:     "bar",
			PolicyStatus: "success",
			LastUpdated:  time.Now(),
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

	mockStore.EXPECT().GetRepositoryByID(ctx, gomock.Any())
	mockStore.EXPECT().GetPolicyStatusByRepositoryId(ctx, gomock.Any()).Return(expectedStatus, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicyStatusByRepository(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedStatus), len(response.PolicyRepoStatus))
	assert.Equal(t, expectedStatus[0].PolicyType, response.PolicyRepoStatus[0].PolicyType)
	assert.Equal(t, expectedStatus[0].RepoID, response.PolicyRepoStatus[0].RepoId)
	assert.Equal(t, expectedStatus[0].RepoOwner, response.PolicyRepoStatus[0].RepoOwner)
	assert.Equal(t, expectedStatus[0].RepoName, response.PolicyRepoStatus[0].RepoName)
	assert.Equal(t, "success", response.PolicyRepoStatus[0].PolicyStatus)
}

func TestGetPolicyStatusByRepositoryId_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPolicyStatusByRepositoryRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPolicyStatusByRepositoryResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPolicyStatusByRepositoryRequest{RepositoryId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				expectedStatus := []db.GetPolicyStatusByRepositoryIdRow{
					{
						PolicyType:   "branch_protection",
						RepoID:       1,
						RepoOwner:    "foo",
						RepoName:     "bar",
						PolicyStatus: "success",
						LastUpdated:  time.Now(),
					},
				}

				store.EXPECT().GetRepositoryByID(gomock.Any(), gomock.Any())
				store.EXPECT().GetPolicyStatusByRepositoryId(gomock.Any(), gomock.Any()).
					Return(expectedStatus, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPolicyStatusByRepositoryResponse, err error) {
				expectedStatus := []db.GetPolicyStatusByRepositoryIdRow{
					{
						PolicyType:   "branch_protection",
						RepoID:       1,
						RepoOwner:    "foo",
						RepoName:     "bar",
						PolicyStatus: db.PolicyStatusTypes("success"),
						LastUpdated:  time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedStatus), len(res.PolicyRepoStatus))
				assert.Equal(t, expectedStatus[0].PolicyType, res.PolicyRepoStatus[0].PolicyType)
				assert.Equal(t, expectedStatus[0].RepoID, res.PolicyRepoStatus[0].RepoId)
				assert.Equal(t, expectedStatus[0].RepoOwner, res.PolicyRepoStatus[0].RepoOwner)
				assert.Equal(t, expectedStatus[0].RepoName, res.PolicyRepoStatus[0].RepoName)
				assert.Equal(t, "success", res.PolicyRepoStatus[0].PolicyStatus)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicyStatusByRepository(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPolicyViolationsByIdDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPolicyViolationsByIdRequest{Id: 1}

	expectedViolations := []db.GetPolicyViolationsByIdRow{
		{
			PolicyType: "branch_protection",
			RepoID:     1,
			RepoOwner:  "foo",
			RepoName:   "bar",
			Metadata:   json.RawMessage(`{"foo": "bar"}`),
			Violation:  json.RawMessage(`{"key": "value"}`),
			CreatedAt:  time.Now(),
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

	mockStore.EXPECT().GetPolicyByID(ctx, gomock.Any())
	mockStore.EXPECT().GetPolicyViolationsById(ctx, gomock.Any()).
		Return(expectedViolations, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicyViolationsById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedViolations), len(response.PolicyViolation))
	assert.Equal(t, expectedViolations[0].PolicyType, response.PolicyViolation[0].PolicyType)
	assert.Equal(t, expectedViolations[0].RepoID, response.PolicyViolation[0].RepoId)
	assert.Equal(t, expectedViolations[0].RepoOwner, response.PolicyViolation[0].RepoOwner)
	assert.Equal(t, expectedViolations[0].RepoName, response.PolicyViolation[0].RepoName)
	assert.Equal(t, `{"key": "value"}`, response.PolicyViolation[0].Violation)
	assert.Equal(t, `{"foo": "bar"}`, response.PolicyViolation[0].Metadata)
}

func TestGetViolationsById_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPolicyViolationsByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPolicyViolationsByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPolicyViolationsByIdRequest{Id: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPolicyByID(gomock.Any(), gomock.Any())
				store.EXPECT().GetPolicyViolationsById(gomock.Any(), gomock.Any()).
					Return([]db.GetPolicyViolationsByIdRow{
						{
							PolicyType: "branch_protection",
							RepoID:     1,
							RepoOwner:  "foo",
							RepoName:   "bar",
							Metadata:   json.RawMessage(`{"foo": "bar"}`),
							Violation:  json.RawMessage(`{"key": "value"}`),
							CreatedAt:  time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPolicyViolationsByIdResponse, err error) {
				expectedViolations := []db.GetPolicyViolationsByIdRow{
					{
						PolicyType: "branch_protection",
						RepoID:     1,
						RepoOwner:  "foo",
						RepoName:   "bar",
						Metadata:   json.RawMessage(`{"foo": "bar"}`),
						Violation:  json.RawMessage(`{"key": "value"}`),
						CreatedAt:  time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, expectedViolations[0].PolicyType, res.PolicyViolation[0].PolicyType)
				assert.Equal(t, expectedViolations[0].RepoID, res.PolicyViolation[0].RepoId)
				assert.Equal(t, expectedViolations[0].RepoOwner, res.PolicyViolation[0].RepoOwner)
				assert.Equal(t, expectedViolations[0].RepoName, res.PolicyViolation[0].RepoName)
				assert.Equal(t, `{"key": "value"}`, res.PolicyViolation[0].Violation)
				assert.Equal(t, `{"foo": "bar"}`, res.PolicyViolation[0].Metadata)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicyViolationsById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPolicyViolationsByGroupDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPolicyViolationsByGroupRequest{Provider: github.Github, GroupId: 1}

	expectedViolations := []db.GetPolicyViolationsByGroupRow{
		{
			PolicyType: "branch_protection",
			RepoID:     1,
			RepoOwner:  "foo",
			RepoName:   "bar",
			Metadata:   json.RawMessage(`{"foo": "bar"}`),
			Violation:  json.RawMessage(`{"key": "value"}`),
			CreatedAt:  time.Now(),
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

	mockStore.EXPECT().GetPolicyViolationsByGroup(ctx, gomock.Any()).
		Return(expectedViolations, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicyViolationsByGroup(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedViolations), len(response.PolicyViolation))
	assert.Equal(t, expectedViolations[0].PolicyType, response.PolicyViolation[0].PolicyType)
	assert.Equal(t, expectedViolations[0].RepoID, response.PolicyViolation[0].RepoId)
	assert.Equal(t, expectedViolations[0].RepoOwner, response.PolicyViolation[0].RepoOwner)
	assert.Equal(t, expectedViolations[0].RepoName, response.PolicyViolation[0].RepoName)
	assert.Equal(t, `{"key": "value"}`, response.PolicyViolation[0].Violation)
	assert.Equal(t, `{"foo": "bar"}`, response.PolicyViolation[0].Metadata)
}

func TestGetViolationsByGroup_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPolicyViolationsByGroupRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPolicyViolationsByGroupResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPolicyViolationsByGroupRequest{Provider: github.Github, GroupId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetPolicyViolationsByGroup(gomock.Any(), gomock.Any()).
					Return([]db.GetPolicyViolationsByGroupRow{
						{
							PolicyType: "branch_protection",
							RepoID:     1,
							RepoOwner:  "foo",
							RepoName:   "bar",
							Metadata:   json.RawMessage(`{"foo": "bar"}`),
							Violation:  json.RawMessage(`{"key": "value"}`),
							CreatedAt:  time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPolicyViolationsByGroupResponse, err error) {
				expectedViolations := []db.GetPolicyViolationsByGroupRow{
					{
						PolicyType: "branch_protection",
						RepoID:     1,
						RepoOwner:  "foo",
						RepoName:   "bar",
						Metadata:   json.RawMessage(`{"foo": "bar"}`),
						Violation:  json.RawMessage(`{"key": "value"}`),
						CreatedAt:  time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, expectedViolations[0].PolicyType, res.PolicyViolation[0].PolicyType)
				assert.Equal(t, expectedViolations[0].RepoID, res.PolicyViolation[0].RepoId)
				assert.Equal(t, expectedViolations[0].RepoOwner, res.PolicyViolation[0].RepoOwner)
				assert.Equal(t, expectedViolations[0].RepoName, res.PolicyViolation[0].RepoName)
				assert.Equal(t, `{"key": "value"}`, res.PolicyViolation[0].Violation)
				assert.Equal(t, `{"foo": "bar"}`, res.PolicyViolation[0].Metadata)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicyViolationsByGroup(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetPolicyViolationsByRepositoryDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetPolicyViolationsByRepositoryRequest{RepositoryId: 1}

	expectedViolations := []db.GetPolicyViolationsByRepositoryIdRow{
		{
			PolicyType: "branch_protection",
			RepoID:     1,
			RepoOwner:  "foo",
			RepoName:   "bar",
			Metadata:   json.RawMessage(`{"foo": "bar"}`),
			Violation:  json.RawMessage(`{"key": "value"}`),
			CreatedAt:  time.Now(),
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

	mockStore.EXPECT().GetRepositoryByID(ctx, gomock.Any())
	mockStore.EXPECT().GetPolicyViolationsByRepositoryId(ctx, gomock.Any()).
		Return(expectedViolations, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetPolicyViolationsByRepositoryId(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedViolations), len(response.PolicyViolation))
	assert.Equal(t, expectedViolations[0].PolicyType, response.PolicyViolation[0].PolicyType)
	assert.Equal(t, expectedViolations[0].RepoID, response.PolicyViolation[0].RepoId)
	assert.Equal(t, expectedViolations[0].RepoOwner, response.PolicyViolation[0].RepoOwner)
	assert.Equal(t, expectedViolations[0].RepoName, response.PolicyViolation[0].RepoName)
	assert.Equal(t, `{"key": "value"}`, response.PolicyViolation[0].Violation)
	assert.Equal(t, `{"foo": "bar"}`, response.PolicyViolation[0].Metadata)
}

func TestGetViolationsByRepositoryId_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetPolicyViolationsByRepositoryRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetPolicyViolationsByRepositoryResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetPolicyViolationsByRepositoryRequest{RepositoryId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRepositoryByID(gomock.Any(), gomock.Any())
				store.EXPECT().GetPolicyViolationsByRepositoryId(gomock.Any(), gomock.Any()).
					Return([]db.GetPolicyViolationsByRepositoryIdRow{
						{
							PolicyType: "branch_protection",
							RepoID:     1,
							RepoOwner:  "foo",
							RepoName:   "bar",
							Metadata:   json.RawMessage(`{"foo": "bar"}`),
							Violation:  json.RawMessage(`{"key": "value"}`),
							CreatedAt:  time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetPolicyViolationsByRepositoryResponse, err error) {
				expectedViolations := []db.GetPolicyViolationsByRepositoryIdRow{
					{
						PolicyType: "branch_protection",
						RepoID:     1,
						RepoOwner:  "foo",
						RepoName:   "bar",
						Metadata:   json.RawMessage(`{"foo": "bar"}`),
						Violation:  json.RawMessage(`{"key": "value"}`),
						CreatedAt:  time.Now(),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, len(expectedViolations), len(res.PolicyViolation))
				assert.Equal(t, expectedViolations[0].PolicyType, res.PolicyViolation[0].PolicyType)
				assert.Equal(t, expectedViolations[0].RepoID, res.PolicyViolation[0].RepoId)
				assert.Equal(t, expectedViolations[0].RepoOwner, res.PolicyViolation[0].RepoOwner)
				assert.Equal(t, expectedViolations[0].RepoName, res.PolicyViolation[0].RepoName)
				assert.Equal(t, `{"key": "value"}`, res.PolicyViolation[0].Violation)
				assert.Equal(t, `{"foo": "bar"}`, res.PolicyViolation[0].Metadata)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetPolicyViolationsByRepositoryId(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
