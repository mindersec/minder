// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"errors"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCreatedAt(t *testing.T) {
	t.Parallel()

	buildTime := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	// poison is returned by the config getter for cases where the annotation is
	// present: if the implementation wrongly consulted the config, the assertion
	// against buildTime (or the expected error) would fail.
	poison := time.Date(1999, time.December, 31, 23, 59, 59, 0, time.UTC)
	epoch := time.Unix(0, 0).UTC()

	configReturning := func(created time.Time) func() (*v1.ConfigFile, error) {
		return func() (*v1.ConfigFile, error) {
			return &v1.ConfigFile{Created: v1.Time{Time: created}}, nil
		}
	}
	configFailing := func() (*v1.ConfigFile, error) {
		return nil, errors.New("config blob unavailable")
	}

	tests := []struct {
		name       string
		man        *v1.Manifest
		configFile func() (*v1.ConfigFile, error)
		want       time.Time
		wantErr    bool
	}{
		{
			name: "annotation present is preferred over config",
			man: &v1.Manifest{Annotations: map[string]string{
				imgspecv1.AnnotationCreated: buildTime.Format(time.RFC3339),
			}},
			configFile: configReturning(poison),
			want:       buildTime,
		},
		{
			name: "invalid annotation returns error and does not use config",
			man: &v1.Manifest{Annotations: map[string]string{
				imgspecv1.AnnotationCreated: "not-a-timestamp",
			}},
			configFile: configReturning(poison),
			wantErr:    true,
		},
		{
			name:       "missing annotation falls back to config created",
			man:        &v1.Manifest{},
			configFile: configReturning(buildTime),
			want:       buildTime,
		},
		{
			name:       "epoch config timestamp is preserved",
			man:        &v1.Manifest{Annotations: map[string]string{}},
			configFile: configReturning(epoch),
			want:       epoch,
		},
		{
			name:       "zero config timestamp is preserved",
			man:        &v1.Manifest{},
			configFile: configReturning(time.Time{}),
			want:       time.Time{},
		},
		{
			name:       "config fetch error is propagated",
			man:        &v1.Manifest{},
			configFile: configFailing,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveCreatedAt(tc.man, tc.configFile)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Truef(t, got.Equal(tc.want), "got %s, want %s", got, tc.want)
		})
	}
}
