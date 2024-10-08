// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package options provides necessary interfaces and implementations
// for implementing evaluator configuration options.
package options

import (
	"github.com/open-feature/go-sdk/openfeature"

	"github.com/stacklok/minder/internal/engine/interfaces"
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
