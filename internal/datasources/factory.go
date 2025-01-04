// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package datasources implements data sources for Minder.
package datasources

import (
	"fmt"

	"github.com/mindersec/minder/internal/datasources/rest"
	"github.com/mindersec/minder/internal/datasources/structured"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

// BuildFromProtobuf is a factory function that builds a new data source based on the given
// data source type.
func BuildFromProtobuf(ds *minderv1.DataSource) (v1datasources.DataSource, error) {
	if ds == nil {
		return nil, fmt.Errorf("data source is nil")
	}

	if ds.GetDriver() == nil {
		return nil, fmt.Errorf("data source driver is nil")
	}

	switch ds.GetDriver().(type) {
	case *minderv1.DataSource_Structured:
		return structured.NewStructDataSource(ds.GetStructured())
	case *minderv1.DataSource_Rest:
		return rest.NewRestDataSource(ds.GetRest())
	default:
		return nil, fmt.Errorf("unknown data source type: %T", ds)
	}
}
