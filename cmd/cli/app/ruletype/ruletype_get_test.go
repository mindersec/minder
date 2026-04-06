// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestGetCommand(t *testing.T) {
	// Serial execution required due to global getRuleTypeClient and os.Stdout hijacking
	const (
		zeroUUID = "00000000-0000-0000-0000-000000000000"
		ruleID   = "00000000-0000-0000-0000-000000000001"
		ruleName = "secret_push_protection"
	)

	tests := []struct {
		name           string
		args           map[string]string
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name: "get by id - table output",
			args: map[string]string{"id": ruleID, "output": app.Table},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// Wrap in the specific ByIdResponse type
				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "get_by_id.table",
		},
		{
			name: "get by name - json output",
			args: map[string]string{"name": ruleName, "output": app.JSON},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// Wrap in the specific ByNameResponse type
				client.EXPECT().
					GetRuleTypeByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByNameResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "get_by_name.json",
		},
		{
			name: "get by name - yaml output",
			args: map[string]string{"name": ruleName, "output": app.YAML},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByNameResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "get_by_name.yaml",
		},
		{
			name:          "missing both id and name",
			args:          map[string]string{"output": app.Table},
			mockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "at least one of the flags in the group [id name] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(t, mockClient)

			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })

			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			viper.Reset()
			viper.Set("project", zeroUUID)

			getCmd.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
				f.Changed = false
			})

			for k, v := range tt.args {
				viper.Set(k, v)
				err := getCmd.Flags().Set(k, v)
				require.NoError(t, err)
			}

			// Validate flags BEFORE calling the command logic
			// This prevents unexpected gRPC calls when flags are missing
			if tt.expectedError != "" {
				valErr := getCmd.ValidateFlagGroups()
				if valErr != nil {
					assert.Contains(t, valErr.Error(), tt.expectedError)
					return // Success for this test case
				}
			}

			buf := new(bytes.Buffer)
			getCmd.SetOut(buf)
			getCmd.SetErr(buf)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := getCommand(context.Background(), getCmd, []string{}, nil)

			w.Close()
			os.Stdout = oldStdout
			var capturedStdout bytes.Buffer
			_, _ = io.Copy(&capturedStdout, r)
			r.Close()

			require.NoError(t, err)
			finalOutput := buf.String() + capturedStdout.String()
			checkGoldenFile(t, tt.goldenFileName, finalOutput)
		})
	}
}
