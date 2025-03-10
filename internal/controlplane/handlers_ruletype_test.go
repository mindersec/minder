// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	df "github.com/mindersec/minder/database/mock/fixtures"
	dsf "github.com/mindersec/minder/internal/datasources/service/mock/fixtures"
	db "github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/pkg/flags"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	sf "github.com/mindersec/minder/pkg/ruletypes/mock/fixtures"
)

const ruleDefJSON = `
{
	"rule_schema": {},
	"ingest": {
		"type": "git",
        "git": {}
	},
	"eval": {
		"type": "jq",
		"jq": [{
			"ingested": {"def": ".abc"},
			"profile": {"def": ".xyz"}
		}]
	}
}
`

func TestCreateRuleType(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	tests := []struct {
		name                   string
		mockStoreFunc          df.MockStoreBuilder
		ruleTypeServiceFunc    sf.RuleTypeSvcMockBuilder
		dataSourcesServiceFunc dsf.DataSourcesSvcMockBuilder
		features               map[string]any
		request                *minderv1.CreateRuleTypeRequest
		error                  bool
	}{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(
				sf.WithSuccessfulCreateRuleType,
			),
			request: &minderv1.CreateRuleTypeRequest{
				RuleType: &minderv1.RuleType{},
			},
		},
		{
			name: "guidance sanitize error",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.CreateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: "<div>foo</div>",
				},
			},
			error: true,
		},
		{
			name: "guidance not utf-8",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.CreateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: string([]byte{0xff, 0xfe, 0xfd}),
				},
			},
			error: true,
		},
		{
			name: "guidance too long",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.CreateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: strings.Repeat("a", 4*1<<10),
				},
			},
			error: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var mockStore *mockdb.MockStore
			if tt.mockStoreFunc != nil {
				mockStore = tt.mockStoreFunc(ctrl)
			} else {
				mockStore = mockdb.NewMockStore(ctrl)
			}

			var mockSvc sf.RuleTypeSvcMock
			if tt.ruleTypeServiceFunc != nil {
				mockSvc = tt.ruleTypeServiceFunc(ctrl)
			}

			var mockDsSvc dsf.DataSourcesSvcMock
			if tt.dataSourcesServiceFunc != nil {
				mockDsSvc = tt.dataSourcesServiceFunc(ctrl)
			}

			featureClient := &flags.FakeClient{}
			if tt.features != nil {
				featureClient.Data = tt.features
			}

			srv := newDefaultServer(t, mockStore, nil, nil, nil)
			srv.ruleTypes = mockSvc
			srv.dataSourcesService = mockDsSvc
			srv.featureFlags = featureClient

			ctx := context.Background()
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: "testing"},
			})
			resp, err := srv.CreateRuleType(ctx, tt.request)
			if tt.error {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestUpdateRuleType(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	tests := []struct {
		name                   string
		mockStoreFunc          df.MockStoreBuilder
		ruleTypeServiceFunc    sf.RuleTypeSvcMockBuilder
		dataSourcesServiceFunc dsf.DataSourcesSvcMockBuilder
		features               map[string]any
		request                *minderv1.UpdateRuleTypeRequest
		error                  bool
	}{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(
				sf.WithSuccessfulUpdateRuleType,
			),
			request: &minderv1.UpdateRuleTypeRequest{
				RuleType: &minderv1.RuleType{},
			},
		},
		{
			name: "guidance sanitize error",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.UpdateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: "<div>foo</div>",
				},
			},
			error: true,
		},
		{
			name: "guidance not utf-8",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.UpdateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: string([]byte{0xff, 0xfe, 0xfd}),
				},
			},
			error: true,
		},
		{
			name: "guidance too long",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
			),
			ruleTypeServiceFunc: sf.NewRuleTypeServiceMock(),
			request: &minderv1.UpdateRuleTypeRequest{
				RuleType: &minderv1.RuleType{
					Guidance: strings.Repeat("a", 4*1<<10),
				},
			},
			error: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var mockStore *mockdb.MockStore
			if tt.mockStoreFunc != nil {
				mockStore = tt.mockStoreFunc(ctrl)
			} else {
				mockStore = mockdb.NewMockStore(ctrl)
			}

			var mockSvc sf.RuleTypeSvcMock
			if tt.ruleTypeServiceFunc != nil {
				mockSvc = tt.ruleTypeServiceFunc(ctrl)
			}

			var mockDsSvc dsf.DataSourcesSvcMock
			if tt.dataSourcesServiceFunc != nil {
				mockDsSvc = tt.dataSourcesServiceFunc(ctrl)
			}

			featureClient := &flags.FakeClient{}
			if tt.features != nil {
				featureClient.Data = tt.features
			}

			srv := newDefaultServer(t, mockStore, nil, nil, nil)
			srv.ruleTypes = mockSvc
			srv.dataSourcesService = mockDsSvc
			srv.featureFlags = featureClient

			ctx := context.Background()
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: "testing"},
			})
			resp, err := srv.UpdateRuleType(ctx, tt.request)
			if tt.error {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}

func TestListRuleTypes(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	ruleTypeList := []db.RuleType{
		{ID: uuid.New(), Name: "rule1", ProjectID: projectID, Definition: []byte(ruleDefJSON)},
		{ID: uuid.New(), Name: "rule2", ProjectID: projectID, Definition: []byte(ruleDefJSON)},
	}
	tests := []struct {
		name          string
		mockStoreFunc df.MockStoreBuilder
		ruleTypes     []db.RuleType
		error         bool
	}{
		{
			name: "success with rule types",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
				WithSuccessfulListRuleTypesByProject(projectID, ruleTypeList),
			),
			ruleTypes: ruleTypeList,
		},
		{
			name: "success with no rule types",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
				WithSuccessfulListRuleTypesByProject(projectID, []db.RuleType{}),
			),
			ruleTypes: []db.RuleType{},
		},
		{
			name: "error in entity context",
			mockStoreFunc: df.NewMockStore(
				WithFailedGetProjectByID(),
			),
			error: true,
		},
		{
			name: "failed to get rule types",
			mockStoreFunc: df.NewMockStore(
				WithSuccessfulGetProjectByID(projectID),
				WithFailedListRuleTypesByProject(projectID),
			),
			error: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var mockStore *mockdb.MockStore
			if tt.mockStoreFunc != nil {
				mockStore = tt.mockStoreFunc(ctrl)
			} else {
				mockStore = mockdb.NewMockStore(ctrl)
			}

			srv := newDefaultServer(t, mockStore, nil, nil, nil)

			ctx := context.Background()
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: "testing"},
			})

			resp, err := srv.ListRuleTypes(ctx, &minderv1.ListRuleTypesRequest{})
			if tt.error {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.RuleTypes, len(tt.ruleTypes))
		})
	}
}

func WithSuccessfulListRuleTypesByProject(projectID uuid.UUID, ruleTypes []db.RuleType) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			ListRuleTypesByProject(gomock.Any(), projectID).
			Return(ruleTypes, nil)
	}
}

func WithFailedListRuleTypesByProject(projectID uuid.UUID) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			ListRuleTypesByProject(gomock.Any(), projectID).
			Return(nil, errors.New("failed to list rule types"))
	}
}

func WithFailedGetProjectByID() func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetProjectByID(gomock.Any(), gomock.Any()).
			Return(db.Project{}, errors.New("failed to get project by ID"))
	}
}

func WithSuccessfulGetProjectByID(projectID uuid.UUID) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetProjectByID(gomock.Any(), gomock.Any()).
			Return(db.Project{ID: projectID}, nil)
	}
}
