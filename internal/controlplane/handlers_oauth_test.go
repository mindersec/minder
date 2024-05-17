// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	mockjwt "github.com/stacklok/minder/internal/auth/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	mockclients "github.com/stacklok/minder/internal/providers/github/clients/mock"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	ghService "github.com/stacklok/minder/internal/providers/github/service"
	mockprovsvc "github.com/stacklok/minder/internal/providers/github/service/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	// Test with CLI set
	cfg, err := auth.NewOAuthConfig("google", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("github", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("google", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("github", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	_, err = auth.NewOAuthConfig("invalid", true)
	if err == nil {
		t.Errorf("Expected error in newOAuthConfig, but got nil")
	}
}

func TestGetAuthorizationURL(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	githubProviderClass := "github"
	githubAppProviderClass := "github-app"
	nonGithubProviderName := "non-github"
	projectIdStr := projectID.String()

	testCases := []struct {
		name               string
		req                *pb.GetAuthorizationURLRequest
		buildStubs         func(store *mockdb.MockStore)
		getToken           func(openid.Token) openid.Token
		checkResponse      func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success OAuth",
			req: &pb.GetAuthorizationURLRequest{
				Context: &pb.Context{
					Provider: &githubProviderClass,
					Project:  &projectIdStr,
				},
				Port: 8080,
				Cli:  true,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateSessionState(gomock.Any(), partialDbParamsMatcher{db.CreateSessionStateParams{
						Provider:  "github",
						ProjectID: projectID,
					}}).
					Return(db.SessionStore{}, nil)
				store.EXPECT().
					DeleteSessionStateByProjectID(gomock.Any(), gomock.Any()).
					Return(nil)
			},

			checkResponse: func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error) {
				t.Helper()

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if res.Url == "" {
					t.Errorf("Unexpected response from GetAuthorizationURL: %v", res)
				}
			},

			expectedStatusCode: codes.OK,
		},
		{
			name: "Success GitHub App",
			req: &pb.GetAuthorizationURLRequest{
				Context: &pb.Context{
					Provider: &githubAppProviderClass,
					Project:  &projectIdStr,
				},
				Port: 8080,
				Cli:  true,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateSessionState(gomock.Any(), partialDbParamsMatcher{db.CreateSessionStateParams{
						Provider:  "github-app",
						ProjectID: projectID,
					}}).
					Return(db.SessionStore{}, nil)
				store.EXPECT().
					DeleteSessionStateByProjectID(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			checkResponse: func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error) {
				t.Helper()

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if res.Url == "" {
					t.Errorf("Unexpected response from GetAuthorizationURL: %v", res)
				}
			},

			expectedStatusCode: codes.OK,
		},
		{
			name: "Unsupported auth flow",
			req: &pb.GetAuthorizationURLRequest{
				Context: &pb.Context{
					Provider: &nonGithubProviderName,
					Project:  &projectIdStr,
				},
				Port: 8080,
				Cli:  true,
			},
			buildStubs: func(_ *mockdb.MockStore) {},

			checkResponse: func(t *testing.T, _ *pb.GetAuthorizationURLResponse, err error) {
				t.Helper()

				assert.Error(t, err, "Expected error in GetAuthorizationURL")
			},

			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "No GitHub id",
			req: &pb.GetAuthorizationURLRequest{
				Context: &pb.Context{
					Provider: &githubProviderClass,
					Project:  &projectIdStr,
				},
				Port: 8080,
				Cli:  true,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateSessionState(gomock.Any(), partialDbParamsMatcher{db.CreateSessionStateParams{
						Provider:   "github",
						ProjectID:  projectID,
						RemoteUser: sql.NullString{Valid: true, String: "31337"},
					}}).
					Return(db.SessionStore{}, nil)
				store.EXPECT().
					DeleteSessionStateByProjectID(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			getToken: func(tok openid.Token) openid.Token {
				if err := tok.Set("gh_id", "31337"); err != nil {
					t.Fatalf("Error setting gh_id: %v", err)
				}
				return tok
			},
			checkResponse: func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error) {
				t.Helper()

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if res.Url == "" {
					t.Errorf("Unexpected response from GetAuthorizationURL: %v", res)
				}
			},

			expectedStatusCode: codes.OK,
		},
	}

	rpcOptions := &pb.RpcOptions{
		TargetResource: pb.TargetResource_TARGET_RESOURCE_USER,
	}

	baseCtx := withRpcOptions(context.Background(), rpcOptions)

	userJWT := openid.New()
	if err := userJWT.Set("sub", "testuser"); err != nil {
		t.Fatalf("Error setting sub: %v", err)
	}

	// Set the entity context
	baseCtx = engine.WithEntityContext(baseCtx, &engine.EntityContext{
		Project: engine.Project{
			ID: projectID,
		},
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tok, err := userJWT.Clone()
			if err != nil {
				t.Fatalf("Failed to clone token: %v", err)
			}
			token := tok.(openid.Token)
			if tc.getToken != nil {
				token = tc.getToken(token)
			}
			ctx := auth.WithAuthTokenContext(baseCtx, token)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")

			tokenKeyPath := generateTokenKey(t)
			c := &serverconfig.Config{
				Auth: serverconfig.AuthConfig{
					TokenKey: tokenKeyPath,
				},
				Provider: serverconfig.ProviderConfig{
					GitHubApp: &serverconfig.GitHubAppConfig{
						AppName: "test-app",
					},
				},
			}
			mockJwt := mockjwt.NewMockJwtValidator(ctrl)

			server := &Server{
				store:               store,
				jwt:                 mockJwt,
				evt:                 evt,
				cfg:                 c,
				mt:                  metrics.NewNoopMetrics(),
				providerAuthFactory: auth.NewOAuthConfig,
			}

			res, err := server.GetAuthorizationURL(ctx, tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}

func TestProviderCallback(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	code := "0xefbeadde"

	testCases := []struct {
		name             string
		redirectUrl      string
		remoteUser       sql.NullString
		code             int
		existingProvider bool
		err              string
	}{{
		name:             "Success",
		redirectUrl:      "http://localhost:8080",
		existingProvider: true,
		code:             307,
	}, {
		name:             "Success with remote user",
		redirectUrl:      "http://localhost:8080",
		remoteUser:       sql.NullString{Valid: true, String: "31337"},
		existingProvider: true,
		code:             307,
	}, {
		name:             "Wrong remote userid",
		remoteUser:       sql.NullString{Valid: true, String: "1234"},
		existingProvider: true,
		code:             403,
		err:              "The provided login token was associated with a different GitHub user.\n",
	}, {
		name:             "No existing provider",
		redirectUrl:      "http://localhost:8080",
		existingProvider: false,
		code:             307,
	}}

	for _, tt := range testCases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := http.Request{}

			resp := httptest.ResponseRecorder{Body: new(bytes.Buffer)}
			params := map[string]string{"provider": "github"}

			stateBinary := make([]byte, 8)
			// Store a very large timestamp in the state to ensure it's not expired
			binary.BigEndian.PutUint64(stateBinary, 0x0fffffffffffffff)
			stateBinary = append(stateBinary, []byte(tc.name)...)
			state := base64.RawURLEncoding.EncodeToString(stateBinary)

			req.URL = &url.URL{
				RawQuery: url.Values{"state": {state}, "code": {code}}.Encode(),
			}

			oauthServer := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					err := json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "anAccessToken",
					})
					if err != nil {
						t.Fatalf("Failed to write response: %v", err)
					}
				}))
			defer oauthServer.Close()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)

			gh := mockgh.NewMockGitHub(ctrl)
			gh.EXPECT().GetUserId(gomock.Any()).Return(int64(31337), nil).AnyTimes()

			var clientFactory *mockclients.MockGitHubClientFactory
			if tc.remoteUser.String != "" {
				delegate := mockgh.NewMockDelegate(ctrl)
				delegate.EXPECT().
					GetUserId(gomock.Any()).
					Return(int64(31337), nil)
				clientFactory = mockclients.NewMockGitHubClientFactory(ctrl)
				clientFactory.EXPECT().
					BuildOAuthClient(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, delegate, nil)
			}

			s, _ := newDefaultServer(t, store, clientFactory)

			var err error
			encryptedUrlString, err := s.cryptoEngine.EncryptString(tc.redirectUrl)
			if err != nil {
				t.Fatalf("Failed to encrypt redirect URL: %v", err)
			}
			encryptedUrl := sql.NullString{
				Valid:  true,
				String: encryptedUrlString.EncodedData,
			}
			serialized, err := encryptedUrlString.Serialize()
			require.NoError(t, err)

			tx := sql.Tx{}
			store.EXPECT().BeginTransaction().Return(&tx, nil)
			store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

			gh.EXPECT().GetUserId(gomock.Any()).Return(int64(31337), nil).AnyTimes()

			store.EXPECT().GetProjectIDBySessionState(gomock.Any(), state).Return(
				db.GetProjectIDBySessionStateRow{
					ProjectID:   projectID,
					RedirectUrl: encryptedUrl,
					EncryptedRedirect: pqtype.NullRawMessage{
						RawMessage: serialized,
						Valid:      true,
					},
					RemoteUser: tc.remoteUser,
				}, nil)

			if tc.existingProvider {
				store.EXPECT().GetProviderByName(gomock.Any(), gomock.Any()).Return(
					db.Provider{
						Name:       "github",
						Implements: []db.ProviderType{db.ProviderTypeGithub},
						Version:    provinfv1.V1,
					}, nil)
			} else {
				store.EXPECT().GetProviderByName(gomock.Any(), gomock.Any()).Return(
					db.Provider{}, sql.ErrNoRows)
				store.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).Return(db.Provider{}, nil)
			}

			if tc.code < http.StatusBadRequest {
				store.EXPECT().UpsertAccessToken(gomock.Any(), gomock.Any()).Return(
					db.ProviderAccessToken{}, nil)
				store.EXPECT().Commit(gomock.Any())
			}
			store.EXPECT().Rollback(gomock.Any())

			t.Logf("Request: %+v", req.URL)
			s.providerAuthFactory = func(_ string, _ bool) (*oauth2.Config, error) {
				return &oauth2.Config{
					Endpoint: oauth2.Endpoint{
						TokenURL: oauthServer.URL,
					},
				}, nil
			}
			s.HandleOAuthCallback()(&resp, &req, params)

			t.Logf("Response: %v", resp.Code)
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			t.Logf("Body: %s", string(body))

			if resp.Code != tc.code {
				t.Errorf("Unexpected status code: %v", resp.Code)
			}
			if tc.code >= http.StatusMovedPermanently && tc.code < http.StatusBadRequest {
				if resp.Header().Get("Location") != tc.redirectUrl {
					t.Errorf("Unexpected redirect URL: %v", resp.Header().Get("Location"))
				}
			}
			if tc.err != "" {
				if string(body) != tc.err {
					t.Errorf("Unexpected error message: %q", string(body))
				}
			}
		})
	}
}

