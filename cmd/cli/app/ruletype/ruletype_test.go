// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func checkGoldenFile(t *testing.T, filename string, actual string) {
	t.Helper()
	goldenPath := filepath.Join("testdata", filename+".golden")

	if *update {
		err := os.MkdirAll("testdata", 0755)
		require.NoError(t, err)

		err = os.WriteFile(goldenPath, []byte(actual), 0644)
		require.NoError(t, err)
		t.Logf("Updated golden file: %s", goldenPath)
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "could not read golden file. Run 'go test ./... -update' to generate it")

	assert.Equal(t, string(expected), actual, "Output does not match golden file")
}

//nolint:unparam // keep generic signature to match helper patterns in other test files
func loadFixture(t *testing.T, filename string, target protoreflect.ProtoMessage) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("fixture", filename))
	require.NoError(t, err, "failed to read fixture file. Check if 'fixture/%s' exists", filename)

	err = protojson.Unmarshal(data, target)
	require.NoError(t, err, "failed to unmarshal fixture into protobuf")
}
