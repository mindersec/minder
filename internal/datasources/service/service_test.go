// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/mindersec/minder/pkg/datasources/v1"
)

var (
	projectID              = uuid.New()
	subscriptionID         = uuid.New()
	validRESTDriverFixture = &minderv1.DataSource_Rest{
		Rest: &minderv1.RestDataSource{
			Def: map[string]*minderv1.RestDataSource_Def{
				"test_function": {
					Endpoint: "http://example.com",
					InputSchema: func() *structpb.Struct {
						s, _ := structpb.NewStruct(map[string]any{
							"type": "object",
							"properties": map[string]any{
								"test": map[string]any{
									"type": "string",
								},
							},
						})
						return s
					}(),
				},
			},
		},
	}
)

func TestGetByName(t *testing.T) {
	t.Parallel()

	type args struct {
		name    string
		project uuid.UUID
		opts    *ReadOptions
	}
	tests := []struct {
		name    string
		args    args
		setup   func(mockDB *mockdb.MockStore)
		want    *minderv1.DataSource
		wantErr bool
	}{
		{
			name: "DataSource found",
			args: args{
				name:    "test_name",
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				dsID := uuid.New()
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        dsID,
					Name:      "test_name",
					ProjectID: uuid.New(),
				}, nil)

				is, err := structpb.NewStruct(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"test": "string",
					},
				})
				require.NoError(t, err, "failed to create struct")

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: dsID,
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint:    "http://example.com",
								InputSchema: is,
							}),
						},
					}, nil)
			},
			want: &minderv1.DataSource{
				Name: "test_name",
			},
			wantErr: false,
		},
		{
			name: "DataSource not found",
			args: args{
				name:    "non_existent",
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Database error",
			args: args{
				name:    "test_name",
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "DataSource found with no functions",
			args: args{
				name:    "test_name",
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				dsID := uuid.New()
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        dsID,
					Name:      "test_name",
					ProjectID: uuid.New(),
				}, nil)

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup
			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(mockStore)

			got, err := svc.GetByName(context.Background(), tt.args.name, tt.args.project, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.NotNilf(t, got.Driver, "driver is nil")
		})
	}
}

