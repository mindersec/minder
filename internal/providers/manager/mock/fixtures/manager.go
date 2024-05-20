// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance cf.With the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fixtures contains code for creating ProfileService fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"
	mockmanager "github.com/stacklok/minder/internal/providers/manager/mock"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
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
