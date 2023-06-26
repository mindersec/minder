// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v52/github"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type GithubClientMock struct {
	mock.Mock
}

func (m *GithubClientMock) RunQuery(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	args := m.Called(ctx, query, variables)
	return args.Error(0)
}

func (m *GithubClientMock) GetGraphQLRepositoryInfo(ctx context.Context, owner string, name string) (*RepositoryInfo, error) {
	args := m.Called(ctx, owner, name)
	return args.Get(0).(*RepositoryInfo), args.Error(1)
}

func (m *GithubClientMock) GetRestAPIRepositoryInfo(ctx context.Context, owner string, name string) (*github.Repository, error) {
	args := m.Called(ctx, owner, name)
	return args.Get(0).(*github.Repository), args.Error(1)
}

func TestGetGraphQLRepositoryInfo(t *testing.T) {
	ctx := context.Background()
	client := new(GithubClientMock)
	info := &RepositoryInfo{
		Name:           "test",
		Description:    "test repo",
		StargazerCount: 100,
		ForkCount:      50,
		CreatedAt:      "2023-06-26T14:00:00Z",
	}

	client.On("GetGraphQLRepositoryInfo", ctx, "testOwner", "testRepo").Return(info, nil)

	result, err := client.GetGraphQLRepositoryInfo(ctx, "testOwner", "testRepo")
	assert.NoError(t, err)
	assert.Equal(t, info, result)
}

func TestGetRestAPIRepositoryInfo(t *testing.T) {
	ctx := context.Background()
	client := new(GithubClientMock)
	info := &github.Repository{
		Name:            github.String("test"),
		Description:     github.String("test repo"),
		StargazersCount: github.Int(100),
		ForksCount:      github.Int(50),
		CreatedAt:       &github.Timestamp{Time: time.Now()},
	}

	client.On("GetRestAPIRepositoryInfo", ctx, "testOwner", "testRepo").Return(info, nil)

	result, err := client.GetRestAPIRepositoryInfo(ctx, "testOwner", "testRepo")
	assert.NoError(t, err)
	assert.Equal(t, info, result)
}
