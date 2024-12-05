// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"

	"github.com/puzpuzpuz/xsync/v3"
)

var (
	// ErrDuplicateDataSourceFuncKey is the error returned when a data source
	// function key is already registered.
	ErrDuplicateDataSourceFuncKey = errors.New("duplicate data source function key")
)

// DataSourceRegistry is the interface that a data source registry must implement.
// It provides methods to register a data source and get all functions that
// data sources provide globally.
type DataSourceRegistry struct {
	r *xsync.MapOf[DataSourceFuncKey, DataSourceFuncDef]
}

// NewDataSourceRegistry creates a new data source registry.
func NewDataSourceRegistry() *DataSourceRegistry {
	return &DataSourceRegistry{
		r: xsync.NewMapOf[DataSourceFuncKey, DataSourceFuncDef](),
	}
}

// RegisterDataSource registers a data source with the registry.
// Note that the name of the data source must be unique.
func (reg *DataSourceRegistry) RegisterDataSource(name string, ds DataSource) (err error) {
	for key, f := range ds.GetFuncs() {
		funckey := makeKey(name, key)
		if _, ok := reg.r.Load(funckey); ok {
			return fmt.Errorf("%w: %s", ErrDuplicateDataSourceFuncKey, funckey)
		}

		// We only flush the store if there was no error
		defer func() {
			if err == nil {
				reg.r.Store(funckey, f)
			}
		}()
	}

	return nil
}

// GetFuncs returns all functions that data sources provide globally.
func (reg *DataSourceRegistry) GetFuncs() map[DataSourceFuncKey]DataSourceFuncDef {
	out := make(map[DataSourceFuncKey]DataSourceFuncDef, reg.r.Size())
	reg.r.Range(func(key DataSourceFuncKey, value DataSourceFuncDef) bool {
		out[key] = value
		return true
	})

	return out
}

func makeKey(name string, key DataSourceFuncKey) DataSourceFuncKey {
	return DataSourceFuncKey(name + "." + key.String())
}
