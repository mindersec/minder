// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package options provides necessary interfaces and implementations
// for implementing evaluator configuration options.
package options

import (
	"github.com/open-feature/go-sdk/openfeature"

	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
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

// HasDebuggerSupport interface should be implemented by evaluation
// engines that support interactive debugger. Currently, only
// REGO-based engines should implement this.
type HasDebuggerSupport interface {
	SetDebugFlag(bool) error
}

// WithDebugger sets the evaluation engine to start an interactive
// debugging session. This MUST NOT be used in backend servers, and is
// only meant to be used in CLI tools.
func WithDebugger(flag bool) Option {
	return func(e interfaces.Evaluator) error {
		inner, ok := e.(HasDebuggerSupport)
		if !ok {
			return nil
		}
		return inner.SetDebugFlag(flag)
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
func WithDataSources(ds *v1datasources.DataSourceRegistry) Option {
	return func(e interfaces.Evaluator) error {
		inner, ok := e.(SupportsDataSources)
		if !ok {
			return nil
		}
		inner.RegisterDataSources(ds)
		return nil
	}
}
