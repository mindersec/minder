// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating bundle fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	mockbundle "github.com/mindersec/minder/internal/marketplaces/bundles/mock"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/mindpak"
	"go.uber.org/mock/gomock"
)

type (
	BundleMock        = *mockbundle.MockBundleReader
	BundleMockBuilder = func(*gomock.Controller) BundleMock
)

func NewBundleReaderMock(opts ...func(mock BundleMock)) BundleMockBuilder {
	return func(ctrl *gomock.Controller) BundleMock {
		mock := mockbundle.NewMockBundleReader(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault      = errors.New("bundle operation failed")
	BundleName      = "healthcheck"
	BundleNamespace = "stacklok"
	BundleVersion   = "1.0.0"
)

func WithMetadata(mock BundleMock) {
	metadata := mindpak.Metadata{
		Name:      BundleName,
		Namespace: BundleNamespace,
		Version:   BundleVersion,
	}
	mock.EXPECT().
		GetMetadata().
		Return(&metadata)
}

func WithSuccessfulGetProfile(mock BundleMock) {
	mock.EXPECT().
		GetProfile(gomock.Any()).
		Return(&v1.Profile{}, nil)
}

func WithFailedGetProfile(mock BundleMock) {
	mock.EXPECT().
		GetProfile(gomock.Any()).
		Return(nil, errDefault)
}

func WithSuccessfulForEachRuleType(mock BundleMock) {
	type argType = func(*v1.RuleType) error
	var argument argType
	mock.EXPECT().
		ForEachRuleType(gomock.AssignableToTypeOf(argument)).
		DoAndReturn(func(fn argType) error {
			return fn(&v1.RuleType{})
		})
}

func WithFailedForEachRuleType(mock BundleMock) {
	mock.EXPECT().
		ForEachRuleType(gomock.Any()).
		Return(errDefault)
}

func WithSuccessfulForEachDataSource(mock BundleMock) {
	type argType = func(source *v1.DataSource) error
	var argument argType
	mock.EXPECT().
		ForEachDataSource(gomock.AssignableToTypeOf(argument)).
		DoAndReturn(func(fn argType) error {
			return fn(&v1.DataSource{})
		})
}

func WithFailedForEachDataSource(mock BundleMock) {
	mock.EXPECT().
		ForEachDataSource(gomock.Any()).
		Return(errDefault)
}
