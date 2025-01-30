// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

type idContextKeyType struct{}

var idContextKey idContextKeyType

func WithIdentityContext(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, idContextKey, identity)
}

func IdentityFromContext(ctx context.Context) *Identity {
	id, ok := ctx.Value(idContextKey).(*Identity)
	if !ok {
		return nil
	}
	return id
}