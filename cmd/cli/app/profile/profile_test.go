// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"strings"
	"testing"

	"github.com/mindersec/minder/cmd/cli/app/testutils"
)

func TestProfileCmd_Help(t *testing.T) {
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := testutils.RunCommand(ProfileCmd, tt.args...)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, "Usage") || !strings.Contains(output, "Flags") {
				t.Errorf("unexpected help output:\n%s", output)
			}
		})
	}
}
