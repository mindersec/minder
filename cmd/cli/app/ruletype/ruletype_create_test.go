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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestCreateCommand(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	sampleFile := filepath.Join("fixture", "rule_type_sample.yaml")

	tests := []struct {
		name           string
		fileArgs       []string
		mockSetup      func(client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name:     "create rule type from file",
			fileArgs: []string{sampleFile},
			mockSetup: func(client *mockv1.MockRuleTypeServiceClient) {
				mockResp := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{
						RuleType: mockResp.RuleTypes[0],
					}, nil)
			},
			goldenFileName: "create_success.table",
		},
		{
			name:          "missing required file flag",
			fileArgs:      []string{},
			mockSetup:     func(_ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "required flag(s) \"file\" not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(mockClient)

			// mock Injection
			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })
			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			// create a fresh command and context
			ctx := context.Background()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)

			// define flags
			cmd.Flags().StringArrayP("file", "f", []string{}, "file flag")
			_ = cmd.MarkFlagRequired("file")

			viper.Reset()
			viper.Set("project", zeroUUID)

			// set flags
			for _, f := range tt.fileArgs {
				err := cmd.Flags().Set("file", f)
				require.NoError(t, err)
			}

			// check for required flags (manual trigger for test)
			if tt.expectedError != "" {
				err := cmd.ValidateRequiredFlags()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := createCommand(ctx, cmd, []string{}, nil)

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
