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

package auth

import (
	"os"
	"testing"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/pkg/util"
)

func TestCobraMain(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "auth command",
			args:           []string{"auth"},
			expectedOutput: "auth called\n",
		},
		{
			name:           "auth list command",
			args:           []string{"auth", "list"},
			expectedOutput: "auth list called\n",
		},
		{
			name:           "auth revoke command",
			args:           []string{"auth", "revoke"},
			expectedOutput: "auth revoke called\n",
		},
		{
			name:           "auth delete command",
			args:           []string{"auth", "delete"},
			expectedOutput: "auth delete called\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFile := util.SetupConfigFile()
			defer util.RemoveConfigFile(configFile)

			os.Setenv("CONFIG_FILE", configFile)

			tw := &util.TestWriter{}
			app.RootCmd.SetOut(tw) // stub to capture eventual output
			app.RootCmd.SetArgs(test.args)

			if err := app.RootCmd.Execute(); err != nil {
				t.Errorf("Error executing command: %v", err)
			}

			if tw.Output != test.expectedOutput {
				t.Errorf("Expected %q, got %q", test.expectedOutput, tw.Output)
			}

		})
	}
}
