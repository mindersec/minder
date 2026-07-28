// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdTest_ExitCodeOnLoadError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	brokenFile := filepath.Join(tmpDir, "broken_test.star")
	err := os.WriteFile(brokenFile, []byte("this is invalid starlark syntax\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd := CmdTest()
	cmd.SetArgs([]string{brokenFile})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return an error, got nil")
	}

	if err.Error() != "one or more test files failed to load" {
		t.Errorf("unexpected error message: got %q, want 'one or more test files failed to load'", err.Error())
	}
}
