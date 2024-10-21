// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating ProviderStore fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	"github.com/mindersec/minder/internal/db"
	mockprov "github.com/mindersec/minder/internal/providers/mock"
	"go.uber.org/mock/gomock"
)

type (
	ProviderStoreMock        = *mockprov.MockProviderStore
	ProviderStoreMockBuilder = func(*gomock.Controller) ProviderStoreMock
)

func NewProviderStoreMock(opts ...func(mock ProviderStoreMock)) ProviderStoreMockBuilder {
	return func(ctrl *gomock.Controller) ProviderStoreMock {
		mock := mockprov.NewMockProviderStore(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault = errors.New("error during provider store operation")
)

func WithSuccessfulGetByID(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			GetByID(gomock.Any(), gomock.Eq(provider.ID)).
			Return(provider, nil)
	}
}

func WithSuccessfulGetByIDProject(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			GetByIDProject(gomock.Any(), gomock.Eq(provider.ID), gomock.Eq(provider.ProjectID)).
			Return(provider, nil)
	}
}

func WithSuccessfulGetByName(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			GetByName(gomock.Any(), gomock.Eq(provider.ProjectID), gomock.Eq(provider.Name)).
			Return(provider, nil)
	}
}

func WithSuccessfulGetByNameInSpecificProject(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			GetByNameInSpecificProject(gomock.Any(), gomock.Eq(provider.ProjectID), gomock.Eq(provider.Name)).
			Return(provider, nil)
	}
}

func WithSuccessfulGetByTraitInHierarchy(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			GetByTraitInHierarchy(gomock.Any(), gomock.Eq(provider.ProjectID), gomock.Any(), gomock.Eq(provider.Implements[0])).
			Return([]db.Provider{*provider}, nil)
	}
}

func WithSuccessfulDelete(provider *db.Provider) func(mock ProviderStoreMock) {
	return func(mock ProviderStoreMock) {
		mock.EXPECT().
			Delete(gomock.Any(), gomock.Eq(provider.ID), gomock.Eq(provider.ProjectID)).
			Return(nil)
	}
}

func WithFailedGetByID(mock ProviderStoreMock) {
	mock.EXPECT().
		GetByID(gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithFailedGetByIDProject(mock ProviderStoreMock) {
	mock.EXPECT().
		GetByIDProject(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithFailedGetByName(mock ProviderStoreMock) {
	mock.EXPECT().
		GetByName(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithFailedGetByNameInSpecificProject(mock ProviderStoreMock) {
	mock.EXPECT().
		GetByNameInSpecificProject(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithFailedGetByTraitInHierarchy(mock ProviderStoreMock) {
	mock.EXPECT().
		GetByTraitInHierarchy(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithFailedDelete(mock ProviderStoreMock) {
	mock.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}
