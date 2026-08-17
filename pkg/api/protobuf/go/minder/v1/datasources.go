// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

const (
	// DataSourceDriverStruct is the driver type for the structured data source.
	DataSourceDriverStruct = "structured"
	// DataSourceDriverRest is the driver type for a REST data source.
	DataSourceDriverRest = "rest"
)

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
		return DataSourceDriverRest
	case *DataSource_Structured:
		return DataSourceDriverStruct
	default:
		return "unknown"
	}
}
