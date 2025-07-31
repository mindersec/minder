// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package options provides necessary interfaces and implementations
// for implementing evaluator configuration options.
package options

import (
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/flags"
)

// SupportsFlags interface advertises the fact that the implementer
// can use an `openfeature` client to check for flags being set.
type SupportsFlags interface {
	SetFlagsClient(client flags.Interface) error
}

// Option is a function that takes an evaluator and does some
// unspecified operation to it, returning an error in case of failure.
type Option func(interfaces.Evaluator) error

// WithFlagsClient provides the evaluation engine with an
// `openfeature` client. In case the given evaluator dows not support
// feature flags, WithFlagsClient silently ignores the error.
func WithFlagsClient(client flags.Interface) interfaces.Option {
	return func(e interfaces.Evaluator) error {
		inner, ok := e.(SupportsFlags)
		if !ok {
			return nil
		}
		return inner.SetFlagsClient(client)
	}
}

// SupportsDataSources interface advertises the fact that the implementer
// can register data sources with the evaluator.
type SupportsDataSources interface {
	RegisterDataSources(ds *v1datasources.DataSourceRegistry)
}

// WithDataSources provides the evaluation engine with a list of data sources
// to register. In case the given evaluator does not support data sources,
// WithDataSources silently ignores the error.
func WithDataSources(ds *v1datasources.DataSourceRegistry) interfaces.Option {
	return func(e interfaces.Evaluator) error {
		inner, ok := e.(SupportsDataSources)
		if !ok {
			return nil
		}
		inner.RegisterDataSources(ds)
		return nil
	}
}
