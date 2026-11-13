// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeExpiration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		requested        time.Duration
		maxDuration      time.Duration
		expectedDuration time.Duration
	}{
		{
			name:             "requested shorter than max duration is honored",
			requested:        24 * time.Hour,
			maxDuration:      7 * 24 * time.Hour,
			expectedDuration: 24 * time.Hour,
		},
		{
			name:             "requested longer than max duration is capped at max",
			requested:        14 * 24 * time.Hour,
			maxDuration:      7 * 24 * time.Hour,
			expectedDuration: 7 * 24 * time.Hour,
		},
		{
			name:             "requested zero duration defaults to max",
			requested:        0,
			maxDuration:      7 * 24 * time.Hour,
			expectedDuration: 7 * 24 * time.Hour,
		},
		{
			name:             "requested negative duration defaults to max",
			requested:        -5 * time.Hour,
			maxDuration:      7 * 24 * time.Hour,
			expectedDuration: 7 * 24 * time.Hour,
		},
		{
			name:             "unconfigured max duration defaults to 7 days",
			requested:        0,
			maxDuration:      0,
			expectedDuration: 7 * 24 * time.Hour,
		},
		{
			name:             "requested shorter than unconfigured max is honored",
			requested:        48 * time.Hour,
			maxDuration:      0,
			expectedDuration: 48 * time.Hour,
		},
		{
			name:             "custom server max duration is respected",
			requested:        72 * time.Hour,
			maxDuration:      48 * time.Hour,
			expectedDuration: 48 * time.Hour,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := time.Now()
			got := ComputeExpiration(tt.requested, tt.maxDuration)
			after := time.Now()

			expectedMin := before.Add(tt.expectedDuration)
			expectedMax := after.Add(tt.expectedDuration)

			require.True(t, !got.Before(expectedMin) && !got.After(expectedMax),
				"computed expiration %v was outside expected range [%v, %v]", got, expectedMin, expectedMax)
		})
	}
}
