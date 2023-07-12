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
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viper.SetConfigName("config")
			viper.AddConfigPath("../../../..")
			viper.SetConfigType("yaml")
			viper.AutomaticEnv()
			fmt.Println(viper.GetViper().ConfigFileUsed())

			tw := &util.TestWriter{}
			app.RootCmd.SetOut(tw) // stub to capture eventual output
			app.RootCmd.SetArgs(test.args)
			AuthCmd.Use = test.expectedOutput

			assert.NoError(t, app.RootCmd.Execute(), "Error on execute")
			assert.Contains(t, tw.Output, test.expectedOutput)
		})
	}
}
