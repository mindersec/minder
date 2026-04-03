// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package constants

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionStruct_String(t *testing.T) {
	t.Parallel()

	vvs := &versionStruct{
		Version:   "v1.2.3",
		GoVersion: "go1.25.8",
		Time:      "2026-04-03T22:33:03Z",
		Commit:    "abcdef123456",
		OS:        "linux",
		Arch:      "amd64",
		Modified:  true,
	}

	result := vvs.String()

	assert.Contains(t, result, "Version: v1.2.3")
	assert.Contains(t, result, "Go Version: go1.25.8")
	assert.Contains(t, result, "Git Commit: abcdef123456")
	assert.Contains(t, result, "Commit Date: 2026-04-03T22:33:03Z")
	assert.Contains(t, result, "OS/Arch: linux/amd64")
	assert.Contains(t, result, "Dirty: true")

	// Ensure it has multiple lines
	lines := strings.Split(result, "\n")
	assert.True(t, len(lines) >= 6)
}
