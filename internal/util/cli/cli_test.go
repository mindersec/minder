// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

// TestGetConfigDirPath tests the GetConfigDirPath function
func TestGetConfigDirPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		envVar         string
		expectedPath   string
		expectingError bool
	}{
		{
			name:           "XDG_CONFIG_HOME set",
			envVar:         "/custom/config",
			expectedPath:   "/custom/config/minder",
			expectingError: false,
		},
		{
			name:           "XDG_CONFIG_HOME is not set",
			envVar:         "",
			expectedPath:   filepath.Join(os.Getenv("HOME"), ".config", "minder"),
			expectingError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setEnvVar(t, XdgConfigHomeEnvVar, tt.envVar)
			path, err := cli.GetConfigDirPath()
			if (err != nil) != tt.expectingError {
				t.Errorf("expected error: %v, got: %v", tt.expectingError, err)
			}
			if path != tt.expectedPath {
				t.Errorf("expected path: %s, got: %s", tt.expectedPath, path)
			}
		})
	}
}