func TestHandleGitHubAppCallback(t *testing.T) {
	t.Parallel()

	stateBinary := make([]byte, 8)
	// Store a very large timestamp in the state to ensure it's not expired
	binary.BigEndian.PutUint64(stateBinary, 0x0fffffffffffffff)
	stateBinary = append(stateBinary, []byte("state-test")...)
	validState := base64.RawURLEncoding.EncodeToString(stateBinary)

	code := "test-code"
	installationID := int64(123456)

	testCases := []struct {
		name          string
		state         string
		setupAction   string
		buildStubs    func(store *mockdb.MockStore, service *mockprovsvc.MockGitHubProviderService, gh *mockgh.MockClientService)
		checkResponse func(t *testing.T, resp httptest.ResponseRecorder)
	}{
		{
			name:  "Success with state",
			state: validState,
			buildStubs: func(store *mockdb.MockStore, service *mockprovsvc.MockGitHubProviderService, _ *mockgh.MockClientService) {
				service.EXPECT().
					ValidateGitHubInstallationId(gomock.Any(), gomock.Any(), installationID).
					Return(nil)
				store.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), validState).
					Return(db.GetProjectIDBySessionStateRow{
						ProjectID: uuid.New(),
					}, nil)
				service.EXPECT().
					CreateGitHubAppProvider(gomock.Any(), gomock.Any(), gomock.Any(), installationID, gomock.Any()).
					Return(&db.Provider{}, nil)
			},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, 200, resp.Code)
			},
		}, {
			name:  "Success no provider",
			state: "",
			buildStubs: func(db *mockdb.MockStore, service *mockprovsvc.MockGitHubProviderService, gh *mockgh.MockClientService) {
				service.EXPECT().
					ValidateGitHubInstallationId(gomock.Any(), gomock.Any(), installationID).
					Return(nil)
				userId := int64(31337)
				gh.EXPECT().GetUserIdFromToken(gomock.Any(), gomock.Any()).
					Return(&userId, nil)
				db.EXPECT().BeginTransaction().Return(nil, nil)
				db.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(db)
				db.EXPECT().Commit(gomock.Any()).Return(nil)
				db.EXPECT().Rollback(gomock.Any()).Return(nil)
				service.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), userId, installationID).
					Return(nil, nil)
			},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, 200, resp.Code)
			},
		}, {
			name:  "Invalid installation ID",
			state: validState,
			buildStubs: func(_ *mockdb.MockStore, service *mockprovsvc.MockGitHubProviderService, _ *mockgh.MockClientService) {
				service.EXPECT().
					ValidateGitHubInstallationId(gomock.Any(), gomock.Any(), installationID).
					Return(errors.New("invalid installation ID"))
			},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, 403, resp.Code)
			},
		}, {
			name:  "Wrong remote userid",
			state: validState,
			buildStubs: func(store *mockdb.MockStore, service *mockprovsvc.MockGitHubProviderService, _ *mockgh.MockClientService) {
				service.EXPECT().
					ValidateGitHubInstallationId(gomock.Any(), gomock.Any(), installationID).
					Return(nil)
				store.EXPECT().
					GetProjectIDBySessionState(gomock.Any(), validState).
					Return(db.GetProjectIDBySessionStateRow{
						ProjectID: uuid.New(),
					}, nil)
				service.EXPECT().
					CreateGitHubAppProvider(gomock.Any(), gomock.Any(), gomock.Any(), installationID, gomock.Any()).
					Return(nil, ghService.ErrInvalidTokenIdentity)
			},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, 403, resp.Code)
			},
		}, {
			name:        "Request to install",
			state:       validState,
			setupAction: "request",
			buildStubs:  func(*mockdb.MockStore, *mockprovsvc.MockGitHubProviderService, *mockgh.MockClientService) {},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, 403, resp.Code)
			},
		},
	}

	for _, tt := range testCases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := http.Request{}
			resp := httptest.ResponseRecorder{Body: new(bytes.Buffer)}
			params := map[string]string{"provider": "github-app"}
			setupAction := "install"
			if tc.setupAction != "" {
				setupAction = tc.setupAction
			}

			req.URL = &url.URL{
				RawQuery: url.Values{
					"state":           {tc.state},
					"code":            {code},
					"installation_id": {fmt.Sprintf("%d", installationID)},
					"setup_action":    {setupAction},
				}.Encode(),
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")

			oauthServer := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					err := json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "anAccessToken",
					})
					if err != nil {
						t.Fatalf("Failed to write response: %v", err)
					}
				}))
			defer oauthServer.Close()
			providerAuthFactory := func(_ string, _ bool) (*oauth2.Config, error) {
				return &oauth2.Config{
					Endpoint: oauth2.Endpoint{
						TokenURL: oauthServer.URL,
					},
				}, nil
			}

			providerService := mockprovsvc.NewMockGitHubProviderService(ctrl)
			store := mockdb.NewMockStore(ctrl)
			gh := mockgh.NewMockClientService(ctrl)

			tc.buildStubs(store, providerService, gh)

			s := &Server{
				store:               store,
				ghProviders:         providerService,
				evt:                 evt,
				providerAuthFactory: providerAuthFactory,
				ghClient:            gh,
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{},
				},
			}

			s.HandleGitHubAppCallback()(&resp, &req, params)

			tc.checkResponse(t, resp)
		})
	}
}

