// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"reflect"
)

type rpcKey struct {
	clientType reflect.Type
}

// WithRPCClient injects the provided RPC client into the context,
func WithRPCClient[T any](ctx context.Context, client T) context.Context {
	key := rpcKey{clientType: reflect.TypeOf((*T)(nil)).Elem()}
	return context.WithValue(ctx, key, client)
}

// GetRPCClient extracts the generic RPC client from the provided context.
func GetRPCClient[T any](ctx context.Context) (T, bool) {
	key := rpcKey{clientType: reflect.TypeOf((*T)(nil)).Elem()}
	client, ok := ctx.Value(key).(T)
	return client, ok
}
