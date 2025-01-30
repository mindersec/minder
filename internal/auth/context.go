// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

type idContextKeyType struct{}

var idContextKey idContextKeyType

// WithIdentityContext stores the identity in the context.
func WithIdentityContext(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, idContextKey, identity)
}

// IdentityFromContext retrieves the caller's identity from the context.
// This may return `nil` or an empty Identity if the user is not authenticated.
func IdentityFromContext(ctx context.Context) *Identity {
	id, ok := ctx.Value(idContextKey).(*Identity)
	if !ok {
		return nil
	}
	return id
}
