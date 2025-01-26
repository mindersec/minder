// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package structured implements a data source that parses and returns
// structured data from files stored in a filesystem.
package structured

import (
	"errors"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

type structDataSource struct {
	handlers map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef
}

// ensure that restDataSource implements the v1datasources.DataSource interface
var _ v1datasources.DataSource = (*structDataSource)(nil)

// GetFuncs implements the v1datasources.DataSource interface.
func (r *structDataSource) GetFuncs() map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef {
	return r.handlers
}

// NewStructDataSource builds a new REST data source.
func NewStructDataSource(sds *minderv1.StructDataSource) (v1datasources.DataSource, error) {
	if sds == nil {
		return nil, errors.New("rest data source is nil")
	}

	if sds.GetDef() == nil {
		return nil, errors.New("rest data source definition is nil")
	}

	out := &structDataSource{
		handlers: make(map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef, len(sds.GetDef())),
	}

	for key, handlerCfg := range sds.GetDef() {
		handler, err := newHandlerFromDef(handlerCfg)
		if err != nil {
			return nil, err
		}

		out.handlers[v1datasources.DataSourceFuncKey(key)] = handler
	}

	return out, nil
}
