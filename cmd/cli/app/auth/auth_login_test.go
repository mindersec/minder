// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global rootCmd / viper state
func TestLoginCommand_RejectsArgs(t *testing.T) {
	rootCmd := app.RootCmd
	cli.ResetEntireTree(rootCmd)
	viper.Reset()

	stderrBuf := new(bytes.Buffer)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs([]string{"auth", "login", "unexpected-arg"})

	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown command "unexpected-arg" for "minder auth login"`)
}
