// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
	"github.com/mindersec/minder/internal/util/cli"
)

/*
var (

	// envLockXdgConfigHome is a mutex to ensure that all the tests that run os.SetEnv("XDG_CONFIG_HOME") need to be prevented from running at the same time as each other.
	envLockXdgConfigHome = &sync.Mutex{}

	// envLockXdgConfig is a mutex to ensure that all the tests that run os.SetEnv("MINDER_AUTH_TOKEN") need to be prevented from running at the same time as each other.
	envLockMinderAuthToken = &sync.Mutex{}
	//nolint:gosec // This is not a hardcoded credential
	XdgConfigHomeEnvVar = "XDG_CONFIG_HOME"
	//nolint:gosec // This is not a hardcoded credential
	MinderAuthTokenEnvVar = "MINDER_AUTH_TOKEN"
)

// Based on tests, seemed to need one mutex per env var.
func setEnvVar(t *testing.T, envLock *sync.Mutex, env string, value string) {
	t.Helper() // Keep golangci-lint happy
	envLock.Lock()
	t.Cleanup(envLock.Unlock)

	originalEnvVal := os.Getenv(env)
	err := os.Setenv(env, value)
	if err != nil {
		t.Errorf("error setting %v: %v", env, err)
	}

	t.Cleanup(func() { _ = os.Setenv(env, originalEnvVal) })

}
*/

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
