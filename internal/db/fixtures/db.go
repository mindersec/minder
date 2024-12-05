// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating DB fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	mockdb "github.com/mindersec/minder/database/mock"
	"go.uber.org/mock/gomock"
)

type (
	DBMock        = *mockdb.MockStore
	DBMockBuilder = func(*gomock.Controller) DBMock
)

func NewDBMock(opts ...func(DBMock)) DBMockBuilder {
	return func(ctrl *gomock.Controller) DBMock {
		mock := mockdb.NewMockStore(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}
