// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package v1 provides the interfaces and types for the data sources.
package v1

import (
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// DataFuncSourceArgs is the type of the arguments that a data function source
// can take.
type DataFuncSourceArgs *jsonschema.Schema

// DataSourceFuncKey is the key that uniquely identifies a data source function.
type DataSourceFuncKey string

// String returns the string representation of the data source function key.
func (k DataSourceFuncKey) String() string {
	return string(k)
}

// DataSourceFuncDef is the definition of a data source function.
// It contains the key that uniquely identifies the function and the arguments
// that the function can take.
type DataSourceFuncDef interface {
	GetKey() DataSourceFuncKey
	GetArgs() DataFuncSourceArgs
	ValidateArgs(obj any) error
	Call(args any) (any, error)
}

// DataSource is the interface that a data source must implement.
// It implements several functions that will be used by the engine to
// interact with external systems. These get taken into used by the Evaluator.
// Moreover, a data source must be able to validate an update to itself.
type DataSource interface {
	GetFuncs() []DataSourceFuncDef
	ValidateUpdate(DataSource) error
}
