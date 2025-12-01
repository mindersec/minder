// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
)

type testWriter struct {
	output string
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.output += string(p)
	return len(p), nil
}

func setupConfigFile(configFile string) {
	config := []byte(`logging: "info"`)
	err := os.WriteFile(configFile, config, 0o600)
	if err != nil {
		panic(err)
	}
}

func removeConfigFile(filename string) {
	err := os.Remove(filename)
	if err != nil {
		panic(err)
	}
}

func TestCobraMain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		expectedFile string
	}{
		{
			name:         "pass config flag",
			args:         []string{"--config", "/tmp/config.yaml", "auth"},
			expectedFile: "/tmp/config.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			setupConfigFile(test.expectedFile)
			defer removeConfigFile(test.expectedFile)

			tw := &testWriter{}
			app.RootCmd.SetOut(tw) // stub to capture eventual output
			app.RootCmd.SetArgs(test.args)
			app.Execute()

			actualConfigFile := viper.ConfigFileUsed()
			if actualConfigFile != test.expectedFile {
				t.Errorf("Expected config file %s, got %s", actualConfigFile, test.expectedFile)
			}
		})
	}
}
