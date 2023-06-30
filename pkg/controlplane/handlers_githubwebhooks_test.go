//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.
package controlplane

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"encoding/json"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v53/github"
	"github.com/stacklok/mediator/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

// MockClient is a mock implementation of the GitHub client.
type MockClient struct {
	mock.Mock
}

// RunUnitTestSuite runs the unit test suite.
func RunUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

// Repositories is a mock implementation of the GitHub client's Repositories service.
func (m *MockClient) Repositories() *github.RepositoriesService {
	args := m.Called()
	return args.Get(0).(*github.RepositoriesService)
}

// CreateHook is a mock implementation of the GitHub client's CreateHook method.
func (m *MockClient) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	args := m.Called(ctx, owner, repo, hook)
	return args.Get(0).(*github.Hook), args.Get(1).(*github.Response), args.Error(2)
}

// UnitTestSuite is the test suite for the unit tests.
type UnitTestSuite struct {
	suite.Suite
	mockClient *MockClient
}

// SetupTest is called before each test function is executed.
func (s *UnitTestSuite) SetupTest() {
	s.mockClient = new(MockClient)
}

// TestRegisterWebHook_Success tests the RegisterWebHook function when the webhook registration is successful.
func (s *UnitTestSuite) TestRegisterWebHook_Success() {
	// Set up the test data
	ctx := context.Background()
	token := oauth2.Token{AccessToken: "your-access-token"}
	repositories := []Repository{
		{Owner: "owner1", Repo: "repo1"},
	}
	events := []string{"push", "pull_request"}

	// Set up the expectations for the mock client
	s.mockClient.On("Repositories").Return(&github.RepositoriesService{})
	s.mockClient.On("CreateHook", ctx, "owner1", "repo1", mock.AnythingOfType("*github.Hook")).
		Return(&github.Hook{ID: github.Int64(0), CreatedAt: &github.Timestamp{Time: time.Now()}}, &github.Response{}, nil)

	// Inject the mock client into the RegisterWebHook function
	registerData, err := RegisterWebHook(ctx, token, repositories, events)
	require.NoError(s.T(), err)
	require.Len(s.T(), registerData, 1)

	// Assertions for the first result
	assert.Equal(s.T(), "repo1", registerData[0].Repository)
	assert.Equal(s.T(), "owner1", registerData[0].Owner)
	assert.Equal(s.T(), int64(0), registerData[0].HookID)
}

func (s *UnitTestSuite) TestHandleGitHubWebHook() {
	ctx := context.Background()
	token := oauth2.Token{AccessToken: "your-access-token"}
	repositories := []Repository{
		{Owner: "owner1", Repo: "repo1"},
		{Owner: "owner2", Repo: "repo2"},
	}

	events := []string{"push", "pull_request"}

	// Set up the expectations for the mock client
	s.mockClient.On("Repositories").Return(&github.RepositoriesService{})
	s.mockClient.On("CreateHook", ctx, "owner1", "repo1", mock.AnythingOfType("*github.Hook")).
		Return(&github.Hook{ID: github.Int64(0), CreatedAt: &github.Timestamp{Time: time.Now()}}, &github.Response{}, nil)
	// Call the function under test
	results, err := RegisterWebHook(ctx, token, repositories, events)

	// Assertions
	require.NoError(s.T(), err)
	require.Len(s.T(), results, 2)
	assert.Equal(s.T(), "repo1", results[0].Repository)
	assert.Equal(s.T(), "owner1", results[0].Owner)
	assert.Equal(s.T(), int64(0), results[0].HookID)
	assert.NotEmpty(s.T(), results[0].DeployURL)
}

func TestHandleWebHook(t *testing.T) {
	p := gochannel.NewGoChannel(gochannel.Config{}, nil)
	queued, err := p.Subscribe(context.Background(), "package")
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	hook := HandleGitHubWebHook(p)
	port, err := util.GetRandomPort()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: hook,
	}
	go server.ListenAndServe()

	event := github.PackageEvent{
		Action: github.String("published"),
		Package: &github.Package{
			Name:        github.String("mediator"),
			PackageType: github.String("container"),
		},
		Repo: &github.Repository{
			Name: github.String("stacklok/mediator"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("X-GitHub-Event", "package")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	received := <-queued

	if diff := cmp.Diff(string(packageJson), string(received.Payload)); diff != "" {
		t.Fatalf("payload mismatch (-want +got):\n%s", diff)
	}
	assert.Equal(t, "12345", received.Metadata["id"])
	assert.Equal(t, "package", received.Metadata["type"])
	assert.Equal(t, "https://api.github.com/", received.Metadata["source"])

	p.Close()
}

func TestAll(t *testing.T) {
	RunUnitTestSuite(t)
	// Call other test runner functions for additional test suites
}
