// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

// GetContext returns the v2 context from the CreateDataSourceRequest data source.
func (r *CreateDataSourceRequest) GetContext() *ContextV2 {
	return r.DataSource.GetContext()
}

// GetContext returns the v2 context embedded in the UpdateDataSourceRequest
// data source.
func (r *UpdateDataSourceRequest) GetContext() *ContextV2 {
	return r.DataSource.GetContext()
}
