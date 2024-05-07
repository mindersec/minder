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

package controlplane

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/util/rand"
	"github.com/stacklok/minder/internal/util/testqueue"
)

// MockClient is a mock implementation of the GitHub client.
type MockClient struct {
	mock.Mock
}

// RunUnitTestSuite runs the unit test suite.
func RunUnitTestSuite(t *testing.T) {
	t.Helper()

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

// We should simply respond OK to ping events
func (s *UnitTestSuite) TestHandleWebHookPing() {
	t := s.T()
	t.Parallel()

	whSecretFile, err := os.CreateTemp("", "webhooksecret*")
	require.NoError(t, err, "failed to create temporary file")
	_, err = whSecretFile.WriteString("test")
	require.NoError(t, err, "failed to write to temporary file")
	defer os.Remove(whSecretFile.Name())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	srv.cfg.WebhookConfig.WebhookSecretFile = whSecretFile.Name()
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	hook := srv.HandleGitHubWebHook()
	port, err := rand.GetRandomPort()
	require.NoError(t, err, "failed to get random port")

	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.PingEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal ping event")

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "ping")
	req.Header.Add("X-GitHub-Delivery", "12345")
	// the ping event has an empty body ({}), the value below is a SHA256 hmac of the empty body with the shared key "test"
	req.Header.Add("X-Hub-Signature-256", "sha256=5f5863b9805ad4e66e954a260f9cab3f2e95718798dec0bb48a655195893d10e")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")
	assert.Len(t, queued, 0, "unexpected number of queued events")
}

// We should ignore events from repositories that are not registered
func (s *UnitTestSuite) TestHandleWebHookUnexistentRepository() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	mockStore.EXPECT().
		GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, sql.ErrNoRows)

	hook := srv.HandleGitHubWebHook()
	port, err := rand.GetRandomPort()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.MetaEvent{
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("stacklok/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	// We expect OK since we don't want to leak information about registered repositories
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")
	assert.Len(t, queued, 0)
}

func (s *UnitTestSuite) TestHandleWebHookRepository() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	prevCredsFile, err := os.CreateTemp("", "prevcreds*")
	require.NoError(t, err, "failed to create temporary file")
	_, err = prevCredsFile.WriteString("also-not-our-secret\ntest")
	require.NoError(t, err, "failed to write to temporary file")
	defer os.Remove(prevCredsFile.Name())

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	srv.cfg.WebhookConfig.WebhookSecret = "not-our-secret"
	srv.cfg.WebhookConfig.PreviousWebhookSecretFile = prevCredsFile.Name()
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	providerName := "github"
	repositoryID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()

	mockStore.EXPECT().
		GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
		Return(db.Repository{
			ID:         repositoryID,
			ProjectID:  projectID,
			RepoID:     12345,
			Provider:   providerName,
			ProviderID: providerID,
		}, nil)

	hook := srv.HandleGitHubWebHook()
	port, err := rand.GetRandomPort()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.MetaEvent{
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("stacklok/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Hub-Signature-256", "sha256=ab22bd9a3712e444e110c8088011fd827143ed63ba8655f07e76ed1a0f05edd1")
	resp, err := httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	// We expect OK since we don't want to leak information about registered repositories
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")

	received := <-queued

	assert.Equal(t, "12345", received.Metadata["id"])
	assert.Equal(t, "meta", received.Metadata["type"])
	assert.Equal(t, "https://api.github.com/", received.Metadata["source"])
	assert.Equal(t, providerID.String(), received.Metadata["provider_id"])
	assert.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
	assert.Equal(t, repositoryID.String(), received.Metadata["repository_id"])

	// TODO: assert payload is Repository protobuf

	// test that if no secret matches we get back a 400
	req, err = http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")
	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Hub-Signature-256", "sha256=ab22bd9a3712e444e110c8088011fd827143ed63ba8655f07e76ed1a0f05edd1")

	_, err = prevCredsFile.Seek(0, 0)
	require.NoError(t, err, "failed to seek to beginning of temporary file")
	_, err = prevCredsFile.WriteString("lets-just-overwrite-what-is-here-with-a-bad-secret")
	require.NoError(t, err, "failed to write to temporary file")

	resp, err = httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
}

// We should ignore events from packages from repositories that are not registered
func (s *UnitTestSuite) TestHandleWebHookUnexistentRepoPackage() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	mockStore.EXPECT().
		GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, sql.ErrNoRows)

	hook := srv.HandleGitHubWebHook()
	port, err := rand.GetRandomPort()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.PackageEvent{
		Action: github.String("published"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("stacklok/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	// We expect OK since we don't want to leak information about registered repositories
	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")
	assert.Len(t, queued, 0)
}

// We should simply respond OK
func (s *UnitTestSuite) TestNoopWebhookHandler() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	defer evt.Close()

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	hook := srv.NoopWebhookHandler()
	port, err := rand.GetRandomPort()
	require.NoError(t, err, "failed to get random port")

	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.MarketplacePurchaseEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal marketplace event")

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "marketplace_purchase")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(client, req)

	require.NoError(t, err, "failed to make request")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")
}

func (s *UnitTestSuite) TestHandleWebHookWithTooLargeRequest() {
	t := s.T()
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	srv, evt := newDefaultServer(t, mockStore, nil)
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	hook := withMaxSizeMiddleware(srv.HandleGitHubWebHook())
	port, err := rand.GetRandomPort()
	if err != nil {
		t.Fatal(err)
	}
	addr := fmt.Sprintf("localhost:%d", port)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           hook,
		ReadHeaderTimeout: 1 * time.Second,
	}
	go server.ListenAndServe()

	event := github.PackageEvent{
		Action: github.String("published"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("stacklok/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	maliciousBody := strings.NewReader(strings.Repeat("1337", 1000000000))
	maliciousBodyReader := io.MultiReader(maliciousBody, maliciousBody, maliciousBody, maliciousBody, maliciousBody)
	_ = packageJson

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s", addr), maliciousBodyReader)
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(client, req)
	require.NoError(t, err, "failed to make request")
	// We expect OK since we don't want to leak information about registered repositories
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	assert.Len(t, queued, 0)
}

func TestAll(t *testing.T) {
	t.Parallel()

	RunUnitTestSuite(t)
	// Call other test runner functions for additional test suites
}

func httpDoWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := backoff.Retry(func() error {
		var err error
		resp, err = client.Do(req)
		return err
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 3))

	return resp, err
}
