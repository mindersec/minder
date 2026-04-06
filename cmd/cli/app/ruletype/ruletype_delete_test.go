// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestDeleteCommand(t *testing.T) {
	// Serial execution required due to global getRuleTypeClient variable
	const (
		zeroUUID = "00000000-0000-0000-0000-000000000000"
		// IDs taken directly from your fixture
		ruleID1 = "00000000-0000-0000-0000-000000000001"
		ruleID2 = "00000000-0000-0000-0000-000000000002"
	)

	tests := []struct {
		name           string
		args           map[string]string
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name: "delete single rule type by id",
			args: map[string]string{"id": ruleID1},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// 1. Command calls GetRuleTypeById to verify it exists and get the name
				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[0], // secret_push_protection
					}, nil)

				// 2. Command then calls DeleteRuleType
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil)
			},
			goldenFileName: "delete_single.txt",
		},
		{
			name: "delete all rule types",
			args: map[string]string{"all": "true", "yes": "true"},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// 1. Command calls ListRuleTypes to find everything to delete
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				// 2. Command loops through and deletes each one
				// Since fixture has 3 rules, we expect 3 calls
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil).
					Times(len(mockResp.RuleTypes))
			},
			goldenFileName: "delete_all.txt",
		},
		{
			name: "partial failure - profile reference",
			args: map[string]string{"id": ruleID2}, // secret_scanning
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[1],
					}, nil)

				// Simulate a failure (e.g., rule is in use)
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("referenced by profile"))
			},
			goldenFileName: "delete_partial_failure.txt",
		},
		{
			name:          "missing required flags",
			args:          map[string]string{},
			mockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "at least one of the flags in the group [id all] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(t, mockClient)

			// Mock Injection
			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })
			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			// State Reset
			viper.Reset()
			viper.Set("project", zeroUUID)
			deleteCmd.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			})

			for k, v := range tt.args {
				viper.Set(k, v)
				_ = deleteCmd.Flags().Set(k, v)
			}

			// Validate flags before command logic
			if tt.expectedError != "" {
				err := deleteCmd.ValidateFlagGroups()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			buf := new(bytes.Buffer)
			deleteCmd.SetOut(buf)
			deleteCmd.SetErr(buf)

			// Execute
			err := deleteCommand(context.Background(), deleteCmd, []string{}, nil)

			require.NoError(t, err)
			checkGoldenFile(t, tt.goldenFileName, buf.String())
		})
	}
}
