// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestApplyCommand(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	applyFixture := filepath.Join("fixture", "rule_type_apply.yaml")

	tests := []struct {
		name           string
		args           []string
		mockSetup      func(client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name: "apply - create new rule type via flag",
			args: []string{"-f", applyFixture},
			mockSetup: func(client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "apply_create.table",
		},
		{
			name: "apply - update existing rule type via positional arg",
			args: []string{applyFixture},
			mockSetup: func(client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))

				client.EXPECT().
					UpdateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.UpdateRuleTypeResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			goldenFileName: "apply_update.table",
		},
		{
			name:          "no files specified",
			args:          []string{},
			mockSetup:     func(_ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "no files specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			// reset slice flags
			applyCmd.Flags().VisitAll(func(f *pflag.Flag) {
				if slice, ok := f.Value.(pflag.SliceValue); ok {
					_ = slice.Replace([]string{}) // empty the array
				} else {
					_ = f.Value.Set(f.DefValue)
				}
				f.Changed = false
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(mockClient)

			ctx := cli.WithRPCClient(context.Background(), mockClient)

			cmd := applyCmd
			cmd.SetContext(ctx)

			// parse raw args
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

			// execute
			err = applyCommand(ctx, cmd, cmd.Flags().Args(), nil)

			w.Close()
			os.Stdout = oldStdout
			var capturedStdout bytes.Buffer
			_, _ = io.Copy(&capturedStdout, r)
			r.Close()

			// assertions
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)

			finalOutput := buf.String() + capturedStdout.String()
			checkGoldenFile(t, tt.goldenFileName, finalOutput)
		})

	}
}
