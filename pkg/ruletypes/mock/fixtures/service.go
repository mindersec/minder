// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating ProfileService fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	"errors"

	mockrulesvc "github.com/mindersec/minder/pkg/ruletypes/mock"
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

func WithSuccessfulUpsertRuleType(mock RuleTypeSvcMock) {
	mock.EXPECT().
		UpsertRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func WithFailedUpsertRuleType(mock RuleTypeSvcMock) {
	mock.EXPECT().
		UpsertRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func WithSuccessfulCreateRuleType(mock RuleTypeSvcMock) {
	mock.EXPECT().
		CreateRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
}

func WithSuccessfulUpdateRuleType(mock RuleTypeSvcMock) {
	mock.EXPECT().
		UpdateRuleType(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)
}
