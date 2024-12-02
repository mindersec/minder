// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import v1datasources "github.com/mindersec/minder/pkg/datasources/v1"

// GetContext returns the v2 context from the CreateDataSourceRequest data source.
func (r *CreateDataSourceRequest) GetContext() *ContextV2 {
	return r.DataSource.GetContext()
}

// GetContext returns the v2 context embedded in the UpdateDataSourceRequest
// data source.
func (r *UpdateDataSourceRequest) GetContext() *ContextV2 {
	return r.DataSource.GetContext()
}

// GetDriverType returns the string representation of the driver type of the data source.
func (ds *DataSource) GetDriverType() string {
	if ds == nil {
		return ""
	}

	switch ds.GetDriver().(type) {
	case *DataSource_Rest:
		return v1datasources.DataSourceDriverRest
	case *DataSource_Deps:
		return v1datasources.DataSourceDriverDeps
	default:
		return "unknown"
	}
}
