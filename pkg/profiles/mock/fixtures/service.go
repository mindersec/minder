// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating ProfileService fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockprofsvc "github.com/mindersec/minder/pkg/profiles/mock"
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
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&minderv1.Profile{}, nil)
}

func WithFailedCreateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func WithSuccessfulUpdateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&minderv1.Profile{}, nil)
}

func WithFailedUpdateSubscriptionProfile(mock ProfileSvcMock) {
	mock.EXPECT().
		UpdateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}
