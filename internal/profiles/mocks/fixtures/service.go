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
	mockprofsvc "github.com/stacklok/minder/internal/profiles/mocks"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"go.uber.org/mock/gomock"
)

type (
	ProfileSvcMock        = *mockprofsvc.MockProfileService
	ProfileSvcMockBuilder = func(*gomock.Controller) ProfileSvcMock
)

func NewProfileServiceMock(opts ...func(mock ProfileSvcMock)) ProfileSvcMockBuilder {
	return func(ctrl *gomock.Controller) ProfileSvcMock {
		mock := mockprofsvc.NewMockProfileService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault = errors.New("error during profile service operation")
)

func WithSuccessfulCreateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		CreateSubscriptionProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&minderv1.Profile{}, nil)
}

func WithFailedCreateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		CreateSubscriptionProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithSuccessfulUpdateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		UpdateSubscriptionProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&minderv1.Profile{}, nil)
}

func WithFailedUpdateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		UpdateSubscriptionProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}
