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

// Package fixtures contains fixtures used within the github repositories
// package. It is for testing purposes only.
//
//nolint:all
package fixtures

import (
	"errors"
	"net/http"

	"github.com/google/go-github/v60/github"
	"go.uber.org/mock/gomock"

	ghprovider "github.com/stacklok/minder/internal/providers/github"
	mockghclient "github.com/stacklok/minder/internal/repositories/github/clients/mock"
	"github.com/stacklok/minder/internal/util/ptr"
)

type (
	ClientMock        = *mockghclient.MockGitHubRepoClient
	ClientMockBuilder = func(*gomock.Controller) ClientMock
)

var (
	// ErrClientTest is a sample error used by the fixtures
	ErrClientTest = errors.New("oh no")
	ResultHook    = &github.Hook{ID: ptr.Ptr[int64](HookID)}
)

const (
	RepoOwner = "acme"
	RepoName  = "api-gateway"
	HookID    = int64(12345)
)

func NewClientMock(opts ...func(ClientMock)) ClientMockBuilder {
	return func(ctrl *gomock.Controller) ClientMock {
		mock := mockghclient.NewMockGitHubRepoClient(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func WithSuccessfulGet(repo *github.Repository) func(ClientMock) {
	return func(mock ClientMock) {
		mock.EXPECT().
			GetRepository(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(repo, nil)
	}
}

func WithFailedGet(mock ClientMock) {
	mock.EXPECT().
		GetRepository(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, ErrClientTest)
}

func WithSuccessfulDeletion(mock *mockghclient.MockGitHubRepoClient) {
	stubDelete(mock, nil, nil)
}

func WithFailedDeletion(mock ClientMock) {
	stubDelete(mock, nil, ErrClientTest)
}

func WithNotFoundDeletion(mock ClientMock) {
	githubResp := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	stubDelete(mock, githubResp, ErrClientTest)
}

func stubDelete(mock ClientMock, resp *github.Response, err error) {
	mock.EXPECT().
		DeleteHook(gomock.Any(), gomock.Eq(RepoOwner), gomock.Eq(RepoName), gomock.Eq(HookID)).
		Return(resp, err)
}

func WithSuccessfulList(url string) func(ClientMock) {
	hooks := []*github.Hook{
		{
			ID:  ptr.Ptr[int64](HookID),
			URL: &url,
		},
	}
	return func(mock ClientMock) {
		stubList(mock, hooks, nil)
	}
}

func WithFailedList(mock ClientMock) {
	stubList(mock, []*github.Hook{}, ErrClientTest)
}

func WithNotFoundList(mock ClientMock) {
	stubList(mock, []*github.Hook{}, ghprovider.ErrNotFound)
}

func stubList(mock ClientMock, hooks []*github.Hook, err error) {
	mock.EXPECT().
		ListHooks(gomock.Any(), gomock.Eq(RepoOwner), gomock.Eq(RepoName)).
		Return(hooks, err)
}

func WithFailedCreation(mock ClientMock) {
	stubCreate(mock, nil, ErrClientTest)
}

func WithSuccessfulCreation(mock ClientMock) {
	stubCreate(mock, ResultHook, nil)
}

func stubCreate(mock ClientMock, hook *github.Hook, err error) {
	// it would be nice to be able to make some assertions about the webhook
	// config which gets passed here... this requires more investigation
	mock.EXPECT().
		CreateHook(gomock.Any(), gomock.Eq(RepoOwner), gomock.Eq(RepoName), gomock.Any()).
		Return(hook, err)
}
