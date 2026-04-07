// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"flag"
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

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

var update = flag.Bool("update", false, "update .golden files")

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestListCommand(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	tests := []struct {
		name           string
		args           []string // Meshery style: raw CLI argument
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		expectedError  string
		goldenFileName string
	}{
		{
			name: "table output with data",
			args: []string{"-o", app.Table},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResponse := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
			},
			goldenFileName: "list_populated.table",
		},
		{
			name: "table output empty",
			args: []string{"-o", app.Table},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRuleTypesResponse{
						RuleTypes: []*minderv1.RuleType{},
					}, nil)
			},
			goldenFileName: "list_empty.table",
		},
		{
			name: "yaml output",
			args: []string{"-o", app.YAML},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResponse := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
			},
			goldenFileName: "list_populated.yaml",
		},
		{
			name: "grpc error handling",
			args: []string{"-o", app.Table},
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
			},
			expectedError: "request timed out",
		},
		{
			name:          "invalid output format",
			args:          []string{"-o", "csv"},
			mockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "invalid argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			listCmd.Flags().VisitAll(func(f *pflag.Flag) {
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

			cmd := listCmd
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

			execErr := listCommand(ctx, cmd, cmd.Flags().Args(), nil)

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
