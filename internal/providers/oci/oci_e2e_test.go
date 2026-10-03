//go:build e2e

// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOCI_AuthE2E verifies that configured OAuth2 credentials are actually
// forwarded to the registry on every call. Run against a private image so that
// an unauthenticated request would return 401.
//
// Required env vars:
//
//	OCI_E2E_REGISTRY  - registry hostname, e.g. quay.io
//	OCI_E2E_NAMESPACE - namespace / org within the registry, e.g. myorg
//	OCI_E2E_IMAGE     - image name within the namespace, e.g. myimage
//	OCI_E2E_TOKEN     - OAuth2 bearer token (Quay.io robot token, Docker Hub access token, etc.)
//
// Example (Quay.io robot account):
//
//	OCI_E2E_REGISTRY=quay.io \
//	OCI_E2E_NAMESPACE=myorg \
//	OCI_E2E_IMAGE=myimage \
//	OCI_E2E_TOKEN=<robot-token> \
//	go test -tags=e2e ./internal/providers/oci/... -run TestOCI_AuthE2E -v
func TestOCI_AuthE2E(t *testing.T) {
	registry := requireEnv(t, "OCI_E2E_REGISTRY")
	namespace := requireEnv(t, "OCI_E2E_NAMESPACE")
	image := requireEnv(t, "OCI_E2E_IMAGE")
	token := requireEnv(t, "OCI_E2E_TOKEN")

	baseURL := registry + "/" + namespace
	cred := MockCredential{token: token}

	ctx := context.Background()

	t.Run("authenticated client can list tags", func(t *testing.T) {
		o := New(cred, registry, baseURL)
		tags, err := o.ListTags(ctx, image)
		require.NoError(t, err)
		assert.NotEmpty(t, tags, "expected at least one tag for %s/%s", namespace, image)
		t.Logf("tags: %v", tags)
	})

	t.Run("unauthenticated client cannot list tags on private image", func(t *testing.T) {
		o := New(nil, registry, baseURL)
		_, err := o.ListTags(ctx, image)
		assert.Error(t, err, "expected 401 without credentials")
	})

	t.Run("authenticated client can get digest", func(t *testing.T) {
		o := New(cred, registry, baseURL)
		tags, err := o.ListTags(ctx, image)
		require.NoError(t, err)
		require.NotEmpty(t, tags)

		digest, err := o.GetDigest(ctx, image, tags[0])
		require.NoError(t, err)
		assert.Regexp(t, `^sha256:[0-9a-f]{64}$`, digest)
		t.Logf("digest for %s: %s", tags[0], digest)
	})
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return v
}
