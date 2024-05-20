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
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	gf "github.com/stacklok/minder/internal/providers/github/mock/fixtures"
	pf "github.com/stacklok/minder/internal/providers/manager/mock/fixtures"
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

	ts := httptest.NewServer(srv.HandleGitHubWebHook())
	defer ts.Close()

	event := github.PingEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal ping event")

	req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "ping")
	req.Header.Add("X-GitHub-Delivery", "12345")
	// the ping event has an empty body ({}), the value below is a SHA256 hmac of the empty body with the shared key "test"
	req.Header.Add("X-Hub-Signature-256", "sha256=5f5863b9805ad4e66e954a260f9cab3f2e95718798dec0bb48a655195893d10e")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)
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

	ts := httptest.NewServer(srv.HandleGitHubWebHook())
	defer ts.Close()

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

	req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)
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

	ts := httptest.NewServer(srv.HandleGitHubWebHook())
	defer ts.Close()

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

	req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Hub-Signature-256", "sha256=ab22bd9a3712e444e110c8088011fd827143ed63ba8655f07e76ed1a0f05edd1")
	resp, err := httpDoWithRetry(ts.Client(), req)
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
	req, err = http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")
	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Hub-Signature-256", "sha256=ab22bd9a3712e444e110c8088011fd827143ed63ba8655f07e76ed1a0f05edd1")

	_, err = prevCredsFile.Seek(0, 0)
	require.NoError(t, err, "failed to seek to beginning of temporary file")
	_, err = prevCredsFile.WriteString("lets-just-overwrite-what-is-here-with-a-bad-secret")
	require.NoError(t, err, "failed to write to temporary file")

	resp, err = httpDoWithRetry(ts.Client(), req)
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

	ts := httptest.NewServer(srv.HandleGitHubWebHook())

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

	req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)
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

	ts := httptest.NewServer(srv.NoopWebhookHandler())
	defer ts.Close()

	event := github.MarketplacePurchaseEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal marketplace event")

	req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "marketplace_purchase")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)

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

	ts := httptest.NewServer(withMaxSizeMiddleware(srv.HandleGitHubWebHook()))

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

	req, err := http.NewRequest("POST", ts.URL, maliciousBodyReader)
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)
	require.NoError(t, err, "failed to make request")
	// We expect OK since we don't want to leak information about registered repositories
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "unexpected status code")
	assert.Len(t, queued, 0)
}