func TestVerifyProviderCredential(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	enrollmentNonce := "enrollmentNonce"
	githubAppProviderName := "github-app-my-org"

	testCases := []struct {
		name          string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, resp *pb.VerifyProviderCredentialResponse, err error)
	}{
		{
			name: "Success with access token",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccessTokenByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderAccessToken{
						Provider: "github",
					}, nil)
			},
			checkResponse: func(t *testing.T, resp *pb.VerifyProviderCredentialResponse, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.Equal(t, "github", resp.ProviderName)
				assert.True(t, resp.Created)
			},
		}, {
			name: "Success with installation ID",
			buildStubs: func(store *mockdb.MockStore) {
				providerId := uuid.New()
				store.EXPECT().
					GetAccessTokenByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderAccessToken{}, sql.ErrNoRows)
				store.EXPECT().
					GetInstallationIDByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderGithubAppInstallation{
						ProviderID: uuid.NullUUID{
							Valid: true,
							UUID:  providerId,
						},
					}, nil)
				store.EXPECT().
					GetProviderByID(gomock.Any(), providerId).
					Return(db.Provider{
						Name: githubAppProviderName,
					}, nil)
			},
			checkResponse: func(t *testing.T, resp *pb.VerifyProviderCredentialResponse, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.Equal(t, githubAppProviderName, resp.ProviderName)
				assert.True(t, resp.Created)
			},
		}, {
			name: "No credential",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccessTokenByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderAccessToken{}, sql.ErrNoRows)
				store.EXPECT().
					GetInstallationIDByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderGithubAppInstallation{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, resp *pb.VerifyProviderCredentialResponse, err error) {
				t.Helper()
				assert.NoError(t, err)
				assert.False(t, resp.Created)
			},
		}, {
			name: "Failure due to error",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccessTokenByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderAccessToken{}, sql.ErrNoRows)
				store.EXPECT().
					GetInstallationIDByEnrollmentNonce(gomock.Any(), gomock.Any()).
					Return(db.ProviderGithubAppInstallation{}, errors.New("error"))
			},
			checkResponse: func(t *testing.T, _ *pb.VerifyProviderCredentialResponse, err error) {
				t.Helper()
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range testCases {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engine.WithEntityContext(context.Background(), &engine.EntityContext{
				Project: engine.Project{ID: projectID},
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")

			oauthServer := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					err := json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "anAccessToken",
					})
					if err != nil {
						t.Fatalf("Failed to write response: %v", err)
					}
				}))
			defer oauthServer.Close()
			providerAuthFactory := func(_ string, _ bool) (*oauth2.Config, error) {
				return &oauth2.Config{
					Endpoint: oauth2.Endpoint{
						TokenURL: oauthServer.URL,
					},
				}, nil
			}

			store := mockdb.NewMockStore(ctrl)

			tc.buildStubs(store)

			s := &Server{
				store:               store,
				evt:                 evt,
				providerAuthFactory: providerAuthFactory,
				providerStore:       providers.NewProviderStore(store),
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{},
				},
			}

			resp, err := s.VerifyProviderCredential(ctx, &pb.VerifyProviderCredentialRequest{
				Context:         &pb.Context{},
				EnrollmentNonce: enrollmentNonce,
			})

			tc.checkResponse(t, resp, err)
		})
	}
}

type partialDbParamsMatcher struct {
	value db.CreateSessionStateParams
}

func (p partialDbParamsMatcher) Matches(x interface{}) bool {
	typedX, ok := x.(db.CreateSessionStateParams)
	if !ok {
		return false
	}

	typedX.SessionState = ""
	return cmp.Equal(typedX, p.value,
		cmpopts.IgnoreFields(db.CreateSessionStateParams{}, "ProviderConfig"))
}

func (m partialDbParamsMatcher) String() string {
	return fmt.Sprintf("matches %+v", m.value)
}
