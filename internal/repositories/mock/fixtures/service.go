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

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	mockghrepo "github.com/stacklok/minder/internal/repositories/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type (
	RepoServiceMock = *mockghrepo.MockRepositoryService
	RepoMockBuilder = func(*gomock.Controller) RepoServiceMock
)

func NewRepoService(opts ...func(RepoServiceMock)) RepoMockBuilder {
	return func(ctrl *gomock.Controller) RepoServiceMock {
		mock := mockghrepo.NewMockRepositoryService(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func WithSuccessfulCreate(
	projectID uuid.UUID,
	repoOwner string,
	repoName string,
	pbRepo *pb.Repository,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			CreateRepository(
				gomock.Any(),
				gomock.Any(),
				projectID,
				repoOwner,
				repoName,
			).
			Return(pbRepo, nil)
	}
}

func WithFailedCreate(
	err error,
	projectID uuid.UUID,
	repoOwner string,
	repoName string,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			CreateRepository(
				gomock.Any(),
				gomock.Any(),
				projectID,
				repoOwner,
				repoName,
			).
			Return(nil, err)
	}
}

func WithSuccessfulDeleteByIDDetailed(
	repositoryID uuid.UUID,
	projectID uuid.UUID,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			DeleteByID(
				gomock.Any(),
				repositoryID,
				projectID,
			).
			Return(nil)
	}
}

func WithSuccessfulDeleteByID() func(RepoServiceMock) {
	return WithFailedDeleteByID(nil)
}

func WithFailedDeleteByID(err error) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			DeleteByID(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(err)
	}
}

func WithSuccessfulDeleteByName() func(RepoServiceMock) {
	return WithFailedDeleteByName(nil)
}

func WithFailedDeleteByName(err error) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			DeleteByName(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(err)
	}
}

func WithSuccessfulListRepositories(
	repositories ...db.Repository,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			ListRepositories(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(repositories, nil).AnyTimes()
	}
}

func WithFailedListRepositories(err error) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			ListRepositories(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(nil, err).AnyTimes()
	}
}
