// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/pkg/util"
)

func TestCobraMain(t *testing.T) {
	t.Parallel()

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
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			viper.SetConfigName("config")
			viper.AddConfigPath("../../../..")
			viper.SetConfigType("yaml")
			viper.AutomaticEnv()

			tw := &util.TestWriter{}
			app.RootCmd.SetOut(tw) // stub to capture eventual output
			app.RootCmd.SetArgs(test.args)
			AuthCmd.Use = test.expectedOutput

			assert.NoError(t, app.RootCmd.Execute(), "Error on execute")
			assert.Contains(t, tw.Output, test.expectedOutput)
		})
	}
}
