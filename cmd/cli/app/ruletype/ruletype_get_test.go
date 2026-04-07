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

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestGetCommand(t *testing.T) {
	const (
		zeroUUID = "00000000-0000-0000-0000-000000000000"
		ruleID   = "00000000-0000-0000-0000-000000000001"
		ruleName = "secret_push_protection"
	)

	tests := []struct {
		name           string
		args           []string
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name: "get by id - table output",
			args: []string{"--id", ruleID, "-o", app.Table},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "get_by_id.table",
		},
		{
			name: "get by name - yaml output",
			args: []string{"--name", ruleName, "-o", app.YAML},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
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
			args:          []string{"-o", app.Table},
			mockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "at least one of the flags in the group [id name] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			getCmd.Flags().VisitAll(func(f *pflag.Flag) {
				if slice, ok := f.Value.(pflag.SliceValue); ok {
					_ = slice.Replace([]string{})
				} else {
					_ = f.Value.Set(f.DefValue)
				}
				f.Changed = false
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(t, mockClient)

			ctx := cli.WithRPCClient(context.Background(), mockClient)

			cmd := getCmd
			cmd.SetContext(ctx)

			err := cmd.Flags().Parse(tt.args)
			require.NoError(t, err)

			_ = viper.BindPFlags(cmd.Flags())
			viper.Set("project", zeroUUID)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			valErr := cmd.ValidateFlagGroups()
			var execErr error
			if valErr == nil {
				execErr = getCommand(ctx, cmd, cmd.Flags().Args(), nil)
			} else {
				execErr = valErr
			}

			w.Close()
			os.Stdout = oldStdout
			var capturedStdout bytes.Buffer
			_, _ = io.Copy(&capturedStdout, r)
			r.Close()

			if tt.expectedError != "" {
				require.Error(t, execErr)
				assert.Contains(t, execErr.Error(), tt.expectedError)
				return
			}

			require.NoError(t, execErr)

			finalOutput := buf.String() + capturedStdout.String()
			checkGoldenFile(t, tt.goldenFileName, finalOutput)
		})
	}
}
