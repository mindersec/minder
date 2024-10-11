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
	ghmock "github.com/stacklok/minder/internal/providers/github/mock"
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
