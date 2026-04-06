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

	// 1. Create a real temporary YAML file for testing
	tmpDir := t.TempDir()
	tmpFilePath := filepath.Join(tmpDir, "rule.yaml")
	yamlContent := `
version: v1
type: rule-type
name: branch_protection_reviews
context:
  project: 00000000-0000-0000-0000-000000000000
def:
  in_entity: repository
`
	err := os.WriteFile(tmpFilePath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	tests := []struct {
		name           string
		fileArgs       []string
		mockSetup      func(client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name:     "create rule type from file",
			fileArgs: []string{tmpFilePath},
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

			// 2. Inject Mock
			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })
			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			// 3. Create a FRESH command and context
			ctx := context.Background()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)

			// Define flags and mark as required to match production init()
			cmd.Flags().StringArrayP("file", "f", []string{}, "file flag")
			_ = cmd.MarkFlagRequired("file")

			viper.Reset()
			viper.Set("project", zeroUUID)

			// 4. Set flags
			for _, f := range tt.fileArgs {
				err := cmd.Flags().Set("file", f)
				require.NoError(t, err)
			}

			// --- CHECK FOR REQUIRED FLAGS ---
			// This manually triggers the validation Cobra usually does automatically
			if tt.expectedError != "" {
				err := cmd.ValidateRequiredFlags()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return // Success: we caught the missing flag!
			}

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// --- THE INTERCEPTOR ---
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// 5. Execute
			err = createCommand(ctx, cmd, []string{}, nil)

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