func (s *UnitTestSuite) TestHandlGitHubWebHook() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name    string
		event   string
		payload any
		queued  bool
	}{
		{
			name: "ping",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: github.Int64(54321),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
		},
		{
			name: "ping no hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: nil,
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://example.com/random/url"),
				},
			},
		},
		{
			name: "package published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &github.PackageEvent{
				Action: github.String("published"),
				Package: &github.Package{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &github.PackageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						TagName: github.String("tagname"),
					},
					Owner: &github.User{
						Login: github.String("login"),
					},
				},
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "package updated",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &github.PackageEvent{
				Action: github.String("updated"),
				Package: &github.Package{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					PackageVersion: &github.PackageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						TagName: github.String("tagname"),
					},
					Owner: &github.User{
						Login: github.String("login"),
					},
				},
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
		},
		{
			name: "package no package",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &github.PackageEvent{
				Action: github.String("updated"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
		},
		{
			name: "meta",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#meta
			event: "meta",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#MetaEvent
			payload: &github.MetaEvent{
				Action: github.String("deleted"),
				HookID: github.Int64(54321),
				Hook: &github.Hook{
					ID: github.Int64(54321),
				},
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "meta no hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#meta
			event: "meta",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#MetaEvent
			payload: &github.MetaEvent{
				Action: github.String("deleted"),
				HookID: github.Int64(54321),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "branch_protection_rule created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "branch_protection_rule deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "branch_protection_rule edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("edited"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "code_scanning_alert",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#code_scanning_alert
			event: "code_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#CodeScanningAlertEvent
			payload: &github.CodeScanningAlertEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "create",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#create
			event: "create",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#CreateEvent
			payload: &github.CreateEvent{
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "member",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#member
			event: "member",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#MemberEvent
			payload: &github.MemberEvent{
				Action: github.String("added"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "public",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#public
			event: "public",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PublicEvent
			payload: &github.PublicEvent{
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: true,
		},
		{
			name: "repository archived",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("archived"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("edited"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository privatized",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("privatized"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository publicized",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("publicized"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository renamed",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("renamed"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository transferred",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("transferred"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository unarchived",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("unarchived"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository_import",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_import
			event: "repository_import",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryImportEvent
			payload: &github.RepositoryImportEvent{
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "secret_scanning_alert created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "secret_scanning_alert reopened",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("reopened"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "secret_scanning_alert resolved",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("resolved"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "secret_scanning_alert revoked",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("revoked"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "secret_scanning_alert validated",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("validated"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team_add",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team_add
			event: "team_add",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamAddEvent
			payload: &github.TeamAddEvent{
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team added_to_repository",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("added_to_repository"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("edited"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "team removed_from_repository",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("removed_from_repository"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository_vulnerability_alert create",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("create"),
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository_vulnerability_alert dismiss",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("dismiss"),
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository_vulnerability_alert reopen",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("reopen"),
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "repository_vulnerability_alert resolve",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("resolve"),
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "security_advisory",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#security_advisory
			event: "security_advisory",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecurityAdvisoryEvent
			payload: &github.SecurityAdvisoryEvent{
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "security_and_analysis",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#security_and_analysis
			event: "security_and_analysis",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecurityAndAnalysisEvent
			payload: &github.SecurityAndAnalysisEvent{
				Repository: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			queued: true,
		},
		{
			name: "org_block",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#org_block
			event: "org_block",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#OrgBlockEvent
			payload: &github.OrgBlockEvent{},
			queued:  false,
		},
		{
			name: "push",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
			event: "push",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PushEvent
			payload: &github.PushEvent{},
		},

		// The following test cases are related to events not
		// currently available in go-github.
		{
			name: "branch_protection_configuration enabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event: "branch_protection_configuration",
			payload: &branchProtectionConfigurationEvent{
				Action: github.String("enabled"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "branch_protection_configuration disabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event: "branch_protection_configuration",
			payload: &branchProtectionConfigurationEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "repository_advisory published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repositoryAdvisoryEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "repository_advisory reported",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repositoryAdvisoryEvent{
				Action: github.String("reported"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "repository_ruleset created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repositoryRulesetEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "repository_ruleset deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repositoryRulesetEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "repository_ruleset created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repositoryRulesetEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},
		{
			name: "secret_scanning_alert_location",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "secret_scanning_alert_location",
			payload: &secretScanningAlertLocationEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},

		// package/artifact specific tests
		{
			name: "pull_request opened",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("opened"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			queued: true,
		},
		{
			name: "pull_request closed",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("closed"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			queued: false,
		},
		{
			name: "pull_request not handled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("random"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			queued: false,
		},
		{
			name: "pull_request no details",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				// There are many possible actions for
				// PR events, but we don't really
				// care.
				Action: github.String("whatever"),
				Repo: newRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			queued: false,
		},

		// garbage
		{
			name:  "garbage",
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
		},
		{
			name:  "total garbage",
			event: "garbage",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			prevCredsFile, err := os.CreateTemp("", "prevcreds*")
			require.NoError(t, err, "failed to create temporary file")
			_, err = prevCredsFile.WriteString("also-not-our-secret\ntest")
			require.NoError(t, err, "failed to write to temporary file")
			defer os.Remove(prevCredsFile.Name())

			visibility := "visibility"
			ghProvider := gf.NewGitHubMock(
				gf.WithSuccessfulGetPackageByName(&github.Package{
					Visibility: &visibility,
				}),
				gf.WithSuccessfulGetPackageVersionById(&github.PackageVersion{
					Metadata: &github.PackageMetadata{
						Container: &github.PackageContainerMetadata{
							Tags: []string{"tag"},
						},
					},
				}),
				gf.WithSuccessfulGetPullRequest(&github.PullRequest{
					Head: &github.PullRequestBranch{
						SHA: github.String("sha"),
					},
				}),
			)
			providerSetup := pf.NewProviderManagerMock(
				pf.WithSuccessfulInstantiateFromID(ghProvider(ctrl)),
			)

			mockStore := mockdb.NewMockStore(ctrl)
			srv, evt := newDefaultServer(t, mockStore, nil)
			srv.cfg.WebhookConfig.WebhookSecret = "not-our-secret"
			srv.cfg.WebhookConfig.PreviousWebhookSecretFile = prevCredsFile.Name()
			srv.providerManager = providerSetup(ctrl)
			defer evt.Close()

			pq := testqueue.NewPassthroughQueue(t)
			queued := pq.GetQueue()

			evt.Register(events.TopicQueueEntityEvaluate, pq.Pass)
			evt.Register(events.TopicQueueReconcileEntityDelete, pq.Pass)

			go func() {
				err := evt.Run(context.Background())
				require.NoError(t, err, "failed to run eventer")
			}()

			<-evt.Running()

			providerName := "github"
			repositoryID := uuid.New()
			projectID := uuid.New()
			providerID := uuid.New()
			artifactID := uuid.New()

			mockStore.EXPECT().
				GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
				Return(db.Repository{
					ID:         repositoryID,
					ProjectID:  projectID,
					RepoID:     12345,
					Provider:   providerName,
					ProviderID: providerID,
				}, nil).
				AnyTimes()
			mockStore.EXPECT().
				UpsertArtifact(gomock.Any(), gomock.Any()).
				Return(db.Artifact{
					ID: uuid.New(),
				}, nil).
				AnyTimes()
			mockStore.EXPECT().
				GetArtifactByID(gomock.Any(), gomock.Any()).
				Return(db.Artifact{
					ID: artifactID,
				}, nil).
				AnyTimes()
			mockStore.EXPECT().
				GetArtifactByID(gomock.Any(), gomock.Any()).
				Return(db.Artifact{
					ID: artifactID,
					RepositoryID: uuid.NullUUID{
						UUID:  repositoryID,
						Valid: true,
					},
					ProviderName:       providerName,
					ArtifactName:       "name",
					ArtifactType:       "type",
					ArtifactVisibility: visibility,
				}, nil).
				AnyTimes()
			mockStore.EXPECT().
				UpsertPullRequest(gomock.Any(), gomock.Any()).
				Return(db.PullRequest{}, nil).
				AnyTimes()
			mockStore.EXPECT().
				DeletePullRequest(gomock.Any(), gomock.Any()).
				Return(nil).
				AnyTimes()

			ts := httptest.NewServer(http.HandlerFunc(srv.HandleGitHubWebHook()))
			defer ts.Close()

			packageJson, err := json.Marshal(tt.payload)
			require.NoError(t, err, "failed to marshal package event")

			mac := hmac.New(sha256.New, []byte("test"))
			mac.Write(packageJson)
			expectedMAC := hex.EncodeToString(mac.Sum(nil))

			client := &http.Client{}
			req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
			require.NoError(t, err, "failed to create request")

			req.Header.Add("X-GitHub-Event", tt.event)
			req.Header.Add("X-GitHub-Delivery", "12345")
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", expectedMAC))
			resp, err := httpDoWithRetry(client, req)
			require.NoError(t, err, "failed to make request")
			// We expect OK since we don't want to leak information about registered repositories
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")

			require.Len(t, queued, 0)
			if tt.queued {
				received := <-queued

				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, tt.event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			}

			// TODO: assert payload is Repository protobuf
		})
	}
}

func newRepo(id int, name, fullname, url string) *github.Repository {
	return &github.Repository{
		ID:       github.Int64(12345),
		Name:     github.String(name),
		FullName: github.String(fullname),
		HTMLURL:  github.String(url),
	}
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

type garbage struct {
	Action  *string `json:"action,omitempty"`
	Garbage *string `json:"garbage,omitempty"`
}

type branchProtectionConfigurationEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type repositoryAdvisoryEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type repositoryRulesetEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type secretScanningAlertLocationEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}
