// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasources

import (
	"maps"
	"slices"
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

func TestBuildFromProtobuf(t *testing.T) {
	t.Parallel()
	// mockProv := &mock.Provider{name: "test-provider"}
	tests := []struct {
		name          string
		ds            *minderv1.DataSource
		withProvider  bool
		provider      provinfv1.Provider
		expectedFuncs []string
		errorMsg      string
	}{
		{
			name:     "nil data source",
			ds:       nil,
			provider: nil,
			errorMsg: "data source is nil",
		},
		{
			name: "nil driver",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "test",
				Name:    "test-ds",
				Id:      "12345",
				Driver:  nil,
			},
			errorMsg: "data source driver is nil",
		},
		{
			name: "successful structured data source creation",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "structured",
				Name:    "test-structured-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Structured{
					Structured: &minderv1.StructDataSource{
						Def: map[string]*minderv1.StructDataSource_Def{
							"test": {
								Path: &minderv1.StructDataSource_Def_Path{
									FileName: "test.yaml",
								},
							},
						},
					},
				},
			},
			expectedFuncs: []string{"test"},
		},
		{
			name: "successful structured data source creation with provider (ignored)",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "structured",
				Name:    "test-structured-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Structured{
					Structured: &minderv1.StructDataSource{
						Def: map[string]*minderv1.StructDataSource_Def{
							"test": {
								Path: &minderv1.StructDataSource_Def_Path{
									FileName: "test.yaml",
								},
							},
						},
					},
				},
			},
			withProvider:  true, // Provider should be ignored for structured data sources
			expectedFuncs: []string{"test"},
		},
		{
			name: "successful REST data source creation without provider",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "rest",
				Name:    "test-rest-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: map[string]*minderv1.RestDataSource_Def{
							"get_data": {
								Endpoint: "https://api.example.com/data",
								Method:   "GET",
							},
						},
					},
				},
			},
			expectedFuncs: []string{"get_data"},
		},
		{
			name: "successful REST data source creation with provider",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "rest",
				Name:    "test-rest-ds-auth",
				Id:      "12345",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: map[string]*minderv1.RestDataSource_Def{
							"get_authenticated_data": {
								Endpoint: "https://api.example.com/secure/data",
								Method:   "GET",
								Headers: map[string]string{
									"Content-Type": "application/json",
								},
							},
						},
						ProviderAuth: true, // This field indicates provider auth should be used
					},
				},
			},
			withProvider:  true,
			expectedFuncs: []string{"get_authenticated_data"},
		},
		{
			name: "REST data source with complex configuration",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "rest",
				Name:    "complex-rest-ds",
				Id:      "67890",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: map[string]*minderv1.RestDataSource_Def{
							"post_data": {
								Endpoint: "https://api.example.com/submit",
								Method:   "POST",
								Headers: map[string]string{
									"Content-Type": "application/json",
									"Accept":       "application/json",
								},
								Body: &minderv1.RestDataSource_Def_Bodyobj{
									Bodyobj: func() *structpb.Struct {
										s, _ := structpb.NewStruct(map[string]interface{}{
											"key": "value",
										})
										return s
									}(),
								},
								Parse:          "json",
								ExpectedStatus: []int32{200, 201},
							},
						},
						ProviderAuth: false,
					},
				},
			},
			expectedFuncs: []string{"post_data"},
		},
		{
			name: "invalid structured data source",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "structured",
				Name:    "invalid-structured-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Structured{
					Structured: &minderv1.StructDataSource{
						Def: map[string]*minderv1.StructDataSource_Def{
							"invalid": nil, // Invalid definition
						},
					},
				},
			},
			errorMsg: "data source handler definition is nil",
		},
		{
			name: "invalid REST data source",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "rest",
				Name:    "invalid-rest-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: map[string]*minderv1.RestDataSource_Def{
							"invalid_endpoint": nil, // Invalid: nil definition
						},
					},
				},
			},
			errorMsg: "rest data source handler definition is nil",
		},
		{
			name: "empty REST data source definition",
			ds: &minderv1.DataSource{
				Version: "v1",
				Type:    "rest",
				Name:    "empty-rest-ds",
				Id:      "12345",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: nil, // Empty definition
					},
				},
			},
			errorMsg: "rest data source definition is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mockProv provinfv1.Provider
			if tt.withProvider {
				ctrl := gomock.NewController(t)
				mockProv = mock_v1.NewMockProvider(ctrl)
			}

			result, err := BuildFromProtobuf(tt.ds, mockProv)

			if tt.errorMsg != "" {
				assert.ErrorContains(t, err, tt.errorMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			funcs := result.GetFuncs()
			want := make([]v1datasources.DataSourceFuncKey, len(funcs))
			for i, v := range tt.expectedFuncs {
				want[i] = v1datasources.DataSourceFuncKey(v)
			}
			require.ElementsMatch(t, slices.Collect(maps.Keys(funcs)), want)
		})
	}
}
