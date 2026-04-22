// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

var update = flag.Bool("update", false, "update golden files")

// CmdTestCase is the shared struct for all rule type CLI tests
type CmdTestCase struct {
	Name           string
	Args           []string
	MockSetup      func(t *testing.T, client *mockv1.MockRuleTypeServiceClient)
	GoldenFileName string
	ExpectedError  string
}

// RunCmdTests iterates over a slice of test cases and executes them
func RunCmdTests(
	t *testing.T,
	tests []CmdTestCase,
	cmd *cobra.Command,
	execFunc func(ctx context.Context, c *cobra.Command) error,
) {

	t.Helper()
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			viper.Reset()
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
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
			if tc.MockSetup != nil {
				tc.MockSetup(t, mockClient)
			}

			ctx := WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), mockClient)
			cmd.SetContext(ctx)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Flags().Parse(tc.Args)
			require.NoError(t, err, "flag parsing should not fail")

			_ = viper.BindPFlags(cmd.Flags())
			viper.Set("project", zeroUUID)

			err = execFunc(ctx, cmd)

			if tc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.ExpectedError)
				return
			}

			require.NoError(t, err, "command execution should not fail")
			checkGoldenFile(t, tc.GoldenFileName, buf.String())
		})
	}
}

func checkGoldenFile(t *testing.T, filename string, actual string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", filename+".golden")

	if *update {
		err := os.MkdirAll("testdata", 0750)
		require.NoError(t, err)

		err = os.WriteFile(goldenPath, []byte(actual), 0600)
		require.NoError(t, err)
		t.Logf("Updated golden file: %s", goldenPath)
	}

	// #nosec G304
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "could not read golden file. Run 'go test ./... -update' to generate it")

	assert.Equal(t, string(expected), actual, "Output does not match golden file")
}

// LoadFixture reads a JSON file from the "fixture" directory and unmarshals
func LoadFixture(t *testing.T, filename string, target proto.Message) {
	t.Helper()

	// #nosec G304
	data, err := os.ReadFile(filepath.Join("fixture", filename))
	require.NoError(t, err, "failed to read fixture file. Check if 'fixture/%s' exists", filename)

	err = protojson.Unmarshal(data, target)
	require.NoError(t, err, "failed to unmarshal fixture into protobuf")
}
