// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package v1 provides the interfaces and types for the data sources.
package v1

import "context"

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

const (
	// DataSourceDriverRest is the driver type for a REST data source.
	DataSourceDriverRest = "rest"
)

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
	// ValidateArgs validates the arguments of the function.
	ValidateArgs(obj any) error
	// ValidateUpdate validates the update to the data source.
	// The data source implementation should respect the update and return an error
	// if the update is invalid.
	ValidateUpdate(obj any) error
	// Call calls the function with the given arguments.
	// It is the responsibility of the data source implementation to handle the call.
	// It is also the responsibility of the caller to validate the arguments
	// before calling the function.
	Call(ctx context.Context, args any) (any, error)
}

// DataSource is the interface that a data source must implement.
// It implements several functions that will be used by the engine to
// interact with external systems. These get taken into used by the Evaluator.
// Moreover, a data source must be able to validate an update to itself.
type DataSource interface {
	// GetFuncs returns the functions that the data source provides.
	GetFuncs() map[DataSourceFuncKey]DataSourceFuncDef
}

type DataSourceContext struct {
}
