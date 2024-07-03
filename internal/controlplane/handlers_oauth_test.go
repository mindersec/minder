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
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth/jwt"
	mockjwt "github.com/stacklok/minder/internal/auth/jwt/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/engcontext"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	mockclients "github.com/stacklok/minder/internal/providers/github/clients/mock"
	ghmanager "github.com/stacklok/minder/internal/providers/github/manager"
	mockgh "github.com/stacklok/minder/internal/providers/github/mock"
	ghService "github.com/stacklok/minder/internal/providers/github/service"
	mockprovsvc "github.com/stacklok/minder/internal/providers/github/service/mock"
	"github.com/stacklok/minder/internal/providers/manager"
	mockmanager "github.com/stacklok/minder/internal/providers/manager/mock"
	"github.com/stacklok/minder/internal/providers/session"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func Test_NewOAuthConfig(t *testing.T) {
	t.Parallel()

	const (
		githubRedirectURI     = "http://github"
		githubClientID        = "ghClientID"
		githubClientSecret    = "ghClientSecret"
		githubAppRedirectURI  = "http://github-app"
		githubAppClientID     = "ghAppClientID"
		githubAppClientSecret = "ghAppClientSecret" //nolint: gosec // This is a test file
	)

	scenarios := []struct {
		name          string
		providerClass db.ProviderClass
		cli           bool
		expected      *oauth2.Config
		err           string
	}{
		{
			name:          "github cli",
			providerClass: db.ProviderClassGithub,
			cli:           true,
			expected: &oauth2.Config{
				ClientID:     githubClientID,
				ClientSecret: githubClientSecret,
				RedirectURL:  githubRedirectURI + "/cli",
				Endpoint:     github.Endpoint,
				Scopes:       []string{"repo", "read:org", "workflow"},
			},
		},
		{
			name:          "github web",
			providerClass: db.ProviderClassGithub,
			cli:           false,
			expected: &oauth2.Config{
				ClientID:     githubClientID,
				ClientSecret: githubClientSecret,
				RedirectURL:  githubRedirectURI + "/web",
				Endpoint:     github.Endpoint,
				Scopes:       []string{"repo", "read:org", "workflow"},
			},
		},
		{
			name:          "github app cli",
			providerClass: db.ProviderClassGithubApp,
			cli:           true,
			expected: &oauth2.Config{
				ClientID:     githubAppClientID,
				ClientSecret: githubAppClientSecret,
				RedirectURL:  githubAppRedirectURI,
				Endpoint:     github.Endpoint,
				Scopes:       []string{},
			},
		},
		{
			name:          "github app web",
			providerClass: db.ProviderClassGithubApp,
			cli:           true,
			expected: &oauth2.Config{
				ClientID:     githubAppClientID,
				ClientSecret: githubAppClientSecret,
				RedirectURL:  githubAppRedirectURI,
				Endpoint:     github.Endpoint,
				Scopes:       []string{},
			},
		},
		{
			name:          "dockerhub fails as expected",
			providerClass: db.ProviderClassDockerhub,
			cli:           true,
			err:           "class manager does not implement OAuthManager",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			githubProviderManager := ghmanager.NewGitHubProviderClassManager(
				nil,
				nil,
				&serverconfig.ProviderConfig{
					GitHub: &serverconfig.GitHubConfig{
						OAuthClientConfig: serverconfig.OAuthClientConfig{
							ClientID:     githubClientID,
							ClientSecret: githubClientSecret,
							RedirectURI:  githubRedirectURI,
						},
					},
					GitHubApp: &serverconfig.GitHubAppConfig{
						OAuthClientConfig: serverconfig.OAuthClientConfig{
							ClientID:     githubAppClientID,
							ClientSecret: githubAppClientSecret,
							RedirectURI:  githubAppRedirectURI,
						},
					},
				},
				nil,
				nil,
				nil,
				nil,
				nil,
			)
			dockerhubProviderManager := dockerhub.NewDockerHubProviderClassManager(nil, nil)

			providerAuthManager, err := manager.NewAuthManager(githubProviderManager, dockerhubProviderManager)
			require.NoError(t, err)

			config, err := providerAuthManager.NewOAuthConfig(scenario.providerClass, scenario.cli)
			if scenario.err == "" {
				require.NoError(t, err)
				require.NotNil(t, config)
				require.Equal(t, scenario.expected.ClientID, config.ClientID)
				require.Equal(t, scenario.expected.ClientSecret, config.ClientSecret)
				require.Equal(t, scenario.expected.RedirectURL, config.RedirectURL)
				require.Equal(t, scenario.expected.Endpoint, config.Endpoint)
				require.Subsetf(t, config.Scopes, scenario.expected.Scopes, "expected: %v, got: %v", scenario.expected.Scopes, config.Scopes)
			} else {
				require.Error(t, err)
				require.ErrorContains(t, err, scenario.err)
			}
		})
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

	basengcontext := withRpcOptions(context.Background(), rpcOptions)

	userJWT := openid.New()
	if err := userJWT.Set("sub", "testuser"); err != nil {
		t.Fatalf("Error setting sub: %v", err)
	}

	// Set the entity context
	basengcontext = engcontext.WithEntityContext(basengcontext, &engcontext.EntityContext{
		Project: engcontext.Project{
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
			ctx := jwt.WithAuthTokenContext(basengcontext, token)

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
					GitHub: &serverconfig.GitHubConfig{
						OAuthClientConfig: serverconfig.OAuthClientConfig{
							ClientID:     "clientID",
							ClientSecret: "clientSecret",
							RedirectURI:  "redirectURI",
						},
					},
					GitHubApp: &serverconfig.GitHubAppConfig{
						OAuthClientConfig: serverconfig.OAuthClientConfig{
							ClientID:     "clientID",
							ClientSecret: "clientSecret",
							RedirectURI:  "redirectURI",
						},
						AppName: "test-app",
					},
				},
			}
			mockJwt := mockjwt.NewMockValidator(ctrl)
			mockAuthManager := mockmanager.NewMockAuthManager(ctrl)
			mockAuthManager.EXPECT().NewOAuthConfig(gomock.Any(), gomock.Any()).Return(&oauth2.Config{}, nil).AnyTimes()

			server := &Server{
				store:               store,
				jwt:                 mockJwt,
				evt:                 evt,
				cfg:                 c,
				mt:                  metrics.NewNoopMetrics(),
				providerAuthManager: mockAuthManager,
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

	withProviderSearch := func(store *mockdb.MockStore) {
		store.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
		store.EXPECT().FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{
				{
					Name:       "github",
					Implements: []db.ProviderType{db.ProviderTypeGithub},
					Version:    provinfv1.V1,
				},
			}, nil)
	}

	withProviderCreate := func(store *mockdb.MockStore) {
		store.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
		store.EXPECT().FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{}, nil)
		store.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).Return(db.Provider{}, nil)
	}

	withProviderNotFound := func(store *mockdb.MockStore) {
		store.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
		store.EXPECT().FindProviders(gomock.Any(), gomock.Any()).
			Return([]db.Provider{}, nil)
	}

	testCases := []struct {
		name                       string
		redirectUrl                string
		remoteUser                 sql.NullString
		code                       int
		storeMockSetup             func(store *mockdb.MockStore)
		projectIDBySessionNumCalls int
		err                        string
		config                     []byte
	}{{
		name:                       "Success",
		redirectUrl:                "http://localhost:8080",
		projectIDBySessionNumCalls: 2,
		storeMockSetup: func(store *mockdb.MockStore) {
			withProviderSearch(store)
		},
		code: 307,
	}, {
		name:                       "Success with remote user",
		redirectUrl:                "http://localhost:8080",
		remoteUser:                 sql.NullString{Valid: true, String: "31337"},
		projectIDBySessionNumCalls: 2,
		storeMockSetup: func(store *mockdb.MockStore) {
			withProviderSearch(store)
		},
		code: 307,
	}, {
		name:                       "Wrong remote userid",
		remoteUser:                 sql.NullString{Valid: true, String: "1234"},
		projectIDBySessionNumCalls: 1,
		storeMockSetup: func(_ *mockdb.MockStore) {
			// this codepath fails before the store is called
		},
		code: 403,
		err:  "The provided login token was associated with a different user.\n",
	}, {
		name:        "No existing provider",
		redirectUrl: "http://localhost:8080",
		// such fallback config is stored when generating the authorization URL, but here we mock
		// the state response, so let's provide the fallback ourselves.
		config:                     []byte(`{}`),
		code:                       307,
		projectIDBySessionNumCalls: 2,
		storeMockSetup: func(store *mockdb.MockStore) {
			withProviderCreate(store)
		},
	}, {
		name:                       "Config does not validate",
		redirectUrl:                "http://localhost:8080",
		remoteUser:                 sql.NullString{Valid: true, String: "31337"},
		projectIDBySessionNumCalls: 2,
		storeMockSetup: func(store *mockdb.MockStore) {
			withProviderNotFound(store)
		},
		config: []byte(`{`),
		code:   http.StatusBadRequest,
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
			tc.storeMockSetup(store)

			s, _ := newDefaultServer(t, store, clientFactory)
			s.cfg.Provider = serverconfig.ProviderConfig{
				GitHub: &serverconfig.GitHubConfig{
					OAuthClientConfig: serverconfig.OAuthClientConfig{
						Endpoint: &serverconfig.OAuthEndpoint{
							TokenURL: oauthServer.URL,
						},
					},
				},
			}

			tokenKeyPath := generateTokenKey(t)
			eng, err := crypto.NewEngineFromConfig(&serverconfig.Config{
				Auth: serverconfig.AuthConfig{
					TokenKey: tokenKeyPath,
				},
			})
			require.NoError(t, err)

			ghClientService := ghService.NewGithubProviderService(
				store,
				eng,
				metrics.NewNoopMetrics(),
				// These nil dependencies do not matter for the current tests
				&serverconfig.ProviderConfig{
					GitHubApp: &serverconfig.GitHubAppConfig{
						WebhookSecret: "test",
					},
				},
				nil,
				clientFactory,
			)

			githubProviderManager := ghmanager.NewGitHubProviderClassManager(
				nil,
				nil,
				&serverconfig.ProviderConfig{
					GitHub: &serverconfig.GitHubConfig{
						OAuthClientConfig: serverconfig.OAuthClientConfig{
							Endpoint: &serverconfig.OAuthEndpoint{
								TokenURL: oauthServer.URL,
							},
						},
					},
				},
				nil,
				nil,
				nil,
				nil,
				ghClientService,
			)
			dockerhubProviderManager := dockerhub.NewDockerHubProviderClassManager(nil, nil)

			authManager, err := manager.NewAuthManager(githubProviderManager, dockerhubProviderManager)
			require.NoError(t, err)
			s.providerAuthManager = authManager

			providerStore := providers.NewProviderStore(store)
			providerManager, err := manager.NewProviderManager(providerStore, githubProviderManager, dockerhubProviderManager)
			require.NoError(t, err)
			s.providerManager = providerManager

			sessionService := session.NewProviderSessionService(providerManager, providerStore, store)
			s.sessionService = sessionService

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

			store.EXPECT().GetProjectIDBySessionState(gomock.Any(), state).Return(
				db.GetProjectIDBySessionStateRow{
					ProjectID:   projectID,
					RedirectUrl: encryptedUrl,
					EncryptedRedirect: pqtype.NullRawMessage{
						RawMessage: serialized,
						Valid:      true,
					},
					RemoteUser:     tc.remoteUser,
					ProviderConfig: tc.config,
				}, nil).Times(tc.projectIDBySessionNumCalls)

			if tc.code < http.StatusBadRequest {
				store.EXPECT().UpsertAccessToken(gomock.Any(), gomock.Any()).Return(
					db.ProviderAccessToken{}, nil)
			}

			t.Logf("Request: %+v", req.URL)

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
			name:  "Invalid config",
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
					Return(nil, providers.NewErrProviderInvalidConfig("invalid config"))
			},
			checkResponse: func(t *testing.T, resp httptest.ResponseRecorder) {
				t.Helper()
				assert.Equal(t, http.StatusBadRequest, resp.Code)
				assert.Contains(t, resp.Body.String(), "The provider configuration is invalid: invalid config")
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

			providerService := mockprovsvc.NewMockGitHubProviderService(ctrl)
			store := mockdb.NewMockStore(ctrl)
			gh := mockgh.NewMockClientService(ctrl)

			mockAuthManager := mockmanager.NewMockAuthManager(ctrl)
			mockAuthManager.EXPECT().NewOAuthConfig(gomock.Any(), gomock.Any()).Return(&oauth2.Config{
				Endpoint: oauth2.Endpoint{
					TokenURL: oauthServer.URL,
				},
			}, nil).AnyTimes()

			tc.buildStubs(store, providerService, gh)

			s := &Server{
				store:               store,
				ghProviders:         providerService,
				evt:                 evt,
				providerAuthManager: mockAuthManager,
				ghClient:            gh,
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{},
					Provider: serverconfig.ProviderConfig{
						GitHub: &serverconfig.GitHubConfig{
							OAuthClientConfig: serverconfig.OAuthClientConfig{
								ClientID: "clientID",
								Endpoint: &serverconfig.OAuthEndpoint{
									TokenURL: oauthServer.URL,
								},
							},
						},
					},
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

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
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
			store := mockdb.NewMockStore(ctrl)

			tc.buildStubs(store)

			s := &Server{
				store:         store,
				evt:           evt,
				providerStore: providers.NewProviderStore(store),
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{},
					Provider: serverconfig.ProviderConfig{
						GitHub: &serverconfig.GitHubConfig{
							OAuthClientConfig: serverconfig.OAuthClientConfig{
								Endpoint: &serverconfig.OAuthEndpoint{
									TokenURL: oauthServer.URL,
								},
							},
						},
					},
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
