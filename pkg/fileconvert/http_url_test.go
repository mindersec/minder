// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestResourcesFromPathsRemoteUnsupportedExtension(t *testing.T) {
	t.Parallel()

	var requested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requested.Store(true)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	resources, err := ResourcesFromPathsContext(context.Background(), t.Logf, server.URL+"/policy.yamlf")
	require.ErrorContains(t, err, "unsupported file extension")
	require.ErrorContains(t, err, ".yamlf")
	assert.Empty(t, resources)
	assert.False(t, requested.Load(), "unsupported URLs should be rejected before fetching")
}

func TestResourcesFromPathsRemoteYAMLWithTextPlainContentType(t *testing.T) {
	t.Parallel()

	const profileYAML = "type: profile\nversion: v1\nname: remote-profile\nrepository:\n  - type: sample-rule\n    def: {}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(profileYAML))
	}))
	defer server.Close()

	url := server.URL + "/minder-rules-and-profiles/raw/main/security-baseline/profile.yaml"
	resources, err := ResourcesFromPathsContext(context.Background(), t.Logf, url)
	require.NoError(t, err)
	require.Len(t, resources, 1)

	profile, ok := resources[0].(*minderv1.Profile)
	require.True(t, ok)
	assert.Equal(t, "remote-profile", profile.GetName())
}
