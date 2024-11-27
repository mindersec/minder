// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating DataSourceService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"errors"

	"github.com/google/uuid"
	mockdssvc "github.com/mindersec/minder/internal/datasources/service/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

var (
	errDefault = errors.New("error during data sources service operation")
)

func WithSuccessfulListDataSources(datasources ...*minderv1.DataSource) func(DataSourcesSvcMock) {
	return func(mock DataSourcesSvcMock) {
		mock.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(datasources, nil)
	}
}

func WithFailedListDataSources() func(DataSourcesSvcMock) {
	return func(mock DataSourcesSvcMock) {
		mock.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errDefault)
	}
}

func WithSuccessfulGetByName(projectID uuid.UUID, datasource *minderv1.DataSource) func(DataSourcesSvcMock) {
	return func(mock DataSourcesSvcMock) {
		mock.EXPECT().
			GetByName(gomock.Any(), datasource.Name, projectID, gomock.Any()).
			Return(datasource, nil)
	}
}

func WithNotFoundGetByName(projectID uuid.UUID) func(DataSourcesSvcMock) {
	return func(mock DataSourcesSvcMock) {
		mock.EXPECT().
			GetByName(gomock.Any(), gomock.Any(), projectID, gomock.Any()).
			Return(&minderv1.DataSource{}, errDefault)
	}
}

func WithFailedGetByName() func(DataSourcesSvcMock) {
	return func(mock DataSourcesSvcMock) {
		mock.EXPECT().
			GetByName(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errDefault)
	}
}
