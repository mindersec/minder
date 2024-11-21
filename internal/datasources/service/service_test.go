// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/datasources"
	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
							Type:         string(datasources.DataSourceDriverRest),
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
							Type:         string(datasources.DataSourceDriverRest),
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
							Type:         string(datasources.DataSourceDriverRest),
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

func (_ *fakeTxBuilder) Rollback() error {
	return nil
}

func restDriverToJson(t *testing.T, rs *minderv1.RestDataSource_Def) []byte {
	t.Helper()

	out, err := protojson.Marshal(rs)
	require.NoError(t, err)

	return out
}