func TestGetByID(t *testing.T) {
	t.Parallel()

	type args struct {
		id      uuid.UUID
		project uuid.UUID
		opts    *ReadOptions
	}
	tests := []struct {
		name    string
		args    args
		setup   func(id uuid.UUID, mockDB *mockdb.MockStore)
		want    *minderv1.DataSource
		wantErr bool
	}{
		{
			name: "DataSource found",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(id uuid.UUID, mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSource(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        id,
					Name:      "test_name",
					ProjectID: uuid.New(),
				}, nil)

				is, err := structpb.NewStruct(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"test": "string",
					},
				})
				require.NoError(t, err, "failed to create struct")

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: id,
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint:    "http://example.com",
								InputSchema: is,
							}),
						},
					}, nil)
			},
			want: &minderv1.DataSource{
				Name: "test_name",
			},
			wantErr: false,
		},
		{
			name: "DataSource not found",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(_ uuid.UUID, mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, sql.ErrNoRows)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Database error",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(_ uuid.UUID, mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup
			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(tt.args.id, mockStore)

			got, err := svc.GetByID(context.Background(), tt.args.id, tt.args.project, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.NotNilf(t, got.Driver, "driver is nil")
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	type args struct {
		project uuid.UUID
		opts    *ReadOptions
	}
	tests := []struct {
		name    string
		args    args
		setup   func(mockDB *mockdb.MockStore)
		want    []*minderv1.DataSource
		wantErr bool
	}{
		{
			name: "List data sources",
			args: args{
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				dsID := uuid.New()
				mockDB.EXPECT().ListDataSources(gomock.Any(), gomock.Any()).Return([]db.DataSource{
					{
						ID:        dsID,
						Name:      "test_name",
						ProjectID: uuid.New(),
					},
				}, nil)

				is, err := structpb.NewStruct(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"test": "string",
					},
				})
				require.NoError(t, err, "failed to create struct")

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: dsID,
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint:    "http://example.com",
								InputSchema: is,
							}),
						},
					}, nil)
			},
			want: []*minderv1.DataSource{
				{
					Name: "test_name",
				},
			},
			wantErr: false,
		},
		{
			name: "Database error",
			args: args{
				project: uuid.New(),
				opts:    &ReadOptions{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().ListDataSources(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("database error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup
			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(mockStore)

			got, err := svc.List(context.Background(), tt.args.project, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, len(tt.want))

			for i, want := range tt.want {
				assert.Equal(t, want.Name, got[i].Name)
				assert.NotNilf(t, got[i].Driver, "driver is nil")
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ds             *minderv1.DataSource
		opts           *Options
		subscriptionId uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(mockDB *mockdb.MockStore)
		want    *minderv1.DataSource
		wantErr bool
	}{
		{
			name: "Successfully create REST data source",
			args: args{
				ds: &minderv1.DataSource{
					Name: "test_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)

				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, sql.ErrNoRows)

				mockDB.EXPECT().CreateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.New(),
						Name: "test_ds",
					}, nil)

				mockDB.EXPECT().AddDataSourceFunction(gomock.Any(), gomock.Any()).
					Return(db.DataSourcesFunction{}, nil)
			},
			want: &minderv1.DataSource{
				Name: "test_ds",
			},
			wantErr: false,
		},
		{
			name: "Invalid namespace name",
			args: args{
				ds: &minderv1.DataSource{
					Name: "name-with-no-namespace",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				subscriptionId: subscriptionID,
				opts:           &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Nil data source",
			args: args{
				ds:   nil,
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Invalid project ID",
			args: args{
				ds: &minderv1.DataSource{
					Context: &minderv1.ContextV2{
						ProjectId: "invalid-uuid",
					},
				},
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Data source already exists",
			args: args{
				ds: &minderv1.DataSource{
					Name: "existing_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{ID: uuid.New()}, nil)
			},
			wantErr: true,
		},
		{
			name: "Unsupported driver type",
			args: args{
				ds: &minderv1.DataSource{
					Name: "test_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: nil,
				},
				opts: &Options{},
			},
			setup: func(_ *mockdb.MockStore) {
				// This fails on the early validation side.
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(mockStore)

			got, err := svc.Create(context.Background(), projectID, tt.args.subscriptionId, tt.args.ds, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.NotEmpty(t, got.Id)
		})
	}
}

func TestBuildDataSourceRegistry(t *testing.T) {
	t.Parallel()

	type args struct {
		rt   *minderv1.RuleType
		opts *Options
	}
	tests := []struct {
		name    string
		args    args
		setup   func(rawProjectID string, mockDB *mockdb.MockStore)
		wantErr bool
	}{
		{
			name: "Successful registry build",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)
				dsID := uuid.New()

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        dsID,
					Name:      "test_data_source",
					ProjectID: projectID,
				}, nil)

				is, err := structpb.NewStruct(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"test": map[string]any{
							"type": "string",
						},
					},
				})
				require.NoError(t, err, "failed to create struct")

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: dsID,
							ProjectID:    projectID,
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint:    "http://example.com",
								InputSchema: is,
							}),
						},
					}, nil)
			},
			wantErr: false,
		},
		{
			name: "Project UUID parse error",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr("invalid_uuid"),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup:   func(_ string, _ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "nil data source name reference",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								nil,
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
			},
			wantErr: true,
		},
		{
			name: "Empty data source name reference",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
			},
			wantErr: true,
		},
		{
			name: "Database error when getting parent projects",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)
				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).
					Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "Database error when getting data source by name",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "Database error when getting data source functions",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)
				dsID := uuid.New()

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        dsID,
					Name:      "test_data_source",
					ProjectID: projectID,
				}, nil)

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{}, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			// This should not happen, but we test anyway
			name: "data source without functions",
			args: args{
				rt: &minderv1.RuleType{
					Context: &minderv1.Context{
						Project: ptr.Ptr(uuid.New().String()),
					},
					Def: &minderv1.RuleType_Definition{
						Eval: &minderv1.RuleType_Definition_Eval{
							DataSources: []*minderv1.DataSourceReference{
								{
									Name: "test_data_source",
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(rawProjectID string, mockDB *mockdb.MockStore) {
				projectID := uuid.MustParse(rawProjectID)
				dsID := uuid.New()

				mockDB.EXPECT().GetParentProjects(gomock.Any(), projectID).Return([]uuid.UUID{projectID}, nil)
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(db.DataSource{
					ID:        dsID,
					Name:      "test_data_source",
					ProjectID: projectID,
				}, nil)

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{},
					}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup
			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(tt.args.rt.GetContext().GetProject(), mockStore)

			_, err := svc.BuildDataSourceRegistry(context.Background(), tt.args.rt, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

type fakeTxBuilder struct {
	store         db.Store
	errorOnCommit bool
}

func (f *fakeTxBuilder) Q() db.ExtendQuerier {
	return f.store
}

func (f *fakeTxBuilder) Commit() error {
	if f.errorOnCommit {
		return fmt.Errorf("error on commit")
	}
	return nil
}

func (*fakeTxBuilder) Rollback() error {
	return nil
}

func restDriverToJson(t *testing.T, rs *minderv1.RestDataSource_Def) []byte {
	t.Helper()

	out, err := protojson.Marshal(rs)
	require.NoError(t, err)

	return out
}

func TestDelete(t *testing.T) {
	t.Parallel()

	type args struct {
		id      uuid.UUID
		project uuid.UUID
		opts    *Options
	}

	tests := []struct {
		name    string
		args    args
		setup   func(args args, mockDB *mockdb.MockStore)
		wantErr bool
	}{
		{
			name: "Successful deletion",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				// Mock ListRuleTypesReferencesByDataSource to return empty list
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return([]db.RuleTypeDataSource{}, nil)

				mockDB.EXPECT().
					GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             args.id,
						SubscriptionID: uuid.NullUUID{Valid: false},
					}, nil)

				// Mock DeleteDataSource to succeed
				mockDB.EXPECT().
					DeleteDataSource(gomock.Any(), gomock.Eq(db.DeleteDataSourceParams{
						ID:        args.id,
						ProjectID: args.project,
					})).
					Return(db.DataSource{}, nil)
			},
			wantErr: false,
		},
		{
			name: "Data source not found",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				// Mock ListRuleTypesReferencesByDataSource to return empty list
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return([]db.RuleTypeDataSource{}, nil)

				mockDB.EXPECT().
					GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             args.id,
						SubscriptionID: uuid.NullUUID{Valid: false},
					}, nil)

				// Mock DeleteDataSource to return sql.ErrNoRows
				mockDB.EXPECT().
					DeleteDataSource(gomock.Any(), gomock.Eq(db.DeleteDataSourceParams{
						ID:        args.id,
						ProjectID: args.project,
					})).
					Return(db.DataSource{}, sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "Data source is in use",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				// Mock ListRuleTypesReferencesByDataSource to return non-empty list
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return([]db.RuleTypeDataSource{
						{RuleTypeID: uuid.New()},
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "Data source is part of a bundle",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return([]db.RuleTypeDataSource{}, nil)
				mockDB.EXPECT().
					GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             args.id,
						SubscriptionID: uuid.NullUUID{Valid: true, UUID: subscriptionID},
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "Database error when listing references",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				// Mock ListRuleTypesReferencesByDataSource to return an error
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
		{
			name: "Database error when deleting data source",
			args: args{
				id:      uuid.New(),
				project: uuid.New(),
				opts:    &Options{},
			},
			setup: func(args args, mockDB *mockdb.MockStore) {
				// Mock ListRuleTypesReferencesByDataSource to return empty list
				mockDB.EXPECT().
					ListRuleTypesReferencesByDataSource(gomock.Any(), args.id).
					Return([]db.RuleTypeDataSource{}, nil)
				mockDB.EXPECT().
					GetDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             args.id,
						SubscriptionID: uuid.NullUUID{Valid: false},
					}, nil)

				// Mock DeleteDataSource to return an error
				mockDB.EXPECT().
					DeleteDataSource(gomock.Any(), gomock.Eq(db.DeleteDataSourceParams{
						ID:        args.id,
						ProjectID: args.project,
					})).
					Return(db.DataSource{}, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Setup
			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}

			tt.setup(tt.args, mockStore)

			err := svc.Delete(context.Background(), tt.args.id, tt.args.project, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type args struct {
		ds             *minderv1.DataSource
		opts           *Options
		subscriptionId uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		setup   func(mockDB *mockdb.MockStore)
		want    *minderv1.DataSource
		wantErr bool
	}{
		{
			name: "Successfully update REST data source",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: &minderv1.DataSource_Rest{
						Rest: &minderv1.RestDataSource{
							Def: map[string]*minderv1.RestDataSource_Def{
								"test_function": {
									Endpoint: "http://example.com/updated",
									InputSchema: func() *structpb.Struct {
										s, _ := structpb.NewStruct(map[string]any{
											"type": "object",
											"properties": map[string]any{
												"test": map[string]any{
													"type": "string",
												},
												"foo": map[string]any{
													"type": "string",
												},
											},
										})
										return s
									}(),
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.MustParse(uuid.New().String()),
						Name: "test_ds",
					}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: uuid.New(),
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint: "http://example.com",
								InputSchema: func() *structpb.Struct {
									s, _ := structpb.NewStruct(map[string]any{
										"type": "object",
										"properties": map[string]any{
											"test": map[string]any{
												"type": "string",
											},
										},
									})
									return s
								}(),
							}),
						},
					}, nil)

				mockDB.EXPECT().UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.New(),
						Name: "updated_ds",
					}, nil)

				mockDB.EXPECT().DeleteDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return(nil, nil)

				mockDB.EXPECT().AddDataSourceFunction(gomock.Any(), gomock.Any()).
					Return(db.DataSourcesFunction{}, nil)
			},
			want: &minderv1.DataSource{
				Name: "updated_ds",
			},
			wantErr: false,
		},
		{
			name: "Successfully update REST data source with matching subscription ID",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: &minderv1.DataSource_Rest{
						Rest: &minderv1.RestDataSource{
							Def: map[string]*minderv1.RestDataSource_Def{
								"test_function": {
									Endpoint: "http://example.com/updated",
									InputSchema: func() *structpb.Struct {
										s, _ := structpb.NewStruct(map[string]any{})
										return s
									}(),
								},
							},
						},
					},
				},
				subscriptionId: subscriptionID,
				opts:           &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             uuid.MustParse(uuid.New().String()),
						Name:           "test_ds",
						SubscriptionID: uuid.NullUUID{Valid: true, UUID: subscriptionID},
					}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: uuid.New(),
							Name:         "test_function",
							Type:         v1.DataSourceDriverRest,
							Definition:   restDriverToJson(t, &minderv1.RestDataSource_Def{}),
						},
					}, nil)

				mockDB.EXPECT().UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.New(),
						Name: "updated_ds",
					}, nil)

				mockDB.EXPECT().DeleteDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return(nil, nil)

				mockDB.EXPECT().AddDataSourceFunction(gomock.Any(), gomock.Any()).
					Return(db.DataSourcesFunction{}, nil)
			},
			want: &minderv1.DataSource{
				Name: "updated_ds",
			},
			wantErr: false,
		},
		{
			name: "Non-matching subscription ID",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: &minderv1.DataSource_Rest{
						Rest: &minderv1.RestDataSource{
							Def: map[string]*minderv1.RestDataSource_Def{
								"test_function": {
									Endpoint: "http://example.com/updated",
									InputSchema: func() *structpb.Struct {
										s, _ := structpb.NewStruct(map[string]any{})
										return s
									}(),
								},
							},
						},
					},
				},
				subscriptionId: uuid.New(),
				opts:           &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             uuid.MustParse(uuid.New().String()),
						Name:           "test_ds",
						SubscriptionID: uuid.NullUUID{Valid: true, UUID: subscriptionID},
					}, nil)
			},
			wantErr: true,
		},
		{
			name: "Nil data source",
			args: args{
				ds:   nil,
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Invalid project ID",
			args: args{
				ds: &minderv1.DataSource{
					Id: uuid.New().String(),
					Context: &minderv1.ContextV2{
						ProjectId: "invalid-uuid",
					},
				},
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Invalid data source ID",
			args: args{
				ds: &minderv1.DataSource{
					Id: "invalid-uuid",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
				},
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Data source not found",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "Database error on update",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{ID: uuid.New()}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: uuid.New(),
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint: "http://example.com",
								InputSchema: func() *structpb.Struct {
									s, _ := structpb.NewStruct(map[string]any{
										"type": "object",
										"properties": map[string]any{
											"test": map[string]any{
												"type": "string",
											},
										},
									})
									return s
								}(),
							}),
						},
					}, nil)

				mockDB.EXPECT().UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
		{
			name: "Database error on delete functions",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{ID: uuid.New()}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: uuid.New(),
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint: "http://example.com",
								InputSchema: func() *structpb.Struct {
									s, _ := structpb.NewStruct(map[string]any{
										"type": "object",
										"properties": map[string]any{
											"test": map[string]any{
												"type": "string",
											},
										},
									})
									return s
								}(),
							}),
						},
					}, nil)

				mockDB.EXPECT().UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, nil)

				mockDB.EXPECT().DeleteDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
		{
			name: "Database error on get functions",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{ID: uuid.New()}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{}, fmt.Errorf("database error"))
			},
			wantErr: true,
		},
		{
			name: "Update REST data source fails on update validation",
			args: args{
				ds: &minderv1.DataSource{
					Id:   uuid.New().String(),
					Name: "updated_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: &minderv1.DataSource_Rest{
						Rest: &minderv1.RestDataSource{
							Def: map[string]*minderv1.RestDataSource_Def{
								"test_function": {
									Endpoint: "http://example.com/updated",
									InputSchema: func() *structpb.Struct {
										s, _ := structpb.NewStruct(map[string]any{
											"type": "object",
											"properties": map[string]any{
												// This is not allowed as we're missing the "test" field
												"foo": map[string]any{
													"type": "string",
												},
											},
											// This is not allowed as we're adding a required field
											"required": []any{"foo"},
										})
										return s
									}(),
								},
							},
						},
					},
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.MustParse(uuid.New().String()),
						Name: "test_ds",
					}, nil)
				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: uuid.New(),
							Name:         "test_function",
							Type:         string(v1.DataSourceDriverRest),
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint: "http://example.com",
								InputSchema: func() *structpb.Struct {
									s, _ := structpb.NewStruct(map[string]any{
										"type": "object",
										"properties": map[string]any{
											"test": map[string]any{
												"type": "string",
											},
										},
									})
									return s
								}(),
							}),
						},
					}, nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}

			tt.setup(mockStore)

			got, err := svc.Update(context.Background(), projectID, tt.args.subscriptionId, tt.args.ds, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Name, got.Name)
		})
	}
}

