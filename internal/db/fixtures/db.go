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

// Package fixtures contains code for creating DB fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	mockdb "github.com/stacklok/minder/database/mock"
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
