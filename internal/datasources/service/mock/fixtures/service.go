// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating DataSourceService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	mockdssvc "github.com/mindersec/minder/internal/datasources/service/mock"
	"go.uber.org/mock/gomock"
)

type (
	DataSourcesSvcMock        = *mockdssvc.MockDataSourcesService
	DataSourcesSvcMockBuilder = func(*gomock.Controller) DataSourcesSvcMock
)

func NewDataSourcesServiceMock(opts ...func(mock DataSourcesSvcMock)) DataSourcesSvcMockBuilder {
	return func(ctrl *gomock.Controller) DataSourcesSvcMock {
		mock := mockdssvc.NewMockDataSourcesService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func WithSuccessfulUpsertDataSource(mock DataSourcesSvcMock) {
	mock.EXPECT().
		Upsert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}
