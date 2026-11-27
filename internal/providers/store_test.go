// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
)

type providerStoreTestEnv struct {
	store     ProviderStore
	mockStore *mockdb.MockStore
}

func newTestProviderStore(t *testing.T) *providerStoreTestEnv {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockStore := mockdb.NewMockStore(ctrl)
	return &providerStoreTestEnv{
		store:     NewProviderStore(mockStore),
		mockStore: mockStore,
	}
}

func TestProviderStore_Create(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	cfg := json.RawMessage(`{"github":{"endpoint":"https://api.github.com"}}`)
	traits := []db.ProviderType{db.ProviderTypeGithub}
	flows := []db.AuthorizationFlow{db.AuthorizationFlowUserInput}

	tests := []struct {
		name      string
		projectID uuid.UUID
		setup     func(m *mockdb.MockStore)
		wantErr   string
		wantCode  codes.Code
	}{
		{
			name:      "creates provider successfully",
			projectID: projectID,
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().CreateProvider(gomock.Any(), db.CreateProviderParams{
					Name:       "github",
					ProjectID:  projectID,
					Class:      db.ProviderClassGithub,
					Implements: traits,
					Definition: cfg,
					AuthFlows:  flows,
				}).Return(db.Provider{
					ID:         providerID,
					Name:       "github",
					ProjectID:  projectID,
					Class:      db.ProviderClassGithub,
					Implements: traits,
					Definition: cfg,
					AuthFlows:  flows,
				}, nil)
			},
		},
		{
			name:      "rejects nil project ID",
			projectID: uuid.Nil,
			setup:     func(_ *mockdb.MockStore) {},
			wantErr:   "invalid arguments",
			wantCode:  codes.InvalidArgument,
		},
		{
			name:      "propagates database constraint error",
			projectID: projectID,
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().CreateProvider(gomock.Any(), gomock.Any()).
					Return(db.Provider{}, errors.New("pq: duplicate key value violates unique constraint"))
			},
			wantErr: "error creating provider",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.Create(
				context.Background(),
				db.ProviderClassGithub,
				"github",
				tc.projectID,
				cfg,
				traits,
				flows,
			)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				if tc.wantCode != codes.OK {
					require.Equal(t, tc.wantCode, status.Code(err))
				}
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, providerID, got.ID)
			require.Equal(t, "github", got.Name)
		})
	}
}

func TestProviderStore_GetByID(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()

	tests := []struct {
		name    string
		setup   func(m *mockdb.MockStore)
		wantErr string
		wantIs  error
	}{
		{
			name: "returns provider when found",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByID(gomock.Any(), providerID).
					Return(db.Provider{ID: providerID, Name: "github"}, nil)
			},
		},
		{
			name: "wraps sql.ErrNoRows when missing",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByID(gomock.Any(), providerID).
					Return(db.Provider{}, sql.ErrNoRows)
			},
			wantErr: "failed to find provider by ID",
			wantIs:  sql.ErrNoRows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.GetByID(context.Background(), providerID)
			// Assert expected errors and error wrapping
			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				if tc.wantIs != nil {
					require.ErrorIs(t, err, tc.wantIs)
				}
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, providerID, got.ID)
		})
	}
}

func TestProviderStore_GetByIDProject(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name    string
		setup   func(m *mockdb.MockStore)
		wantErr string
		wantIs  error
	}{
		{
			name: "returns provider scoped to project",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByIDAndProject(gomock.Any(), db.GetProviderByIDAndProjectParams{
					ID:        providerID,
					ProjectID: projectID,
				}).Return(db.Provider{ID: providerID, ProjectID: projectID}, nil)
			},
		},
		{
			name: "wraps error when provider is in another project",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByIDAndProject(gomock.Any(), gomock.Any()).
					Return(db.Provider{}, sql.ErrNoRows)
			},
			wantErr: "failed to find provider by ID and project",
			wantIs:  sql.ErrNoRows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.GetByIDProject(context.Background(), providerID, projectID)
			// Assert expected errors and error wrapping
			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				if tc.wantIs != nil {
					require.ErrorIs(t, err, tc.wantIs)
				}
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, providerID, got.ID)
			require.Equal(t, projectID, got.ProjectID)
		})
	}
}

