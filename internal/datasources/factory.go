// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package datasources implements data sources for Minder.
package datasources

import (
	"fmt"
	"net/http"

	"github.com/mindersec/minder/internal/datasources/rest"
	"github.com/mindersec/minder/internal/datasources/structured"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// BuildOption allows customizing the data source creation
type BuildOption func(*buildOptions)

type buildOptions struct {
	testOnlyTransport http.RoundTripper
}

// WithTestOnlyTransport allows passing a custom HTTP transport for testing.
func WithTestOnlyTransport(transport http.RoundTripper) BuildOption {
	return func(opts *buildOptions) {
		opts.testOnlyTransport = transport
	}
}

// BuildFromProtobuf is a factory function that builds a new data source based on the given
// data source type.
func BuildFromProtobuf(
	ds *minderv1.DataSource,
	provider provinfv1.Provider,
	opts ...BuildOption,
) (v1datasources.DataSource, error) {
	if ds == nil {
		return nil, fmt.Errorf("data source is nil")
	}

	if ds.GetDriver() == nil {
		return nil, fmt.Errorf("data source driver is nil")
	}

	bOpts := &buildOptions{}
	for _, opt := range opts {
		opt(bOpts)
	}

	switch ds.GetDriver().(type) {
	case *minderv1.DataSource_Structured:
		return structured.NewStructDataSource(ds.GetStructured())
	case *minderv1.DataSource_Rest:
		return rest.NewRestDataSource(ds.GetRest(), provider, rest.WithTestOnlyTransport(bOpts.testOnlyTransport))
	default:
		return nil, fmt.Errorf("unknown data source type: %T", ds)
	}
}
