// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating ProfileService fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
	"go.uber.org/mock/gomock"
)

type (
	ProviderManagerMock        = *mockmanager.MockProviderManager
	ProviderManagerMockBuilder = func(*gomock.Controller) ProviderManagerMock
)

func NewProviderManagerMock(opts ...func(mock ProviderManagerMock)) ProviderManagerMockBuilder {
	return func(ctrl *gomock.Controller) ProviderManagerMock {
		mock := mockmanager.NewMockProviderManager(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault = errors.New("error during provider manager operation")
)

func WithSuccessfulInstantiateFromID(provider provinfv1.Provider) func(mock ProviderManagerMock) {
	return func(mock ProviderManagerMock) {
		mock.EXPECT().
			InstantiateFromID(gomock.Any(), gomock.Any()).
			Return(provider, nil).
			AnyTimes()
	}
}

func WithFailedInstantiateFromID(mock ProviderManagerMock) {
	mock.EXPECT().
		InstantiateFromID(gomock.Any(), gomock.Any()).
		Return(nil, errDefault).
		AnyTimes()
}
