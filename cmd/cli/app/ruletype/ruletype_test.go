// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Cannot run in parallel because it swaps a global client creator
func TestRuleTypeRootCommand(t *testing.T) {
	// setup the command buffer
	buf := new(bytes.Buffer)
	ruleTypeCmd.SetOut(buf)
	ruleTypeCmd.SetErr(buf)

	err := ruleTypeCmd.RunE(ruleTypeCmd, []string{})
	require.NoError(t, err)

	checkGoldenFile(t, "ruletype_root.help", buf.String())

	projectFlag := ruleTypeCmd.PersistentFlags().Lookup("project")
	require.NotNil(t, projectFlag, "project flag should be registered")
	assert.Equal(t, "j", projectFlag.Shorthand)
}
