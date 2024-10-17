// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package options provides necessary interfaces and implementations
// for implementing evaluator configuration options.
package options

import (
	"github.com/open-feature/go-sdk/openfeature"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// SupportsFlags interface advertises the fact that the implementer
// can use an `openfeature` client to check for flags being set.
type SupportsFlags interface {
	SetFlagsClient(client openfeature.IClient) error
}

// Option is a function that takes an evaluator and does some
// unspecified operation to it, returning an error in case of failure.
type Option func(interfaces.Evaluator) error

// WithFlagsClient provides the evaluation engine with an
// `openfeature` client. In case the given evaluator dows not support
// feature flags, WithFlagsClient silently ignores the error.
func WithFlagsClient(client openfeature.IClient) Option {
	return func(e interfaces.Evaluator) error {
		inner, ok := e.(SupportsFlags)
		if !ok {
			return nil
		}
		return inner.SetFlagsClient(client)
	}
}
