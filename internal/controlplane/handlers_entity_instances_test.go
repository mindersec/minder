// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/entities/models"
	mockentitysvc "github.com/mindersec/minder/internal/entities/service/mock"
	"github.com/mindersec/minder/internal/entities/service/validators"
	mockproviders "github.com/mindersec/minder/internal/providers/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// toIdentifyingProps converts a map[string]any to map[string]*structpb.Value for tests
func toIdentifyingProps(m map[string]any) map[string]*structpb.Value {
	result := make(map[string]*structpb.Value, len(m))
	for k, v := range m {
		val, _ := structpb.NewValue(v)
		result[k] = val
	}
	return result
}

func TestServer_RegisterEntity(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	entityID := uuid.New()
	providerName := "github"

	tests := []struct {
		name         string
		request      *pb.RegisterEntityRequest
		setupContext func(context.Context) context.Context
		setupMocks   func(*mockproviders.MockProviderStore, *mockentitysvc.MockEntityCreator)
		wantErr      bool
		wantCode     codes.Code
		errContains  string
		validateResp func(*testing.T, *pb.RegisterEntityResponse)
	}{
		{
			name: "successfully registers repository",
			request: &pb.RegisterEntityRequest{
				Context:    &pb.ContextV2{},
				EntityType: pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: toIdentifyingProps(map[string]any{
					"github/repo_owner": "test-owner",
					"github/repo_name":  "test-repo",
				}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project: engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{
						Name: providerName,
					},
				})
			},
			setupMocks: func(provStore *mockproviders.MockProviderStore, creator *mockentitysvc.MockEntityCreator) {
				// Get provider
				provStore.EXPECT().
					GetByName(gomock.Any(), projectID, providerName).
					Return(&db.Provider{
						ID:        providerID,
						Name:      providerName,
						ProjectID: projectID,
					}, nil)

				// Create entity
				creator.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), nil).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         entityID,
							Type:       pb.Entity_ENTITY_REPOSITORIES,
							Name:       "test-owner/test-repo",
							ProjectID:  projectID,
							ProviderID: providerID,
						},
						Properties: properties.NewProperties(map[string]any{
							"github/repo_owner": "test-owner",
							"github/repo_name":  "test-repo",
						}),
					}, nil)
			},
			wantErr: false,
			validateResp: func(t *testing.T, resp *pb.RegisterEntityResponse) {
				t.Helper()
				assert.NotNil(t, resp)
				assert.NotNil(t, resp.GetEntity())
				assert.Equal(t, entityID.String(), resp.GetEntity().GetId())
				assert.Equal(t, pb.Entity_ENTITY_REPOSITORIES, resp.GetEntity().GetType())
				assert.Equal(t, "test-owner/test-repo", resp.GetEntity().GetName())
			},
		},
		{
			name: "fails when entity_type is unspecified",
			request: &pb.RegisterEntityRequest{
				Context:               &pb.ContextV2{},
				EntityType:            pb.Entity_ENTITY_UNSPECIFIED,
				IdentifyingProperties: toIdentifyingProps(map[string]any{"key": "value"}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			// No mocks needed - should fail early
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			errContains: "entity_type must be specified",
		},
		{
			name: "fails when identifying_properties is nil",
			request: &pb.RegisterEntityRequest{
				Context:               &pb.ContextV2{},
				EntityType:            pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: nil,
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			// No mocks needed - should fail early
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			errContains: "identifying_properties is required",
		},
		{
			name: "fails when provider not found",
			request: &pb.RegisterEntityRequest{
				Context:               &pb.ContextV2{},
				EntityType:            pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: toIdentifyingProps(map[string]any{"key": "value"}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			setupMocks: func(provStore *mockproviders.MockProviderStore, _ *mockentitysvc.MockEntityCreator) {
				provStore.EXPECT().
					GetByName(gomock.Any(), projectID, providerName).
					Return(nil, sql.ErrNoRows)
			},
			wantErr:     true,
			wantCode:    codes.NotFound,
			errContains: "provider not found",
		},
		{
			name: "rejects archived repository",
			request: &pb.RegisterEntityRequest{
				Context:    &pb.ContextV2{},
				EntityType: pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: toIdentifyingProps(map[string]any{
					"github/repo_owner": "test-owner",
					"github/repo_name":  "archived-repo",
				}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			setupMocks: func(provStore *mockproviders.MockProviderStore, creator *mockentitysvc.MockEntityCreator) {
				provStore.EXPECT().
					GetByName(gomock.Any(), projectID, providerName).
					Return(&db.Provider{
						ID:        providerID,
						Name:      providerName,
						ProjectID: projectID,
					}, nil)

				// Entity creator returns validation error
				creator.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), nil).
					Return(nil, validators.ErrArchivedRepoForbidden)
			},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			errContains: "archived",
		},
		{
			name: "rejects private repository when forbidden",
			request: &pb.RegisterEntityRequest{
				Context:    &pb.ContextV2{},
				EntityType: pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: toIdentifyingProps(map[string]any{
					"github/repo_owner": "test-owner",
					"github/repo_name":  "private-repo",
				}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			setupMocks: func(provStore *mockproviders.MockProviderStore, creator *mockentitysvc.MockEntityCreator) {
				provStore.EXPECT().
					GetByName(gomock.Any(), projectID, providerName).
					Return(&db.Provider{
						ID:        providerID,
						Name:      providerName,
						ProjectID: projectID,
					}, nil)

				creator.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), nil).
					Return(nil, validators.ErrPrivateRepoForbidden)
			},
			wantErr:     true,
			wantCode:    codes.InvalidArgument,
			errContains: "private",
		},
		{
			name: "handles internal errors appropriately",
			request: &pb.RegisterEntityRequest{
				Context:               &pb.ContextV2{},
				EntityType:            pb.Entity_ENTITY_REPOSITORIES,
				IdentifyingProperties: toIdentifyingProps(map[string]any{"key": "value"}),
			},
			setupContext: func(ctx context.Context) context.Context {
				return engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
					Project:  engcontext.Project{ID: projectID},
					Provider: engcontext.Provider{Name: providerName},
				})
			},
			setupMocks: func(provStore *mockproviders.MockProviderStore, creator *mockentitysvc.MockEntityCreator) {
				provStore.EXPECT().
					GetByName(gomock.Any(), projectID, providerName).
					Return(&db.Provider{
						ID:        providerID,
						Name:      providerName,
						ProjectID: projectID,
					}, nil)

				creator.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), nil).
					Return(nil, errors.New("unexpected internal error"))
			},
			wantErr:     true,
			wantCode:    codes.Internal,
			errContains: "unable to register entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProvStore := mockproviders.NewMockProviderStore(ctrl)
			mockEntityCreator := mockentitysvc.NewMockEntityCreator(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvStore, mockEntityCreator)
			}

			server := &Server{
				providerStore: mockProvStore,
				entityCreator: mockEntityCreator,
			}

			ctx := tt.setupContext(context.Background())

			resp, err := server.RegisterEntity(ctx, tt.request)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantCode != codes.OK {
					st, ok := status.FromError(err)
					require.True(t, ok, "error should be a gRPC status error")
					assert.Equal(t, tt.wantCode, st.Code())
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
		})
	}
}

func TestParseIdentifyingProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *pb.RegisterEntityRequest
		wantErr     bool
		errContains string
		validate    func(*testing.T, *properties.Properties)
	}{
		{
			name: "parses valid properties",
			request: &pb.RegisterEntityRequest{
				IdentifyingProperties: toIdentifyingProps(map[string]any{
					"github/repo_owner": "stacklok",
					"github/repo_name":  "minder",
					"upstream_id":       "12345",
				}),
			},
			wantErr: false,
			validate: func(t *testing.T, props *properties.Properties) {
				t.Helper()
				owner := props.GetProperty("github/repo_owner").GetString()
				assert.Equal(t, "stacklok", owner)

				name := props.GetProperty("github/repo_name").GetString()
				assert.Equal(t, "minder", name)

				id := props.GetProperty("upstream_id").GetString()
				assert.Equal(t, "12345", id)
			},
		},
		{
			name: "fails when properties is nil",
			request: &pb.RegisterEntityRequest{
				IdentifyingProperties: nil,
			},
			wantErr:     true,
			errContains: "identifying_properties is required",
		},
		{
			name: "rejects properties that are too large",
			request: &pb.RegisterEntityRequest{
				IdentifyingProperties: func() map[string]*structpb.Value {
					// Create a value large enough to exceed 32KB limit
					largeValue := strings.Repeat("x", 33*1024)
					return toIdentifyingProps(map[string]any{
						"large_key": largeValue,
					})
				}(),
			},
			wantErr:     true,
			errContains: "identifying_properties too large",
		},
		{
			name: "rejects property key that is too long",
			request: &pb.RegisterEntityRequest{
				IdentifyingProperties: func() map[string]*structpb.Value {
					longKey := strings.Repeat("a", 201)
					return toIdentifyingProps(map[string]any{
						longKey: "value",
					})
				}(),
			},
			wantErr:     true,
			errContains: "property key too long",
		},
		{
			name: "handles empty properties map",
			request: &pb.RegisterEntityRequest{
				IdentifyingProperties: toIdentifyingProps(map[string]any{}),
			},
			wantErr:     true, // Empty map is now an error (changed behavior)
			errContains: "identifying_properties is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			props, err := parseIdentifyingProperties(tt.request)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, props)
				if tt.validate != nil {
					tt.validate(t, props)
				}
			}
		})
	}
}
