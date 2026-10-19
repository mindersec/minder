// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ghcr

import (
	"context"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/require"
)

func TestListImages_NoResponseOnError(t *testing.T) {
	t.Parallel()

	// A canceled context makes go-github return a nil response with the error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	il := FromGitHubClient(github.NewClient(nil), "test-namespace")
	images, err := il.ListImages(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, images)
}
