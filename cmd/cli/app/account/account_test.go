//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package account

import (
	"os"
	"testing"

	"github.com/stacklok/mediator/cmd/cli/app"
)

type testWriter struct {
	output string
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.output += string(p)
	return len(p), nil
}

func setupConfigFile() string {
	configFile := "config.yaml"
	config := []byte(`logging: "info"`)
	err := os.WriteFile(configFile, config, 0o644)
	if err != nil {
		panic(err)
	}
	return configFile
}

func removeConfigFile(filename string) {
	os.Remove(filename)
}

func TestCobraMain(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "account command",
			args:           []string{"account", "--help"},
			expectedOutput: "account called\n",
		},
		{
			name:           "account login command",
			args:           []string{"account", "login", "--help"},
			expectedOutput: "account login called\n",
		},
		{
			name:           "account list command",
			args:           []string{"account", "list", "--help"},
			expectedOutput: "account list called\n",
		},
		{
			name:           "account create command",
			args:           []string{"account", "create", "--help"},
			expectedOutput: "account create called\n",
		},
		{
			name:           "account revoke command",
			args:           []string{"account", "revoke", "--help"},
			expectedOutput: "account revoke called\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFile := setupConfigFile()
			defer removeConfigFile(configFile)

			os.Setenv("CONFIG_FILE", configFile)

			tw := &testWriter{}
			app.RootCmd.SetOut(tw) // stub to capture eventual output
			app.RootCmd.SetArgs(test.args)

			if err := app.RootCmd.Execute(); err != nil {
				t.Errorf("Error executing command: %v", err)
			}

			// stub to capture eventual output and compare
			// or specfic tests according to the output of the command
			// if got := strings.TrimSpace(tw.output); got != test.expectedOutput {
			// 	t.Errorf("Expected output: %v, got: %v", test.expectedOutput, got)
			// }
		})
	}
}
