// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryHandlerEmitsLogEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	err := recoveryHandler(ctx, "boom")

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
	require.Equal(t, "boom", st.Message())

	logged := buf.String()
	require.Contains(t, logged, "recovered from panic")
	require.Contains(t, logged, "boom")
	require.Contains(t, logged, "stack")
}
