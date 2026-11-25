// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package manager contains the GitLabProviderClassManager
package manager

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/crypto/algorithms"
	mockcrypto "github.com/mindersec/minder/internal/crypto/mock"
	"github.com/mindersec/minder/internal/db"
	entmodels "github.com/mindersec/minder/internal/entities/models"
	propssvc "github.com/mindersec/minder/internal/entities/properties/service"
	mockpropssvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/providers/gitlab"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func Test_tokenNeedsRefresh(t *testing.T) {
	t.Parallel()

	baseTime := time.Now()

	tests := []struct {
		name  string
		token oauth2.Token
		want  bool
	}{
		{
			name:  "token is expired",
			token: accessTokenWithExpiration(baseTime.Add(-1 * time.Minute)),
			want:  true,
		},
		{
			name:  "token is not expired and does not need refresh",
			token: accessTokenWithExpiration(baseTime.Add(15 * time.Minute)),
			want:  false,
		},
		{
			name:  "token is not expired but needs refresh",
			token: accessTokenWithExpiration(baseTime.Add(5 * time.Minute)),
			want:  true,
		},
		{
			name: "token is not valid",
			token: oauth2.Token{
				AccessToken: "",
				Expiry:      baseTime.Add(15 * time.Minute),
			},
			want: true,
		},
		{
			name: "stored a PAT",
			token: oauth2.Token{
				AccessToken:  "glpat-notarealtoken.01.1234567",
				RefreshToken: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			needsRefresh := tokenNeedsRefresh(tt.token)
			assert.Equal(t, tt.want, needsRefresh)
		})
	}
}

func accessTokenWithExpiration(exp time.Time) oauth2.Token {
	return oauth2.Token{
		AccessToken: "ozz-likes-beer",
		Expiry:      exp,
	}
}

const (
	deleteTestUpstreamID = "test-upstream-id"
	deleteTestHookID     = "test-hook-id"
)

func helperGenerateEncryptedToken(t *testing.T) pqtype.NullRawMessage {
	t.Helper()

	data := crypto.EncryptedData{
		Algorithm:   algorithms.Aes256Gcm,
		EncodedData: "dnS6VFiMYrfnbeP6eixmBw==",
		KeyVersion:  "",
	}

	serialized, err := data.Serialize()
	require.NoError(t, err)

	return pqtype.NullRawMessage{
		RawMessage: serialized,
		Valid:      true,
	}
}

func newDeleteTestManager(
	t *testing.T,
	store db.Store,
	crypteng crypto.Engine,
	propsSvc propssvc.PropertiesService,
) *providerClassManager {
	t.Helper()

	return &providerClassManager{
		store:                store,
		crypteng:             crypteng,
		propsSvc:             propsSvc,
		glpcfg:               &server.GitLabConfig{},
		gitConfig:            server.GitConfig{},
		webhookURL:           "https://minder.example.com/api/v1/webhook/gitlab",
		parentContext:        t.Context(),
		currentWebhookSecret: "test-secret",
	}
}

func newDeleteTestProvider(
	t *testing.T,
	providerID, projectID uuid.UUID,
	endpoint string,
) *db.Provider {
	t.Helper()

	definition, err := json.Marshal(map[string]any{
		"gitlab": map[string]any{
			"endpoint": endpoint,
		},
	})
	require.NoError(t, err)

	return &db.Provider{
		ID:         providerID,
		ProjectID:  projectID,
		Name:       "gitlab-provider",
		Class:      db.ProviderClassGitlab,
		Version:    provifv1.V1,
		Definition: definition,
	}
}

func expectDeleteCredentials(
	t *testing.T,
	store *mockdb.MockStore,
	crypteng *mockcrypto.MockEngine,
	projectID uuid.UUID,
) {
	t.Helper()

	store.EXPECT().
		GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
			Provider:  "gitlab-provider",
			ProjectID: projectID,
		}).
		Return(db.ProviderAccessToken{
			Provider:             "gitlab-provider",
			ProjectID:            projectID,
			EncryptedAccessToken: helperGenerateEncryptedToken(t),
		}, nil)

	crypteng.EXPECT().
		DecryptOAuthToken(gomock.Any()).
		Return(oauth2.Token{AccessToken: "glpat-test-token"}, nil)
}

