// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rest implements a REST data source.
//
// The REST data source is a data source that can be used to fetch data from a
// REST API. The data source is defined in the Minder API as a REST data source
// definition. The definition contains a set of handlers that define how to
// fetch data from the REST API.
//
// It gives the caller a simple structured output to represent the
// result of the REST call.
//
// An example of the output is:
//
//	{
//	  "status_code": 200,
//	  "body": {
//	    "key": "value"
//	  }
//	}
package rest

import (
	"errors"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

type restDataSource struct {
	handlers map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef
}

// ensure that restDataSource implements the v1datasources.DataSource interface
var _ v1datasources.DataSource = (*restDataSource)(nil)

// GetFuncs implements the v1datasources.DataSource interface.
func (r *restDataSource) GetFuncs() map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef {
	return r.handlers
}

// NewRestDataSource builds a new REST data source.
func NewRestDataSource(rest *minderv1.RestDataSource) (v1datasources.DataSource, error) {
	if rest == nil {
		return nil, errors.New("rest data source is nil")
	}

	if rest.GetDef() == nil {
		return nil, errors.New("rest data source definition is nil")
	}

	out := &restDataSource{
		handlers: make(map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef, len(rest.GetDef())),
	}

	for key, handlerCfg := range rest.GetDef() {
		handler, err := newHandlerFromDef(handlerCfg)
		if err != nil {
			return nil, err
		}

		out.handlers[v1datasources.DataSourceFuncKey(key)] = handler
	}

	return out, nil
}
