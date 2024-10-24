// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating subscription fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	mocksubscription "github.com/mindersec/minder/pkg/marketplaces/subscriptions/mock"
	"go.uber.org/mock/gomock"
)

type (
	SubscriptionMock        = *mocksubscription.MockSubscriptionService
	SubscriptionMockBuilder = func(*gomock.Controller) SubscriptionMock
)

func NewSubscriptionServiceMock(opts ...func(mock SubscriptionMock)) SubscriptionMockBuilder {
	return func(ctrl *gomock.Controller) SubscriptionMock {
		mock := mocksubscription.NewMockSubscriptionService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault = errors.New("error during marketplace operation")
)

func WithSuccessfulSubscribe(mock SubscriptionMock) {
	mock.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedSubscribe(mock SubscriptionMock) {
	mock.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func WithSuccessfulCreateProfile(mock SubscriptionMock) {
	mock.EXPECT().
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedCreateProfile(mock SubscriptionMock) {
	mock.EXPECT().
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}
