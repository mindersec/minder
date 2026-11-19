// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package validators_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/entities/service/validators"
	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestArtifactValidator_Validate(t *testing.T) {
	t.Parallel()

	containerProps := func(name string) *properties.Properties {
		return properties.NewProperties(map[string]any{
			properties.PropertyName:      name,
			oci.ArtifactPropertyRegistry: "docker.io",
		})
	}

	tests := []struct {
		name        string
		props       *properties.Properties
		errIs       error
		errContains string
	}{
		{
			name:  "allows container artifact with a tag",
			props: containerProps("alpine:latest"),
		},
		{
			name:  "allows container artifact with a dotted tag",
			props: containerProps("myimage:v1.2"),
		},
		{
			name:  "rejects container artifact without a tag",
			props: containerProps("alpine"),
			errIs: validators.ErrContainerArtifactTagRequired,
		},
		{
			name:  "rejects namespaced container artifact without a tag",
			props: containerProps("myorg/alpine"),
			errIs: validators.ErrContainerArtifactTagRequired,
		},
		{
			name:        "rejects container artifact with an empty name",
			props:       containerProps(""),
			errContains: "artifact name is empty",
		},
		{
			name: "skips artifacts that are not from a container registry provider",
			props: properties.NewProperties(map[string]any{
				properties.PropertyName: "myorg/mypackage",
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validators.NewArtifactValidator().Validate(context.Background(), tt.props, uuid.New())

			switch {
			case tt.errIs != nil:
				require.ErrorIs(t, err, tt.errIs)
			case tt.errContains != "":
				require.ErrorContains(t, err, tt.errContains)
			default:
				assert.NoError(t, err)
			}
		})
	}
}
