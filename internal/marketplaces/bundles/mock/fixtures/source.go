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

//nolint:all
package fixtures

import (
	mockbundle "github.com/stacklok/minder/internal/marketplaces/bundles/mock"
	"github.com/stacklok/minder/pkg/mindpak/reader"
	"github.com/stacklok/minder/pkg/mindpak/sources"
	"go.uber.org/mock/gomock"
)

type (
	SourceMock        = *mockbundle.MockBundleSource
	SourceMockBuilder = func(*gomock.Controller) SourceMock
)

func NewBundleSourceMock(opts ...func(mock SourceMock)) SourceMockBuilder {
	return func(ctrl *gomock.Controller) SourceMock {
		mock := mockbundle.NewMockBundleSource(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func WithSuccessfulGetBundle(bundle reader.BundleReader) func(SourceMock) {
	return func(mock SourceMock) {
		mock.EXPECT().
			GetBundle(gomock.Any()).
			Return(bundle, nil)
	}
}

func WithFailedGetBundle(mock SourceMock) {
	mock.EXPECT().
		GetBundle(gomock.Any()).
		Return(nil, sources.ErrBundleNotFound)
}
