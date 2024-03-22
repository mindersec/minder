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

// Package fixtures contains code for creating subscription fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"
	mocksubscription "github.com/stacklok/minder/internal/marketplaces/subscriptions/mock"
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
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedSubscribe(mock SubscriptionMock) {
	mock.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func WithSuccessfulCreateProfile(mock SubscriptionMock) {
	mock.EXPECT().
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedCreateProfile(mock SubscriptionMock) {
	mock.EXPECT().
		CreateProfile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func WithSuccessfulCreateRuleTypes(mock SubscriptionMock) {
	mock.EXPECT().
		CreateRuleTypes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedCreateRuleTypes(mock SubscriptionMock) {
	mock.EXPECT().
		CreateRuleTypes(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}
