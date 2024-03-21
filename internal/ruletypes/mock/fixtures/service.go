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
	mockrulesvc "github.com/stacklok/minder/internal/ruletypes/mock"
	"go.uber.org/mock/gomock"
)

type (
	RuleTypeSvcMock        = *mockrulesvc.MockRuleTypeService
	RuleTypeSvcMockBuilder = func(*gomock.Controller) RuleTypeSvcMock
)

func NewRuleTypeServiceMock(opts ...func(mock RuleTypeSvcMock)) RuleTypeSvcMockBuilder {
	return func(ctrl *gomock.Controller) RuleTypeSvcMock {
		mock := mockrulesvc.NewMockRuleTypeService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

var (
	errDefault = errors.New("error during rule type service operation")
)

func WithSuccessfulUpdateSubscriptionProfile(mock RuleTypeSvcMock) {
	mock.EXPECT().
		UpsertSubscriptionRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedUpdateSubscriptionProfile(mock RuleTypeSvcMock) {
	mock.EXPECT().
		UpsertSubscriptionRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}
