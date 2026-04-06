// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

var update = flag.Bool("update", false, "update .golden files")

func TestListCommand(t *testing.T) {

	tests := []struct {
		name           string
		outputFormat   string
		mockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
		expectedError  string
		goldenFileName string
	}{
		{
			name:         "table output with data",
			outputFormat: app.Table,
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResponse := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
			},
			goldenFileName: "list_populated.table",
		},
		{
			name:         "table output empty",
			outputFormat: app.Table,
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRuleTypesResponse{
						RuleTypes: []*minderv1.RuleType{},
					}, nil)
			},
			goldenFileName: "list_empty.table",
		},
		{
			name:         "json output",
			outputFormat: app.JSON,
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResponse := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
			},
			goldenFileName: "list_populated.json",
		},
		{
			name:         "yaml output",
			outputFormat: app.YAML,
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				mockResponse := &minderv1.ListRuleTypesResponse{}
				loadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
			},
			goldenFileName: "list_populated.yaml",
		},
		{
			name:         "grpc error handling",
			outputFormat: app.Table,
			mockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
			},
			expectedError: "request timed out",
		},
		{
			name:          "invalid output format",
			outputFormat:  "csv",
			mockSetup:     func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {},
			expectedError: "invalid argument",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// setup Mock Client
			mockClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
			tt.mockSetup(t, mockClient)

			originalClientCreator := getRuleTypeClient
			t.Cleanup(func() { getRuleTypeClient = originalClientCreator })

			getRuleTypeClient = func(_ grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
				return mockClient
			}

			// setup Command and Viper state
			viper.Reset()
			viper.Set("project", "00000000-0000-0000-0000-000000000000")
			viper.Set("output", tt.outputFormat)

			buf := new(bytes.Buffer)
			listCmd.SetOut(buf)
			listCmd.SetErr(buf)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// execute the command directly
			err := listCommand(context.Background(), listCmd, []string{}, nil)

			// restore os.Stdout after execution so test logs work
			w.Close()
			os.Stdout = oldStdout

			// read whatever the table library printed
			var capturedStdout bytes.Buffer
			_, _ = io.Copy(&capturedStdout, r)
			r.Close()
			finalOutput := buf.String() + capturedStdout.String()

			// assertions
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)

			// golden File Check
			checkGoldenFile(t, tt.goldenFileName, finalOutput)
		})
	}
}

func checkGoldenFile(t *testing.T, filename string, actual string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", filename+".golden")

	if *update {
		err := os.MkdirAll("testdata", 0755)
		require.NoError(t, err)

		err = os.WriteFile(goldenPath, []byte(actual), 0644)
		require.NoError(t, err)
		t.Logf("Updated golden file: %s", goldenPath)
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "could not read golden file. Run 'go test ./... -update' to generate it")

	assert.Equal(t, string(expected), actual, "Output does not match golden file")
}

func loadFixture(t *testing.T, filename string, target protoreflect.ProtoMessage) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("fixture", filename))
	require.NoError(t, err, "failed to read fixture file. Check if 'fixture/%s' exists", filename)

	err = protojson.Unmarshal(data, target)
	require.NoError(t, err, "failed to unmarshal fixture into protobuf")
}
