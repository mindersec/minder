// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package properties

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/entities/properties"
	pbinternal "github.com/mindersec/minder/internal/proto"
)

func TestEntityInstanceV1FromReleaseProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		props          map[string]any
		expected       *pbinternal.Release
		expectedErrMsg string
	}{
		{
			name: "Valid properties",
			props: map[string]any{
				properties.PropertyUpstreamID: "12345",
				properties.ReleasePropertyTag: "v1.0.0",
				ReleasePropertyOwner:          "owner",
				ReleasePropertyRepo:           "repo",
			},
			expected: &pbinternal.Release{
				Name:       "owner/repo/v1.0.0",
				UpstreamId: "12345",
				Repo:       "repo",
				Owner:      "owner",
				Properties: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						properties.PropertyUpstreamID: structpb.NewStringValue("12345"),
						properties.ReleasePropertyTag: structpb.NewStringValue("v1.0.0"),
						ReleasePropertyOwner:          structpb.NewStringValue("owner"),
						ReleasePropertyRepo:           structpb.NewStringValue("repo"),
					},
				},
			},
		},
		{
			name: "Missing upstream ID",
			props: map[string]any{
				properties.ReleasePropertyTag: "v1.0.0",
				ReleasePropertyOwner:          "owner",
				ReleasePropertyRepo:           "repo",
			},
			expectedErrMsg: "upstream ID not found or invalid",
		},
		{
			name: "Missing tag",
			props: map[string]any{
				properties.PropertyUpstreamID: "12345",
				ReleasePropertyOwner:          "owner",
				ReleasePropertyRepo:           "repo",
			},
			expectedErrMsg: "tag not found or invalid",
		},
		{
			name: "Missing repo",
			props: map[string]any{
				properties.PropertyUpstreamID: "12345",
				properties.ReleasePropertyTag: "v1.0.0",
				ReleasePropertyOwner:          "owner",
			},
			expectedErrMsg: "repo not found or invalid",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			props, err := properties.NewProperties(tt.props)
			assert.NoError(t, err)

			result, err := EntityInstanceV1FromReleaseProperties(props)
			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
