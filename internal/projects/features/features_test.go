// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package features

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
)

func TestProjectAllowsPrivateRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sqlData []byte
		sqlErr  error
		want    bool
	}{
		{
			name:    "enabled",
			sqlData: []byte(`{}`),
			want:    true,
		},
		{
			name:   "disabled when feature not found",
			sqlErr: sql.ErrNoRows,
			want:   false,
		},
		{
			name:   "disabled on store error",
			sqlErr: sql.ErrConnDone,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			projectID := uuid.New()
			store.EXPECT().
				GetFeatureInProject(gomock.Any(), gomock.Any()).
				Return(tt.sqlData, tt.sqlErr)
			if got := ProjectAllowsPrivateRepos(context.Background(), store, projectID); got != tt.want {
				t.Errorf("ProjectAllowsPrivateRepos() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectAllowsProjectHierarchyOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sqlData []byte
		sqlErr  error
		want    bool
	}{
		{
			name:    "enabled",
			sqlData: []byte(`{}`),
			want:    true,
		},
		{
			name:   "disabled when feature not found",
			sqlErr: sql.ErrNoRows,
			want:   false,
		},
		{
			name:   "disabled on store error",
			sqlErr: sql.ErrConnDone,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			projectID := uuid.New()
			store.EXPECT().
				GetFeatureInProject(gomock.Any(), db.GetFeatureInProjectParams{
					ProjectID: projectID,
					Feature:   projectHierarchyOperationsEnabledFlag,
				}).
				Return(tt.sqlData, tt.sqlErr)
			if got := ProjectAllowsProjectHierarchyOperations(context.Background(), store, projectID); got != tt.want {
				t.Errorf("ProjectAllowsProjectHierarchyOperations() = %v, want %v", got, tt.want)
			}
		})
	}
}
