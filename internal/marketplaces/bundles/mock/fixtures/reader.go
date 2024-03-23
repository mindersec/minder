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

// Package fixtures contains code for creating bundle fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"
	mockbundle "github.com/stacklok/minder/internal/marketplaces/bundles/mock"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stacklok/minder/pkg/mindpak"
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
