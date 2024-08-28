//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	df "github.com/stacklok/minder/database/mock/fixtures"
	db "github.com/stacklok/minder/internal/db"
	sf "github.com/stacklok/minder/internal/ruletypes/mock/fixtures"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCreateRuleType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		mockStoreFunc       df.MockStoreBuilder
		ruleTypeServiceFunc sf.RuleTypeSvcMockBuilder
		request             *minderv1.CreateRuleTypeRequest
		error               bool
	}{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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

			srv, _ := newDefaultServer(t, mockStore, nil, nil)
			srv.ruleTypes = mockSvc

			resp, err := srv.CreateRuleType(context.Background(), tt.request)
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

	tests := []struct {
		name                string
		mockStoreFunc       df.MockStoreBuilder
		ruleTypeServiceFunc sf.RuleTypeSvcMockBuilder
		request             *minderv1.UpdateRuleTypeRequest
		error               bool
	}{
		{
			name: "happy path",
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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
				WithSuccessfulGetProjectByID(uuid.Nil),
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

			srv, _ := newDefaultServer(t, mockStore, nil, nil)
			srv.ruleTypes = mockSvc

			resp, err := srv.UpdateRuleType(context.Background(), tt.request)
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

func WithSuccessfulGetProjectByID(projectID uuid.UUID) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetProjectByID(gomock.Any(), gomock.Any()).
			Return(db.Project{ID: projectID}, nil)
	}
}
