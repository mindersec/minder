// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type rpcKey struct {
	clientType reflect.Type
}

// WithRPCClient injects the provided RPC client into the context.
func WithRPCClient[T any](ctx context.Context, client T) context.Context {
	key := rpcKey{clientType: reflect.TypeOf((*T)(nil)).Elem()}
	return context.WithValue(ctx, key, client)
}

// WithCLIClient is an alias for WithRPCClient kept for backwards/alternate usage
// by tests or callers expecting a "CLI"-named helper.
func WithCLIClient[T any](ctx context.Context, client T) context.Context {
	return WithRPCClient[T](ctx, client)
}

// GetRPCClient extracts the generic RPC client from the provided context.
func GetRPCClient[T any](ctx context.Context) (T, bool) {
	key := rpcKey{clientType: reflect.TypeOf((*T)(nil)).Elem()}
	client, ok := ctx.Value(key).(T)
	return client, ok
}

// Cleanup is a function type used to define a routine that releases resources.
type Cleanup = func()

// GetCLIClient takes a factory for a GRPC client service and returns
// a client, a cleanup function to close the connection and an error.
func GetCLIClient[T any](cmd *cobra.Command, client func(grpc.ClientConnInterface) T) (T, Cleanup, error) {
	var empty T

	ctx, cancel := GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := GetRPCClient[T](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return empty, nil, err
	}

	return client(conn), func() {
		cancel()
		_ = conn.Close()
	}, nil
}
