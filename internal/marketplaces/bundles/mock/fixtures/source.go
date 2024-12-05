// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

//nolint:all
package fixtures

import (
	mockbundle "github.com/mindersec/minder/internal/marketplaces/bundles/mock"
	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/reader"
	"github.com/mindersec/minder/pkg/mindpak/sources"
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

func WithListBundles(bundleID mindpak.BundleID) func(SourceMock) {
	return func(mock SourceMock) {
		mock.EXPECT().
			ListBundles().
			Return([]mindpak.BundleID{bundleID}, nil)
	}
}
