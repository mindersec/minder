// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/entities/models"
	mockghrepo "github.com/mindersec/minder/internal/repositories/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	pbRepo *pb.Repository,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			CreateRepository(
				gomock.Any(),
				gomock.Any(),
				projectID,
				gomock.Any(),
			).
			Return(pbRepo, nil)
	}
}

func WithFailedCreate(
	err error,
	projectID uuid.UUID,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			CreateRepository(
				gomock.Any(),
				gomock.Any(),
				projectID,
				gomock.Any(),
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
	repositories ...*models.EntityWithProperties,
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
