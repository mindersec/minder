// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
)

type rpcKeyType string

const rpcClientKey rpcKeyType = "rpcMockClient"

// WithRPCClient injects the provided RPC client into the context.
func WithRPCClient(ctx context.Context, client any) context.Context {
	return context.WithValue(ctx, rpcClientKey, client)
}

// GetRPCClient is a generic function that extracts the client
// and automatically formats it to the correct interface type (T).
func GetRPCClient[T any](ctx context.Context) (T, bool) {
	client, ok := ctx.Value(rpcClientKey).(T)
	return client, ok
}
