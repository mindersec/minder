// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/json"
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
)

var update = flag.Bool("update", false, "update golden files")

// ResetEntireTree wipes flags and contexts so tests don't "bleed" into each other
func ResetEntireTree(c *cobra.Command) {
	//nolint:staticcheck // SA1012: Cobra requires nil to clear the context so it can resume inheriting from the root
	c.SetContext(nil)
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if slice, ok := f.Value.(pflag.SliceValue); ok {
			_ = slice.Replace([]string{})
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, child := range c.Commands() {
		ResetEntireTree(child)
	}
}

// CmdTestCase is the shared struct for all rule type CLI tests
type CmdTestCase struct {
	Name           string
	Args           []string
	MockSetup      func(t *testing.T, ctrl *gomock.Controller) context.Context
	GoldenFileName string
	ExpectedError  string
}

// RunCmdTests iterates over a slice of test cases and executes them
func RunCmdTests(
	t *testing.T,
	tests []CmdTestCase,
	cmd *cobra.Command,
) {
	t.Helper()
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	cwd, _ := os.Getwd()
	dummyConfig := filepath.Join(cwd, "config.yaml")
	if _, err := os.Stat(dummyConfig); os.IsNotExist(err) {
		_ = os.WriteFile(dummyConfig, []byte(""), 0600)
		defer os.Remove(dummyConfig)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			viper.Reset()

			rootCmd := cmd.Root()
			ResetEntireTree(rootCmd)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			if tc.MockSetup != nil {
				ctx = tc.MockSetup(t, ctrl)
			}

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetContext(ctx)

			rootCmd.SetArgs(tc.Args)

			_ = viper.BindPFlags(cmd.Flags())
			viper.Set("project", zeroUUID)

			_, err := cmd.ExecuteContextC(ctx)

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

	if json.Valid(expected) && json.Valid([]byte(actual)) {
		// if it's valid json compare the objects (ignores spaces/newlines)
		require.JSONEq(t, string(expected), actual, "JSON Output does not match golden file")
	} else {
		// if it's a table, txt, or yaml fallback to exact string matching
		require.Equal(t, string(expected), actual, "Output does not match golden file")
	}
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
