// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating ProfileService fixtures and is used in
// various parts of the code. For testing use only.
//
//nolint:all
package fixtures

import (
	ghmock "github.com/mindersec/minder/internal/providers/github/mock"
	"go.uber.org/mock/gomock"
)

type (
	GitHubMock        = *ghmock.MockGitHub
	GitHubMockBuilder = func(*gomock.Controller) GitHubMock
)

func NewGitHubMock(opts ...func(mock GitHubMock)) GitHubMockBuilder {
	return func(ctrl *gomock.Controller) GitHubMock {
		mock := ghmock.NewMockGitHub(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func WithSuccessfulGetPackageByName(artifact any) func(mock GitHubMock) {
	return func(mock GitHubMock) {
		mock.EXPECT().
			GetPackageByName(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			AnyTimes().
			Return(artifact, nil)
	}
}

func WithSuccessfulGetPackageVersionById(version any) func(mock GitHubMock) {
	return func(mock GitHubMock) {
		mock.EXPECT().
			GetPackageVersionById(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			AnyTimes().
			Return(version, nil)
	}
}

func WithSuccessfulGetPullRequest(pr any) func(mock GitHubMock) {
	return func(mock GitHubMock) {
		mock.EXPECT().
			GetPullRequest(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).
			AnyTimes().
			Return(pr, nil)
	}
}

func WithSuccessfulGetEntityName(name string) func(mock GitHubMock) {
	return func(mock GitHubMock) {
		mock.EXPECT().
			GetEntityName(
				gomock.Any(),
				gomock.Any(),
			).
			AnyTimes().
			Return(name, nil)
	}
}
