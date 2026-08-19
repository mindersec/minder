// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdTest_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      map[string]string // filename -> content
		args       []string          // arguments to mindev test (e.g. "--output junit")
		wantErrMsg string            // expected error substring, or empty if success expected
	}{
		{
			name: "only malformed returns non-zero",
			files: map[string]string{
				"broken_test.star": "this is invalid starlark syntax\n",
			},
			wantErrMsg: "one or more test files failed to load",
		},
		{
			name: "malformed plus passing returns non-zero",
			files: map[string]string{
				"broken_test.star": "this is invalid starlark syntax\n",
				"pass_test.star":   "def test_pass():\n  assert.eq(1, 1)\n",
			},
			wantErrMsg: "one or more test files failed to load",
		},
		{
			name: "malformed with junit output returns non-zero",
			files: map[string]string{
				"broken_test.star": "this is invalid starlark syntax\n",
			},
			args:       []string{"--output", "junit"},
			wantErrMsg: "one or more test files failed to load",
		},
		{
			name: "valid passing files return zero",
			files: map[string]string{
				"pass_test.star": "def test_pass():\n  assert.eq(1, 1)\n",
			},
			wantErrMsg: "",
		},
		{
			name: "valid failing files return non-zero",
			files: map[string]string{
				"fail_test.star": "def test_fail():\n  assert.eq(1, 2)\n",
			},
			wantErrMsg: "one or more tests failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()

			var paths []string
			for filename, content := range tt.files {
				filePath := filepath.Join(tmpDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
					t.Fatalf("failed to write test file: %v", err)
				}
				paths = append(paths, filePath)
			}

			cmd := CmdTest()
			cmd.SetArgs(append(paths, tt.args...))

			err := cmd.Execute()
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("expected Execute() to return an error containing %q, got nil", tt.wantErrMsg)
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("unexpected error message: got %q, want %q", err.Error(), tt.wantErrMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("expected Execute() to succeed, got error: %v", err)
				}
			}
		})
	}
}
