// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"strings"
	"testing"

	"github.com/mindersec/minder/cmd/cli/app/testutils"
)

func TestArtifactCmd_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "help flag",
			args: []string{"--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testutils.RunCommand(ArtifactCmd, tt.args...)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, "Usage") || !strings.Contains(output, "Flags") {
				t.Errorf("unexpected help output:\n%s", output)
			}
		})
	}
}
