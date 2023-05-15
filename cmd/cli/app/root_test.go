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

package app

import (
	"os"
	"path/filepath"
	"testing"
	"github.com/spf13/viper"
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

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedFile   string
	}{
		{
			name:           "pass config flag",
			args:           []string{"--config", "config1.yaml"},
			expectedFile: 	"config.yaml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFile := setupConfigFile()
			defer removeConfigFile(configFile)

			os.Setenv("CONFIG_FILE", configFile)

			tw := &testWriter{}
			RootCmd.SetOut(tw) // stub to capture eventual output
			RootCmd.SetArgs(test.args)
			initConfig()

			if err := RootCmd.Execute(); err != nil {
				t.Errorf("Error executing command: %v", err)
			}

			actualConfigFile := viper.ConfigFileUsed()
			t.Log(actualConfigFile)
			t.Log(test.expectedFile)
			if (filepath.Base(actualConfigFile)	!= test.expectedFile) {
				t.Errorf("Expected config file %s, got %s", filepath.Base(actualConfigFile), test.expectedFile)
			}
		})
	}
}
