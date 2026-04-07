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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestDeleteCommand(t *testing.T) {
	const (
		zeroUUID = "00000000-0000-0000-0000-000000000000"
		ruleID1  = "00000000-0000-0000-0000-000000000001"
		ruleID2  = "00000000-0000-0000-0000-000000000002"
	)

	tests := []struct {
		name           string
		args           []string
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name: "delete single rule type by id",
			args: []string{"--id", ruleID1},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// command calls GetRuleTypeById to verify it exists and get the name
				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[0], //secret_push_protection
					}, nil)

				// command then calls DeleteRuleType
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil)
			},
			goldenFileName: "delete_single.txt",
		},
		{
			name: "delete all rule types",
			args: []string{"--all", "--yes"},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				// command calls ListRuleTypes to find everything to delete
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				// command loops through and deletes each one
				// since fixture has 3 rules, we expect 3 calls
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil).
					Times(len(mockResp.RuleTypes))
			},
			goldenFileName: "delete_all.txt",
		},
		{
			name: "partial failure - profile reference",
			args: []string{"--id", ruleID2},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[1],
					}, nil)

				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.FailedPrecondition, "cannot delete: used by profiles my-security-profile"))
			},
			goldenFileName: "delete_partial_failure.txt",
		},
		{
			name:          "missing required flags",
			args:          []string{},
			mockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "at least one of the flags in the group [id name all] is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			deleteCmd.Flags().VisitAll(func(f *pflag.Flag) {
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

			cmd := deleteCmd
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
				execErr = deleteCommand(ctx, cmd, cmd.Flags().Args(), nil)
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
