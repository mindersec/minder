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

package main

import (
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/stacklok/minder/cmd/cli/app"
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
		test := test
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