func TestProviderStore_GetByName(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	parentID := uuid.New()
	hierarchy := []uuid.UUID{projectID, parentID}

	tests := []struct {
		name         string
		providerName string
		setup        func(m *mockdb.MockStore)
		wantErr      string
		wantCode     codes.Code
		wantName     string
		assertErr    func(t *testing.T, err error)
	}{
		{
			name:         "returns single matching provider",
			providerName: "github",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), db.FindProvidersParams{
					Projects: hierarchy,
					Name:     sql.NullString{String: "github", Valid: true},
					Trait:    db.NullProviderType{},
				}).Return([]db.Provider{{Name: "github"}}, nil)
			},
			wantName: "github",
		},
		{
			name:         "returns first provider when name is empty",
			providerName: "",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), db.FindProvidersParams{
					Projects: hierarchy,
					Name:     sql.NullString{},
					Trait:    db.NullProviderType{},
				}).Return([]db.Provider{{Name: "first"}, {Name: "second"}}, nil)
			},
			wantName: "first",
		},
		{
			name:         "returns InvalidArgument when named lookup is ambiguous",
			providerName: "github",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), gomock.Any()).
					Return([]db.Provider{{Name: "github"}, {Name: "github"}}, nil)
			},
			wantErr:  "cannot infer provider, there are 2 providers available",
			wantCode: codes.InvalidArgument,
		},
		{
			name: "returns ErrProviderNotFoundBy when no providers match",
			providerName: "missing",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return([]db.Provider{}, nil)
			},
			wantErr: "provider not found with filters: name: missing",
			assertErr: func(t *testing.T, err error) {
				var notFound ErrProviderNotFoundBy
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, "missing", notFound.Name)
			},
		},
		{
			name:         "returns InvalidArgument when GetParentProjects fails",
			providerName: "github",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).
					Return(nil, errors.New("hierarchy lookup failed"))
			},
			wantErr:  "cannot retrieve parent projects",
			wantCode: codes.InvalidArgument,
		},
		{
			name:         "propagates FindProviders database error",
			providerName: "github",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection reset"))
			},
			wantErr: "cannot retrieve providers",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.GetByName(context.Background(), projectID, tc.providerName)
			if tc.assertErr != nil {
				tc.assertErr(t, err)
			}
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				if tc.wantCode != codes.OK {
					require.Equal(t, tc.wantCode, status.Code(err))
				}
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantName, got.Name)
		})
	}
}

func TestProviderStore_GetByNameInSpecificProject(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()

	tests := []struct {
		name    string
		setup   func(m *mockdb.MockStore)
		wantErr string
		wantIs  error
	}{
		{
			name: "returns provider in exact project",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
					Name:     "github",
					Projects: []uuid.UUID{projectID},
				}).Return(db.Provider{Name: "github", ProjectID: projectID}, nil)
			},
		},
		{
			name: "wraps error when not found in project",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByName(gomock.Any(), gomock.Any()).
					Return(db.Provider{}, sql.ErrNoRows)
			},
			wantErr: "cannot retrieve provider",
			wantIs:  sql.ErrNoRows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.GetByNameInSpecificProject(context.Background(), projectID, "github")
			// Assert expected errors and error wrapping
			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				if tc.wantIs != nil {
					require.ErrorIs(t, err, tc.wantIs)
				}
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "github", got.Name)
		})
	}
}

