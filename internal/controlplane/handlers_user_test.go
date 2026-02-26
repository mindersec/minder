// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/testing/protocmp"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	mockjwt "github.com/mindersec/minder/internal/auth/jwt/mock"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/authz/mock"
	mockcrypto "github.com/mindersec/minder/internal/crypto/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	fake "github.com/mindersec/minder/internal/invites/test"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/providers"
	mockprov "github.com/mindersec/minder/internal/providers/github/service/mock"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer"
)

const (
	//nolint:gosec // not credentials, just an endpoint
	tokenEndpoint = "/realms/stacklok/protocol/openid-connect/token"
)

func TestCreateUser_gRPC(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	keyCloakUserToken := openid.New()
	require.NoError(t, keyCloakUserToken.Set("gh_id", "31337"))

	testCases := []struct {
		name       string
		req        *pb.CreateUserRequest
		buildStubs func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
			prov *mockprov.MockGitHubProviderService) context.Context
		checkResponse      func(t *testing.T, res *pb.CreateUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, jwt *mockjwt.MockValidator,
				_ *mockprov.MockGitHubProviderService) context.Context {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					CreateProjectWithID(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID:   projectID,
						Name: "subject1",
					}, nil)

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)
				store.EXPECT().CreateEntitlements(gomock.Any(), gomock.Any()).
					Return(nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "subject1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "Success with pending App",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
				prov *mockprov.MockGitHubProviderService) context.Context {
				ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)

				store.EXPECT().
					GetUnclaimedInstallationsByUser(gomock.Any(), sql.NullString{String: "31337", Valid: true}).
					Return([]db.ProviderGithubAppInstallation{
						{
							AppInstallationID: 10,
							OrganizationID:    9000,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						},
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(10)).
					Return(&db.Project{
						ID:   projectID,
						Name: "github-org1",
					}, nil)

				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				validator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "github-org1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "Success with two pending Apps",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
				prov *mockprov.MockGitHubProviderService) context.Context {
				ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)

				store.EXPECT().
					GetUnclaimedInstallationsByUser(gomock.Any(), sql.NullString{String: "31337", Valid: true}).
					Return([]db.ProviderGithubAppInstallation{
						{
							AppInstallationID: 10,
							OrganizationID:    9000,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						}, {
							AppInstallationID: 11,
							OrganizationID:    9001,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						},
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(10)).
					Return(&db.Project{
						ID:   projectID,
						Name: "github-org1",
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(11)).
					Return(&db.Project{
						ID:   uuid.New(),
						Name: "github-org2",
					}, nil)

				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				validator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "github-org1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockValidator(ctrl)
			mockProviders := mockprov.NewMockGitHubProviderService(ctrl)
			reqCtx := tc.buildStubs(ctx, mockStore, mockJwtValidator, mockProviders)
			crypeng := mockcrypto.NewMockEngine(ctrl)

			authz := &mock.NoopClient{Authorized: true}
			server := &Server{
				store:        mockStore,
				cfg:          &serverconfig.Config{},
				cryptoEngine: crypeng,
				jwt:          mockJwtValidator,
				ghProviders:  mockProviders,
				authzClient:  authz,
				projectCreator: projects.NewProjectCreator(
					authz,
					marketplaces.NewNoopMarketplace(),
					&serverconfig.DefaultProfilesConfig{},
					&serverconfig.FeaturesConfig{},
				),
			}

			// server, err := NewServer(mockStore, evt, &serverconfig.Config{
			// 	Auth: serverconfig.AuthConfig{
			// 		TokenKey: generateTokenKey(t),
			// 	},
			// }, mockJwtValidator, ghProviders.NewProviderStore(mockStore))
			// require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateUser(reqCtx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockJwtValidator := mockjwt.NewMockValidator(ctrl)

	request := &pb.DeleteUserRequest{}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case tokenEndpoint:
			data := oauth2.Token{
				AccessToken: "some-token",
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(data)
			if err != nil {
				t.Fatal(err)
			}
		case "/admin/realms/stacklok/users/subject1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
		}
	}))
	defer testServer.Close()

	tokenResult, _ := openid.NewBuilder().Subject("subject1").Build()
	mockJwtValidator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

	tx := sql.Tx{}
	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		GetUserBySubject(gomock.Any(), "subject1").
		Return(db.User{IdentitySubject: "subject1"}, nil)
	mockStore.EXPECT().
		DeleteUser(gomock.Any(), gomock.Any()).
		Return(nil)
	mockStore.EXPECT().Commit(gomock.Any())
	// we expect rollback to be called even if there is no error (through defer), in that case it will be a no-op
	mockStore.EXPECT().Rollback(gomock.Any())

	crypeng := mockcrypto.NewMockEngine(ctrl)

	server := &Server{
		store: mockStore,
		cfg: &serverconfig.Config{
			Identity: serverconfig.IdentityConfigWrapper{
				Server: serverconfig.IdentityConfig{
					IssuerUrl:    testServer.URL,
					Realm:        "stacklok",
					ClientId:     "client-id",
					ClientSecret: "client-secret",
				},
			},
		},
		jwt:          mockJwtValidator,
		cryptoEngine: crypeng,
		authzClient:  &mock.NoopClient{Authorized: true},
	}

	response, err := server.DeleteUser(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteUser_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.DeleteUserRequest
		buildStubs         func(store *mockdb.MockStore, jwt *mockjwt.MockValidator)
		checkResponse      func(t *testing.T, res *pb.DeleteUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.DeleteUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockValidator) {
				tokenResult, _ := openid.NewBuilder().Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					GetUserBySubject(gomock.Any(), "subject1").
					Return(db.User{
						IdentitySubject: "subject1",
					}, nil)
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Any()).
					Return(nil)
				store.EXPECT().Commit(gomock.Any())
				// we expect rollback to be called even if there is no error (through defer), in that case it will be a no-op
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.DeleteUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteUserResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockValidator(ctrl)
			tc.buildStubs(mockStore, mockJwtValidator)

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case tokenEndpoint:
					data := oauth2.Token{
						AccessToken: "some-token",
					}
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(data)
					if err != nil {
						t.Fatal(err)
					}
				case "/admin/realms/stacklok/users/subject1":
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
				}
			}))
			defer testServer.Close()

			evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")

			server := &Server{
				evt:           evt,
				store:         mockStore,
				jwt:           mockJwtValidator,
				providerStore: providers.NewProviderStore(mockStore),
				authzClient:   &mock.SimpleClient{},
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{
						TokenKey: generateTokenKey(t),
					},
					Identity: serverconfig.IdentityConfigWrapper{
						Server: serverconfig.IdentityConfig{
							IssuerUrl:    testServer.URL,
							Realm:        "stacklok",
							ClientId:     "client-id",
							ClientSecret: "client-secret",
						},
					},
				},
			}

			resp, err := server.DeleteUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestListInvitations(t *testing.T) {
	t.Parallel()

	userEmail := "user@example.com"
	otherUser := "other@example.com"
	project := uuid.New()
	project2 := uuid.New()
	identitySubject := "subject1"

	inviteCtx := auth.WithIdentityContext(context.Background(), &auth.Identity{
		UserID:    "sponsor",
		HumanName: "Sponsor",
	})
	fakeInviteService := fake.NewFakeInviteService()
	invite, err := fakeInviteService.CreateInvite(
		inviteCtx, nil, nil, serverconfig.EmailConfig{}, project, authz.RoleViewer, userEmail)
	require.NoError(t, err)
	// Add a second invite to project 2
	invite2, err := fakeInviteService.CreateInvite(
		inviteCtx, nil, nil, serverconfig.EmailConfig{}, project2, authz.RoleAdmin, userEmail)
	require.NoError(t, err)
	// And invite another user to project 1
	invite3, err := fakeInviteService.CreateInvite(
		inviteCtx, nil, nil, serverconfig.EmailConfig{}, project, authz.RoleAdmin, otherUser)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		caller         *auth.Identity
		expectedError  string
		expectedResult []*pb.Invitation
	}{
		{
			name: "Main user",
			caller: &auth.Identity{
				UserID:    identitySubject,
				HumanName: userEmail, // fake uses the email as the human name
			},
			expectedResult: []*pb.Invitation{
				{
					Project:        project.String(),
					ProjectDisplay: "Test: " + project.String(),
					Code:           invite.GetCode(),
					Role:           authz.RoleViewer.String(),
					Email:          userEmail,
					Sponsor:        "sponsor",
					SponsorDisplay: "Sponsor",
				},
				{
					Project:        project2.String(),
					ProjectDisplay: "Test: " + project2.String(),
					Code:           invite2.GetCode(),
					Role:           authz.RoleAdmin.String(),
					Email:          userEmail,
					Sponsor:        "sponsor",
					SponsorDisplay: "Sponsor",
				},
			},
		},
		{
			name: "Other user",
			caller: &auth.Identity{
				UserID:    "other-subject",
				HumanName: otherUser, // fake uses the email as the human name
			},
			expectedResult: []*pb.Invitation{
				{
					Project:        project.String(),
					ProjectDisplay: "Test: " + project.String(),
					Code:           invite3.GetCode(),
					Role:           authz.RoleAdmin.String(),
					Email:          otherUser,
					Sponsor:        "sponsor",
					SponsorDisplay: "Sponsor",
				},
			},
		},
		{
			name: "No invitations",
			caller: &auth.Identity{
				UserID:    "no-invites-subject",
				HumanName: "nobody@loves.me", // fake uses the email as the human name
			},
			expectedResult: []*pb.Invitation{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := &Server{
				invites: fakeInviteService,
			}

			ctx := context.Background()
			ctx = auth.WithIdentityContext(ctx, tc.caller)

			response, err := server.ListInvitations(ctx, &pb.ListInvitationsRequest{})

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tc.expectedResult), len(response.Invitations))
			diff := cmp.Diff(tc.expectedResult, response.Invitations,
				protocmp.Transform(), protocmp.IgnoreFields(&pb.Invitation{}, "created_at", "expires_at"),
				cmpopts.SortSlices(func(a, b *pb.Invitation) bool {
					return a.Code < b.Code
				}))
			if diff != "" {
				t.Errorf("Response invitations mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolveInvitation(t *testing.T) {
	t.Parallel()

	userEmail := "user@example.com"
	userSubject := "subject1"
	userGitHubId := "31137"
	projectDisplayName := "Project"
	projectID := uuid.New()
	projectStr := projectID.String()
	projectMetadata, err := json.Marshal(
		projects.Metadata{Public: projects.PublicMetadataV1{DisplayName: projectDisplayName}},
	)
	require.NoError(t, err)
	inviterContext := context.Background()
	inviterContext = auth.WithIdentityContext(inviterContext, &auth.Identity{
		UserID:    "inviter",
		HumanName: "Inviter",
	})
	inviterContext = engcontext.WithEntityContext(inviterContext, &engcontext.EntityContext{
		Project: engcontext.Project{ID: projectID},
	})

	testCases := []struct {
		name            string
		accept          bool
		setup           func(fake *fake.FakeInviteService, store *mockdb.MockStore) string
		roleAssignments map[uuid.UUID][]*pb.RoleAssignment
		expectedError   string
	}{
		{
			name: "code not found",
			setup: func(_ *fake.FakeInviteService, store *mockdb.MockStore) string {
				store.EXPECT().BeginTransaction().Return(&sql.Tx{}, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().Rollback(gomock.Any())
				return "no-such-code"
			},
			expectedError: "invitation not found or already used",
		},
		{
			name: "user self resolving",
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				ctx := auth.WithIdentityContext(inviterContext, &auth.Identity{
					UserID:    userSubject,
					HumanName: userEmail, // Fake uses this to get email
				})
				invite, err := fake.CreateInvite(ctx, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleAdmin, userEmail)
				require.NoError(t, err)

				store.EXPECT().BeginTransaction().Return(&sql.Tx{}, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
			expectedError: "users cannot accept their own invitation",
		},
		{
			name: "expired invitation",
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				fake.Time = time.Now().Add(-10 * 24 * time.Hour)
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleAdmin, userEmail)
				fake.Time = time.Time{}
				require.NoError(t, err)

				store.EXPECT().BeginTransaction().Return(&sql.Tx{}, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
			expectedError: "invitation expired",
		},
		{
			name:   "Success accept",
			accept: true,
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleViewer, userEmail)
				require.NoError(t, err)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().GetUserBySubject(gomock.Any(), userSubject).Return(db.User{
					ID: 2,
				}, nil)
				store.EXPECT().GetProjectByID(gomock.Any(), projectID).Return(db.Project{
					Name:     "project1",
					Metadata: projectMetadata,
				}, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
		},
		{
			name:   "Success decline",
			accept: false,
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleViewer, userEmail)
				require.NoError(t, err)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().GetProjectByID(gomock.Any(), projectID).Return(db.Project{
					Name:     "project1",
					Metadata: projectMetadata,
				}, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
		},
		{
			name:   "Can't accept for current role",
			accept: true,
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleViewer, userEmail)
				require.NoError(t, err)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().GetUserBySubject(gomock.Any(), userSubject).Return(db.User{
					ID: 2,
				}, nil)
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
			roleAssignments: map[uuid.UUID][]*pb.RoleAssignment{
				projectID: {
					{
						Role:    authz.RoleViewer.String(),
						Subject: userSubject,
						Project: &projectStr,
					},
				},
			},
			expectedError: "user already has the same role in the project",
		},
		{
			name:   "Update role if accepting existing invite",
			accept: true,
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleEditor, userEmail)
				require.NoError(t, err)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().GetUserBySubject(gomock.Any(), userSubject).Return(db.User{
					ID: 2,
				}, nil)
				store.EXPECT().GetProjectByID(gomock.Any(), projectID).Return(db.Project{
					Name:     "project1",
					Metadata: projectMetadata,
				}, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
			roleAssignments: map[uuid.UUID][]*pb.RoleAssignment{
				projectID: {
					{
						Role:    authz.RoleViewer.String(),
						Subject: userSubject,
						Project: &projectStr,
					},
				},
			},
		},
		{
			name:   "Success create user",
			accept: true,
			setup: func(fake *fake.FakeInviteService, store *mockdb.MockStore) string {
				invite, err := fake.CreateInvite(inviterContext, nil, nil, serverconfig.EmailConfig{}, projectID, authz.RoleViewer, userEmail)
				require.NoError(t, err)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().GetUserBySubject(gomock.Any(), userSubject).Return(db.User{}, sql.ErrNoRows)
				store.EXPECT().CreateUser(gomock.Any(), userSubject).Return(db.User{ID: 2}, nil)
				store.EXPECT().GetUnclaimedInstallationsByUser(gomock.Any(), sql.NullString{String: userGitHubId, Valid: true}).Return(nil, nil)
				store.EXPECT().GetProjectByID(gomock.Any(), projectID).Return(db.Project{
					Name:     "project1",
					Metadata: projectMetadata,
				}, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				return invite.GetCode()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("email", userEmail))
			assert.NoError(t, user.Set("sub", userSubject))
			assert.NoError(t, user.Set("gh_id", userGitHubId))

			ctx := context.Background()
			ctx = jwt.WithAuthTokenContext(ctx, user)
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID:    userSubject,
				HumanName: userEmail, // Fake uses this to get email
			})
			code := "code"

			mockStore := mockdb.NewMockStore(ctrl)
			fakeInviteService := fake.NewFakeInviteService()
			if tc.setup != nil {
				code = tc.setup(fakeInviteService, mockStore)
			}

			authzClient := &mock.SimpleClient{
				Assignments: tc.roleAssignments,
			}

			server := &Server{
				store:       mockStore,
				invites:     fakeInviteService,
				authzClient: authzClient,
			}

			response, err := server.ResolveInvitation(ctx, &pb.ResolveInvitationRequest{
				Code:   code,
				Accept: tc.accept,
			})

			if tc.expectedError != "" {
				require.Error(t, err)
				var rpcErr *util.NiceStatus
				require.ErrorAs(t, err, &rpcErr)
				require.NotEqual(t, rpcErr.Code, codes.Unknown)
				require.Contains(t, rpcErr.Details, tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.accept, response.IsAccepted)

			// We shouldn't have any lingering invites
			require.Empty(t, fakeInviteService.GetAllInvites())
		})
	}
}
