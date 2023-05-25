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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package group

import (
	"testing"

	"github.com/spf13/viper"
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
			name:           "group command",
			args:           []string{"group"},
			expectedOutput: "group called\n",
		},
		{
			name:           "group list command",
			args:           []string{"group", "list"},
			expectedOutput: "group list called\n",
		},
		{
			name:           "group delete command",
			args:           []string{"group", "delete"},
			expectedOutput: "group delete called\n",
		},
		{
			name:           "group create command",
			args:           []string{"group", "create"},
			expectedOutput: "group create called\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viper.SetConfigName("config")
			viper.AddConfigPath("../../../..")
			viper.SetConfigType("yaml")
			viper.AutomaticEnv()

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