func TestDelete(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()

	t.Run("no entities", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		provider := newDeleteTestProvider(t, providerID, projectID, srv.URL)
		expectDeleteCredentials(t, store, crypteng, projectID)
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return(nil, nil)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		require.NoError(t, mgr.Delete(t.Context(), provider))
	})

	t.Run("deregisters repository webhook", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		var deleteCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := fmt.Sprintf("/projects/%s/hooks/%s", deleteTestUpstreamID, deleteTestHookID)
			if r.Method == http.MethodDelete && r.URL.Path == expectedPath {
				deleteCalls.Add(1)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		provider := newDeleteTestProvider(t, providerID, projectID, srv.URL)
		expectDeleteCredentials(t, store, crypteng, projectID)

		entityID := uuid.New()
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return([]db.EntityInstance{{
				ID:         entityID,
				Name:       "repo",
				ProviderID: providerID,
				ProjectID:  projectID,
				EntityType: db.EntitiesRepository,
			}}, nil)

		propsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), entityID, nil).
			Return(entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderv1.Entity_ENTITY_REPOSITORIES,
				},
				properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: deleteTestUpstreamID,
					gitlab.RepoPropertyHookID:     deleteTestHookID,
				}),
			), nil)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		require.NoError(t, mgr.Delete(t.Context(), provider))
		assert.Equal(t, int32(1), deleteCalls.Load())
	})

	t.Run("skips non-repository entities", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		provider := newDeleteTestProvider(t, providerID, projectID, srv.URL)
		expectDeleteCredentials(t, store, crypteng, projectID)

		entityID := uuid.New()
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return([]db.EntityInstance{{
				ID:         entityID,
				Name:       "pull-request",
				ProviderID: providerID,
				ProjectID:  projectID,
				EntityType: db.EntitiesPullRequest,
			}}, nil)

		propsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), entityID, nil).
			Return(entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderv1.Entity_ENTITY_PULL_REQUESTS,
				},
				properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "1",
				}),
			), nil)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		require.NoError(t, mgr.Delete(t.Context(), provider))
	})

	t.Run("continues when properties lookup fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		var deleteCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleteCalls.Add(1)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		provider := newDeleteTestProvider(t, providerID, projectID, srv.URL)
		expectDeleteCredentials(t, store, crypteng, projectID)

		failingEntityID := uuid.New()
		workingEntityID := uuid.New()
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return([]db.EntityInstance{
				{
					ID:         failingEntityID,
					Name:       "failing-repo",
					ProviderID: providerID,
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
				},
				{
					ID:         workingEntityID,
					Name:       "working-repo",
					ProviderID: providerID,
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
				},
			}, nil)

		propsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), failingEntityID, nil).
			Return(nil, errors.New("properties failure"))
		propsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), workingEntityID, nil).
			Return(entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   workingEntityID,
					Type: minderv1.Entity_ENTITY_REPOSITORIES,
				},
				properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: deleteTestUpstreamID,
					gitlab.RepoPropertyHookID:     deleteTestHookID,
				}),
			), nil)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		require.NoError(t, mgr.Delete(t.Context(), provider))
		assert.Equal(t, int32(1), deleteCalls.Load())
	})

	t.Run("continues when webhook deletion fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		var deleteCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleteCalls.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		provider := newDeleteTestProvider(t, providerID, projectID, srv.URL)
		expectDeleteCredentials(t, store, crypteng, projectID)

		entityID := uuid.New()
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return([]db.EntityInstance{{
				ID:         entityID,
				Name:       "repo",
				ProviderID: providerID,
				ProjectID:  projectID,
				EntityType: db.EntitiesRepository,
			}}, nil)

		propsSvc.EXPECT().
			EntityWithPropertiesByID(gomock.Any(), entityID, nil).
			Return(entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderv1.Entity_ENTITY_REPOSITORIES,
				},
				properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: deleteTestUpstreamID,
					gitlab.RepoPropertyHookID:     deleteTestHookID,
				}),
			), nil)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		require.NoError(t, mgr.Delete(t.Context(), provider))
		assert.Equal(t, int32(1), deleteCalls.Load())
	})

	t.Run("returns error when provider cannot be built", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		provider := newDeleteTestProvider(t, providerID, projectID, "https://gitlab.example.com/api/v4/")
		store.EXPECT().
			GetAccessTokenByProjectID(gomock.Any(), db.GetAccessTokenByProjectIDParams{
				Provider:  "gitlab-provider",
				ProjectID: projectID,
			}).
			Return(db.ProviderAccessToken{}, sql.ErrNoRows)

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		err := mgr.Delete(t.Context(), provider)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to build provider for cleanup")
	})

	t.Run("returns error when entities cannot be listed", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		store := mockdb.NewMockStore(ctrl)
		crypteng := mockcrypto.NewMockEngine(ctrl)
		propsSvc := mockpropssvc.NewMockPropertiesService(ctrl)

		provider := newDeleteTestProvider(t, providerID, projectID, "https://gitlab.example.com/api/v4/")
		expectDeleteCredentials(t, store, crypteng, projectID)
		store.EXPECT().
			GetEntitiesByProvider(gomock.Any(), providerID).
			Return(nil, errors.New("database failure"))

		mgr := newDeleteTestManager(t, store, crypteng, propsSvc)
		err := mgr.Delete(t.Context(), provider)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to retrieve list of entities")
	})
}