func TestUpsert(t *testing.T) {
	t.Parallel()

	type args struct {
		subscriptionID uuid.UUID
		ds             *minderv1.DataSource
		opts           *Options
	}
	tests := []struct {
		name    string
		args    args
		setup   func(mockDB *mockdb.MockStore)
		wantErr bool
	}{
		{
			name: "Successfully create data source",
			args: args{
				subscriptionID: subscriptionID,
				ds: &minderv1.DataSource{
					Name: "namespace/test_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				mockDB.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)

				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{}, sql.ErrNoRows)

				mockDB.EXPECT().CreateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.New(),
						Name: "namespace/test_ds",
					}, nil)

				mockDB.EXPECT().AddDataSourceFunction(gomock.Any(), gomock.Any()).
					Return(db.DataSourcesFunction{}, nil)
			},
			wantErr: false,
		},
		{
			name: "Successfully update existing data source",
			args: args{
				subscriptionID: subscriptionID,
				ds: &minderv1.DataSource{
					Name: "namespace/test_ds",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				opts: &Options{},
			},
			setup: func(mockDB *mockdb.MockStore) {
				dsID := uuid.New()
				mockDB.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)

				// The data source already exists
				mockDB.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:             dsID,
						Name:           "namespace/test_ds",
						SubscriptionID: uuid.NullUUID{Valid: true, UUID: subscriptionID},
					}, nil).AnyTimes()

				mockDB.EXPECT().ListDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return([]db.DataSourcesFunction{
						{
							ID:           uuid.New(),
							DataSourceID: dsID,
							Name:         "test_function",
							Type:         v1.DataSourceDriverRest,
							Definition: restDriverToJson(t, &minderv1.RestDataSource_Def{
								Endpoint: "http://example.com/updated",
								InputSchema: func() *structpb.Struct {
									s, _ := structpb.NewStruct(map[string]any{})
									return s
								}(),
							}),
						},
					}, nil)

				mockDB.EXPECT().UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(db.DataSource{
						ID:   uuid.New(),
						Name: "test_ds",
					}, nil)

				mockDB.EXPECT().DeleteDataSourceFunctions(gomock.Any(), gomock.Any()).
					Return(nil, nil)

				mockDB.EXPECT().AddDataSourceFunction(gomock.Any(), gomock.Any()).
					Return(db.DataSourcesFunction{}, nil)
			},
			wantErr: false,
		},
		{
			name: "Invalid namespace name",
			args: args{
				ds: &minderv1.DataSource{
					Name: "name-with-no-namespace",
					Context: &minderv1.ContextV2{
						ProjectId: uuid.New().String(),
					},
					Driver: validRESTDriverFixture,
				},
				subscriptionID: subscriptionID,
				opts:           &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Nil data source",
			args: args{
				ds:   nil,
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
		{
			name: "Invalid project ID",
			args: args{
				ds: &minderv1.DataSource{
					Context: &minderv1.ContextV2{
						ProjectId: "invalid-uuid",
					},
				},
				opts: &Options{},
			},
			setup:   func(_ *mockdb.MockStore) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)

			svc := NewDataSourceService(mockStore)
			svc.txBuilder = func(_ *dataSourceService, _ txGetter) (serviceTX, error) {
				return &fakeTxBuilder{
					store: mockStore,
				}, nil
			}
			tt.setup(mockStore)

			err := svc.Upsert(context.Background(), projectID, tt.args.subscriptionID, tt.args.ds, tt.args.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