func TestProviderStore_GetByTraitInHierarchy(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	hierarchy := []uuid.UUID{projectID}

	tests := []struct {
		name         string
		providerName string
		trait        db.ProviderType
		setup        func(m *mockdb.MockStore)
		wantCount    int
		wantErr      string
		assertErr    func(t *testing.T, err error)
	}{
		{
			name:         "returns matching providers by trait and name",
			providerName: "github",
			trait:        db.ProviderTypeGithub,
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), db.FindProvidersParams{
					Projects: hierarchy,
					Name:     sql.NullString{String: "github", Valid: true},
					Trait:    db.NullProviderType{ProviderType: db.ProviderTypeGithub, Valid: true},
				}).Return([]db.Provider{{Name: "github"}}, nil)
			},
			wantCount: 1,
		},
		{
			name: "returns ErrProviderNotFoundBy with trait filter",
			providerName: "",
			trait:        db.ProviderTypeGit,
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetParentProjects(gomock.Any(), projectID).Return(hierarchy, nil)
				m.EXPECT().FindProviders(gomock.Any(), gomock.Any()).Return([]db.Provider{}, nil)
			},
			wantErr: "provider not found with filters: trait: git",
			assertErr: func(t *testing.T, err error) {
				var notFound ErrProviderNotFoundBy
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, string(db.ProviderTypeGit), notFound.Trait)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			got, err := env.store.GetByTraitInHierarchy(context.Background(), projectID, tc.providerName, tc.trait)
			if tc.assertErr != nil {
				tc.assertErr(t, err)
			}
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.Len(t, got, tc.wantCount)
		})
	}
}

func TestProviderStore_Delete(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name    string
		setup   func(m *mockdb.MockStore)
		wantErr string
	}{
		{
			name: "deletes provider",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().DeleteProvider(gomock.Any(), db.DeleteProviderParams{
					ID:        providerID,
					ProjectID: projectID,
				}).Return(nil)
			},
		},
		{
			name: "propagates delete error",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().DeleteProvider(gomock.Any(), gomock.Any()).
					Return(errors.New("foreign key violation"))
			},
			wantErr: "foreign key violation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			err := env.store.Delete(context.Background(), providerID, projectID)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProviderStore_Update(t *testing.T) {
	t.Parallel()

	providerID := uuid.New()
	projectID := uuid.New()
	newCfg := json.RawMessage(`{"updated":true}`)
	traits := []db.ProviderType{db.ProviderTypeGithub}
	flows := []db.AuthorizationFlow{db.AuthorizationFlowOauth2AuthorizationCodeFlow}

	tests := []struct {
		name    string
		setup   func(m *mockdb.MockStore)
		wantErr string
		wantIs  error
	}{
		{
			name: "updates config while preserving traits and auth flows",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByID(gomock.Any(), providerID).
					Return(db.Provider{
						ID:         providerID,
						Implements: traits,
						AuthFlows:  flows,
					}, nil)
				m.EXPECT().UpdateProvider(gomock.Any(), db.UpdateProviderParams{
					Implements: traits,
					Definition: newCfg,
					AuthFlows:  flows,
					ID:         providerID,
					ProjectID:  projectID,
				}).Return(nil)
			},
		},
		{
			name: "returns error when provider does not exist",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByID(gomock.Any(), providerID).
					Return(db.Provider{}, sql.ErrNoRows)
			},
			wantErr: "failed to find provider by ID",
			wantIs:  sql.ErrNoRows,
		},
		{
			name: "propagates update error",
			setup: func(m *mockdb.MockStore) {
				m.EXPECT().GetProviderByID(gomock.Any(), providerID).
					Return(db.Provider{ID: providerID}, nil)
				m.EXPECT().UpdateProvider(gomock.Any(), gomock.Any()).
					Return(errors.New("update failed"))
			},
			wantErr: "update failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := newTestProviderStore(t)
			tc.setup(env.mockStore)

			err := env.store.Update(context.Background(), providerID, projectID, newCfg)
			if tc.wantIs != nil {
				require.ErrorIs(t, err, tc.wantIs)
			}
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestErrProviderNotFoundBy_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  ErrProviderNotFoundBy
		want string
	}{
		{
			name: "name only",
			err:  ErrProviderNotFoundBy{Name: "github"},
			want: "provider not found with filters: name: github",
		},
		{
			name: "trait only",
			err:  ErrProviderNotFoundBy{Trait: "git"},
			want: "provider not found with filters: trait: git",
		},
		{
			name: "both name and trait",
			err:  ErrProviderNotFoundBy{Name: "github", Trait: "git"},
			want: "provider not found with filters: name: github, trait: git",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, tc.err.Error())
		})
	}
}
