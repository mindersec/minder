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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/providers/ratecache"
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
	providerName := "github"
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
			name: "Success",
			req: &pb.GetAuthorizationURLRequest{
				Context: &pb.Context{
					Provider: &providerName,
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
					Provider: &providerName,
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

	for i := range testCases {
		tc := testCases[i]
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

			server, _ := newDefaultServer(t, store)

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

			stubClient := StubGitHub{
				UserId: 31337,
			}

			var opts []ServerOption
			if tc.remoteUser.String != "" {
				// TODO: verfifyProviderTokenIdentity
				cancelable, cancel := context.WithCancel(context.Background())
				defer cancel()
				clientCache := ratecache.NewRestClientCache(cancelable)
				clientCache.Set("", "anAccessToken", db.ProviderTypeGithub, &stubClient)
				opts = []ServerOption{
					WithRestClientCache(clientCache),
				}
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			s, _ := newDefaultServer(t, store, opts...)

			var err error
			encryptedUrlString, err := s.cryptoEngine.EncryptString(tc.redirectUrl)
			if err != nil {
				t.Fatalf("Failed to encrypt redirect URL: %v", err)
			}
			encryptedUrl := sql.NullString{
				Valid:  true,
				String: encryptedUrlString,
			}

			tx := sql.Tx{}
			store.EXPECT().BeginTransaction().Return(&tx, nil)
			store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

			store.EXPECT().GetProjectIDBySessionState(gomock.Any(), state).Return(
				db.GetProjectIDBySessionStateRow{
					ProjectID:   projectID,
					RedirectUrl: encryptedUrl,
					RemoteUser:  tc.remoteUser,
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
			s.HandleProviderCallback()(&resp, &req, params)

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

type partialDbParamsMatcher struct {
	value db.CreateSessionStateParams
}

func (p partialDbParamsMatcher) Matches(x interface{}) bool {
	typedX, ok := x.(db.CreateSessionStateParams)
	if !ok {
		return false
	}

	typedX.SessionState = ""

	return typedX == p.value
}

func (m partialDbParamsMatcher) String() string {
	return fmt.Sprintf("matches %+v", m.value)
}
