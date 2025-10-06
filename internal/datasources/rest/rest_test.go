// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
	mock_v1 "github.com/mindersec/minder/pkg/providers/v1/mock"
)

func TestNewRestDataSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rest          *minderv1.RestDataSource
		withProvider  bool
		errMsg        string
		expectedFuncs []string
	}{
		{
			name:   "nil rest data source",
			rest:   nil,
			errMsg: "rest data source is nil",
		},
		{
			name: "nil definition",
			rest: &minderv1.RestDataSource{
				Def: nil,
			},
			errMsg: "rest data source definition is nil",
		},
		{
			name: "empty definition map",
			rest: &minderv1.RestDataSource{
				Def: map[string]*minderv1.RestDataSource_Def{},
			},
		},
		{
			name: "single handler without provider auth",
			rest: &minderv1.RestDataSource{
				ProviderAuth: false, // Explicitly set to false
				Def: map[string]*minderv1.RestDataSource_Def{
					"get_data": {
						Endpoint: "https://api.example.com/data",
						Method:   "GET",
						Headers:  map[string]string{"Accept": "application/json"},
						Parse:    "json",
					},
				},
			},
			withProvider:  true,
			expectedFuncs: []string{"get_data"},
		},
		{
			name: "single handler with provider auth enabled",
			rest: &minderv1.RestDataSource{
				ProviderAuth: true, // Provider should be passed through
				Def: map[string]*minderv1.RestDataSource_Def{
					"authenticated_call": {
						Endpoint: "https://api.example.com/secure",
						Method:   "POST",
						Headers:  map[string]string{"Content-Type": "application/json"},
						Parse:    "json",
					},
				},
			},
			withProvider:  true,
			expectedFuncs: []string{"authenticated_call"},
		},
		{
			name: "multiple handlers with mixed provider auth",
			rest: &minderv1.RestDataSource{
				ProviderAuth: true,
				Def: map[string]*minderv1.RestDataSource_Def{
					"list_items": {
						Endpoint: "https://api.example.com/items",
						Method:   "GET",
						Parse:    "json",
					},
					"create_item": {
						Endpoint: "https://api.example.com/items",
						Method:   "POST",
						Headers:  map[string]string{"Content-Type": "application/json"},
						Body:     &minderv1.RestDataSource_Def_BodyFromField{BodyFromField: "item_data"},
						Parse:    "json",
					},
					"get_item": {
						Endpoint: "https://api.example.com/items/{id}",
						Method:   "GET",
						Parse:    "json",
					},
				},
			},
			withProvider:  true,
			expectedFuncs: []string{"list_items", "create_item", "get_item"},
		},
		{
			name: "provider auth disabled with nil provider",
			rest: &minderv1.RestDataSource{
				ProviderAuth: false,
				Def: map[string]*minderv1.RestDataSource_Def{
					"public_endpoint": {
						Endpoint: "https://api.example.com/public",
						Method:   "GET",
						Parse:    "json",
					},
				},
			},
			expectedFuncs: []string{"public_endpoint"},
		},
		{
			name: "handler creation failure due to invalid definition",
			rest: &minderv1.RestDataSource{
				ProviderAuth: false,
				Def: map[string]*minderv1.RestDataSource_Def{
					"valid_handler": {
						Endpoint: "https://api.example.com/valid",
						Method:   "GET",
						Parse:    "json",
					},
					"invalid_handler": nil, // This will cause newHandlerFromDef to fail
				},
			},
			errMsg: "rest data source handler definition is nil",
		},
		{
			name: "invalid schema definition",
			rest: &minderv1.RestDataSource{
				Def: map[string]*minderv1.RestDataSource_Def{
					"invalid_handler": {
						InputSchema: func() *structpb.Struct {
							res, err := structpb.NewStruct(map[string]any{"items": 2})
							assert.NoError(t, err)
							return res
						}(),
					},
				},
			},
			errMsg: `at '/items': got number, want boolean or object`,
		},
		{
			name: "complex handler with body object",
			rest: &minderv1.RestDataSource{
				ProviderAuth: true,
				Def: map[string]*minderv1.RestDataSource_Def{
					"post_data": {
						Endpoint: "https://api.example.com/submit",
						Method:   "POST",
						Headers:  map[string]string{"Content-Type": "application/json"},
						Body: &minderv1.RestDataSource_Def_Bodystr{
							Bodystr: `{"action": "create", "data": {"key": "value"}}`,
						},
						Parse: "json",
					},
				},
			},
			withProvider:  true,
			expectedFuncs: []string{"post_data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var provider provinfv1.Provider
			if tt.withProvider {
				provider = mock_v1.NewMockProvider(ctrl)
			}
			result, err := NewRestDataSource(tt.rest, provider)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				require.NotNil(t, result)

				funcs := result.GetFuncs()
				require.Len(t, funcs, len(tt.expectedFuncs))

				for _, key := range tt.expectedFuncs {
					handler, exists := funcs[v1datasources.DataSourceFuncKey(key)]
					require.True(t, exists, "expected %s function to exist", key)
					require.NotNil(t, handler, "expected %s handler to be non-nil", key)
				}
			}
		})
	}
}
