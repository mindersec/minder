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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestApplyCommand(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	applyFixture := filepath.Join("fixture", "rule_type_apply.yaml")

	tests := []struct {
		name           string
		fileArgs       []string
		posArgs        []string
		mockSetup      func(client *mockv1.MockRuleTypeServiceClient)
		goldenFileName string
		expectedError  string
	}{
		{
			name:     "apply - create new rule type",
			fileArgs: []string{applyFixture},
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
			name:    "apply - update existing rule type (UPSERT)",
			posArgs: []string{applyFixture},
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
			fileArgs:      []string{},
			posArgs:       []string{},
			mockSetup:     func(_ *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "no files specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(mockClient)

			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })
			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			// fresh command and context
			ctx := context.Background()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)
			cmd.Flags().StringArrayP("file", "f", []string{}, "file flag")

			viper.Reset()
			viper.Set("project", zeroUUID)

			for _, f := range tt.fileArgs {
				err := cmd.Flags().Set("file", f)
				require.NoError(t, err)
			}

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := applyCommand(ctx, cmd, tt.posArgs, nil)

			w.Close()
			os.Stdout = oldStdout
			var capturedStdout bytes.Buffer
			_, _ = io.Copy(&capturedStdout, r)
			r.Close()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			checkGoldenFile(t, tt.goldenFileName, buf.String()+capturedStdout.String())
		})
	}
}
