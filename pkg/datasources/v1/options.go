// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import "net/http"

// Option is a functional option for configuring a data source
type Option func(*Options)

// Options contains configuration for a data source
type Options struct {
	TestOnlyTransport http.RoundTripper
}

// WithTestOnlyTransport sets a custom HTTP transport, primarily used for testing.
func WithTestOnlyTransport(transport http.RoundTripper) Option {
	return func(opts *Options) {
		opts.TestOnlyTransport = transport
	}
}
