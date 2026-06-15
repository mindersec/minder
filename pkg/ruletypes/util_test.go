// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletypes_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/ruletypes"
)

func TestGetPBReleasePhaseFromDBReleaseStatus(t *testing.T) {
	t.Parallel()

	alpha := db.ReleaseStatusAlpha
	beta := db.ReleaseStatusBeta
	ga := db.ReleaseStatusGa
	deprecated := db.ReleaseStatusDeprecated
	bogus := db.ReleaseStatus("bogus")

	tests := []struct {
		name      string
		input     *db.ReleaseStatus
		expected  pb.RuleTypeReleasePhase
		expectErr bool
	}{
		{
			name:     "nil returns unspecified",
			input:    nil,
			expected: pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED,
		},
		{
			name:     "alpha maps correctly",
			input:    &alpha,
			expected: pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
		},
		{
			name:     "beta maps correctly",
			input:    &beta,
			expected: pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA,
		},
		{
			name:     "ga maps correctly",
			input:    &ga,
			expected: pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_GA,
		},
		{
			name:     "deprecated maps correctly",
			input:    &deprecated,
			expected: pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_DEPRECATED,
		},
		{
			name:      "invalid status returns error",
			input:     &bogus,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ruletypes.GetPBReleasePhaseFromDBReleaseStatus(tt.input)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDBReleaseStatusFromPBReleasePhase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     pb.RuleTypeReleasePhase
		expected  db.ReleaseStatus
		expectErr bool
	}{
		{
			name:     "alpha maps correctly",
			input:    pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
			expected: db.ReleaseStatusAlpha,
		},
		{
			name:     "beta maps correctly",
			input:    pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA,
			expected: db.ReleaseStatusBeta,
		},
		{
			name:     "ga maps correctly",
			input:    pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_GA,
			expected: db.ReleaseStatusGa,
		},
		{
			name:     "deprecated maps correctly",
			input:    pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_DEPRECATED,
			expected: db.ReleaseStatusDeprecated,
		},
		{
			name:     "unspecified defaults to ga",
			input:    pb.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED,
			expected: db.ReleaseStatusGa,
		},
		{
			name:      "invalid enum value returns error",
			input:     pb.RuleTypeReleasePhase(999),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ruletypes.GetDBReleaseStatusFromPBReleasePhase(tt.input)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.expected, *result)
		})
	}
}
