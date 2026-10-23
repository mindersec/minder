// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"context"

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
				gomock.Any(),
				gomock.Any(),
			).
			DoAndReturn(func(_ context.Context, _, _, _ uuid.UUID, limit int) ([]*models.EntityWithProperties, uuid.UUID, error) {
				// Return the last entity ID as cursor when a full page is returned,
				// matching real service behaviour used by the handler for next-page detection.
				var nextCursor uuid.UUID
				if len(repositories) == limit {
					nextCursor = repositories[len(repositories)-1].Entity.ID
				}
				return repositories, nextCursor, nil
			}).AnyTimes()
	}
}

func WithFailedListRepositories(err error) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		mock.EXPECT().
			ListRepositories(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			Return(nil, uuid.Nil, err).AnyTimes()
	}
}

// WithSuccessfulListRepositoriesInPages sets up the mock to return different
// pages on successive calls, simulating cursor-based pagination.
// Each call to ListRepositories returns the next page in the sequence.
func WithSuccessfulListRepositoriesInPages(
	pages [][]*models.EntityWithProperties,
) func(RepoServiceMock) {
	return func(mock RepoServiceMock) {
		for _, page := range pages {
			p := page // capture loop variable
			mock.EXPECT().
				ListRepositories(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).
				Return(p, uuid.Nil, nil).
				Times(1)
		}
	}
}
