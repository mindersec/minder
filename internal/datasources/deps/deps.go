// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps implements a data source that extracts dependencies from
// a filesystem or file.
package deps

import (
	"errors"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

type depsDataSource struct {
	handlers map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef
}

// GetFuncs implements the v1datasources.DataSource interface.
func (r *depsDataSource) GetFuncs() map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef {
	return r.handlers
}

// NewDepsDataSource returns a new dependencies datasource
func NewDepsDataSource(ds *minderv1.DepsDataSource) (v1datasources.DataSource, error) {
	if ds == nil {
		return nil, errors.New("rest data source is nil")
	}

	if ds.GetDef() == nil {
		return nil, errors.New("rest data source definition is nil")
	}

	out := &depsDataSource{
		handlers: make(map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef, len(ds.GetDef())),
	}

	for key, handlerCfg := range ds.GetDef() {
		handler, err := newHandlerFromDef(handlerCfg)
		if err != nil {
			return nil, err
		}

		out.handlers[v1datasources.DataSourceFuncKey(key)] = handler
	}

	return out, nil
}
