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
	_ "embed"
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

	"github.com/ThreeDotsLabs/watermill/message"
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
	"github.com/stacklok/minder/internal/providers/github/installations"
	gf "github.com/stacklok/minder/internal/providers/github/mock/fixtures"
	pf "github.com/stacklok/minder/internal/providers/manager/mock/fixtures"
	"github.com/stacklok/minder/internal/util/testqueue"
)

//go:embed test-payloads/package-published.json
var rawPackageEventPublished string

//go:embed test-payloads/push.json
var rawPushEvent string

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
		Action: github.String("deleted"),
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

	// This changes because "meta" event can only trigger a
	// deletion
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

	ts := httptest.NewServer(srv.HandleGitHubWebHook())
	defer ts.Close()

	event := github.MetaEvent{
		Action: github.String("deleted"),
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
	req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", sign(packageJson, "test")))

	resp, err := ts.Client().Do(req)
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
	req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", sign(packageJson, "test")))

	_, err = prevCredsFile.Seek(0, 0)
	require.NoError(t, err, "failed to seek to beginning of temporary file")
	_, err = prevCredsFile.WriteString("lets-just-overwrite-what-is-here-with-a-bad-secret")
	require.NoError(t, err, "failed to write to temporary file")

	resp, err = ts.Client().Do(req)
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

	// mockStore.EXPECT().
	// 	GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
	// 	Return(db.Repository{}, sql.ErrNoRows)

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

	req.Header.Add("X-GitHub-Event", "package")
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

func (s *UnitTestSuite) TestHandleGitHubWebHook() {
	t := s.T()
	t.Parallel()

	providerName := "github"
	repositoryID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	artifactID := uuid.New()
	visibility := "visibility"

	tests := []struct {
		name          string
		event         string
		payload       any
		rawPayload    []byte
		mockStoreFunc func(*gomock.Controller) *mockdb.MockStore
		statusCode    int
		topic         string
		queued        func(*testing.T, string, <-chan *message.Message)
	}{
		{
			name: "ping",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: github.Int64(54321),
				Repo: newGitHubRepo(
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
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
		},
		{
			name: "ping no hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: nil,
				Repo: newGitHubRepo(
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
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
		},
		{
			name: "package published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("published"),
				Package: &pkg{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
					Owner: &user{
						Login: github.String("login"),
					},
				},
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
				withSuccessfulGetArtifactByID(
					db.Artifact{
						ID: artifactID,
					},
				),
				withSuccessfulUpsertArtifact(
					db.Artifact{
						ID: uuid.New(),
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "package published raw payload",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload: []byte(rawPackageEventPublished),
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
				withSuccessfulGetArtifactByID(
					db.Artifact{
						ID: artifactID,
					},
				),
				withSuccessfulUpsertArtifact(
					db.Artifact{
						ID: uuid.New(),
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "package updated",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("updated"),
				Package: &pkg{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
					Owner: &user{
						Login: github.String("login"),
					},
				},
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "package no package",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &github.PackageEvent{
				Action: github.String("updated"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
		},

		// Testing package mandatory fields
		{
			name: "package mandatory repo full name",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("updated"),
				Package: &pkg{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
					Owner: &user{
						Login: github.String("login"),
					},
				},
				Repo: &repo{
					ID:       github.Int64(12345),
					FullName: nil,
					HTMLURL:  github.String("https://example.com/random/url"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "package mandatory package name",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("updated"),
				Package: &pkg{
					Name:        nil,
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
					Owner: &user{
						Login: github.String("login"),
					},
				},
				Repo: &repo{
					ID:       github.Int64(12345),
					FullName: github.String("stacklok/minder"),
					HTMLURL:  github.String("https://github.com/stacklok/minder"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "package mandatory package type",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("updated"),
				Package: &pkg{
					Name:        github.String("package-name"),
					PackageType: nil,
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
					Owner: &user{
						Login: github.String("login"),
					},
				},
				Repo: &repo{
					ID:       github.Int64(12345),
					FullName: github.String("stacklok/minder"),
					HTMLURL:  github.String("https://github.com/stacklok/minder"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "package mandatory owner",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("updated"),
				Package: &pkg{
					Name:        github.String("package-name"),
					PackageType: github.String("package-type"),
					// .package.package_version.container_metadata.tag.name
					PackageVersion: &packageVersion{
						ID:      github.Int64(1),
						Version: github.String("version"),
						ContainerMetadata: &containerMetadata{
							Tag: &tag{
								Digest: github.String("digest"),
								Name:   github.String("tag"),
							},
						},
					},
				},
				Repo: &repo{
					ID:       github.Int64(12345),
					FullName: github.String("stacklok/minder"),
					HTMLURL:  github.String("https://github.com/stacklok/minder"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "package garbage",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload:    []byte("ceci n'est pas une JSON"),
			mockStoreFunc: newMockStore(),
			statusCode:    http.StatusInternalServerError,
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
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueReconcileEntityDelete,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "meta no hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#meta
			event: "meta",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#MetaEvent
			payload: &github.MetaEvent{
				Action: github.String("deleted"),
				HookID: github.Int64(54321),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueReconcileEntityDelete,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "branch_protection_rule created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("created"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "branch_protection_rule deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("deleted"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "branch_protection_rule edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_rule
			event: "branch_protection_rule",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#BranchProtectionRuleEvent
			payload: &github.BranchProtectionRuleEvent{
				Action: github.String("edited"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "code_scanning_alert",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#code_scanning_alert
			event: "code_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#CodeScanningAlertEvent
			payload: &github.CodeScanningAlertEvent{
				Action: github.String("created"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "create",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#create
			event: "create",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#CreateEvent
			payload: &github.CreateEvent{
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "member",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#member
			event: "member",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#MemberEvent
			payload: &github.MemberEvent{
				Action: github.String("added"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "public",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#public
			event: "public",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PublicEvent
			payload: &github.PublicEvent{
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository archived",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("archived"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("created"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("deleted"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueReconcileEntityDelete,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("edited"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository privatized",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("privatized"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository publicized",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("publicized"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository renamed",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("renamed"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository transferred",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("transferred"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository unarchived",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("unarchived"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository_import",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_import
			event: "repository_import",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryImportEvent
			payload: &github.RepositoryImportEvent{
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "secret_scanning_alert created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("created"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "secret_scanning_alert reopened",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("reopened"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "secret_scanning_alert resolved",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("resolved"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "secret_scanning_alert revoked",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("revoked"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "secret_scanning_alert validated",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert
			event: "secret_scanning_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecretScanningAlertEvent
			payload: &github.SecretScanningAlertEvent{
				Action: github.String("validated"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team_add",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team_add
			event: "team_add",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamAddEvent
			payload: &github.TeamAddEvent{
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team added_to_repository",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("added_to_repository"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("created"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("deleted"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("edited"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "team removed_from_repository",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#team
			event: "team",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#TeamEvent
			payload: &github.TeamEvent{
				Action: github.String("removed_from_repository"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository_vulnerability_alert create",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("create"),
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository_vulnerability_alert dismiss",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("dismiss"),
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository_vulnerability_alert reopen",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("reopen"),
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "repository_vulnerability_alert resolve",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_vulnerability_alert
			event: "repository_vulnerability_alert",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryVulnerabilityAlertEvent
			payload: &github.RepositoryVulnerabilityAlertEvent{
				Action: github.String("resolve"),
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "security_advisory",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#security_advisory
			event: "security_advisory",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecurityAdvisoryEvent
			payload: &github.SecurityAdvisoryEvent{
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "security_and_analysis",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#security_and_analysis
			event: "security_and_analysis",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#SecurityAndAnalysisEvent
			payload: &github.SecurityAndAnalysisEvent{
				Repository: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "org_block",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#org_block
			event: "org_block",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#OrgBlockEvent
			payload:    &github.OrgBlockEvent{},
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
		},

		{
			name: "push",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
			event: "push",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PushEvent
			payload: &github.PushEvent{
				Repo: &github.PushEventRepository{
					ID:       github.Int64(12345),
					Name:     github.String("minder"),
					FullName: github.String("stacklok/minder"),
					HTMLURL:  github.String("https://github.com/stacklok/minder"),
				},
			},
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "push raw payload",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
			event: "push",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PushEvent
			rawPayload: []byte(rawPushEvent),
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},

		// The following test cases are related to events not
		// currently available in go-github.
		{
			name: "branch_protection_configuration enabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event: "branch_protection_configuration",
			payload: &repoEvent{
				Action: github.String("enabled"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "branch_protection_configuration disabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event: "branch_protection_configuration",
			payload: &repoEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "repository_advisory published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repoEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "repository_advisory reported",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repoEvent{
				Action: github.String("reported"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "repository_ruleset created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "repository_ruleset deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "repository_ruleset edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("edited"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "secret_scanning_alert_location",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "secret_scanning_alert_location",
			payload: &repoEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusOK,
			queued:        nil,
		},

		// package/artifact specific tests
		{
			name: "pull_request opened",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("opened"),
				Repo: newGitHubRepo(
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
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
				withSuccessfulUpsertPullRequest(
					db.PullRequest{},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, providerID.String(), received.Metadata["provider_id"])
				require.Equal(t, projectID.String(), received.Metadata[entities.ProjectIDEventKey])
				require.Equal(t, repositoryID.String(), received.Metadata["repository_id"])
			},
		},
		{
			name: "pull_request closed",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("closed"),
				Repo: newGitHubRepo(
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
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
				withSuccessfulDeletePullRequest(),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
		},
		{
			name: "pull_request not handled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("random"),
				Repo: newGitHubRepo(
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
			mockStoreFunc: newMockStore(
				withSuccessfulGetRepositoryByRepoID(
					db.Repository{
						ID:         repositoryID,
						ProjectID:  projectID,
						RepoID:     12345,
						Provider:   providerName,
						ProviderID: providerID,
					},
				),
			),
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         events.TopicQueueEntityEvaluate,
			statusCode:    http.StatusInternalServerError,
			queued:        nil,
		},

		// garbage
		{
			name:  "garbage",
			event: "repository",
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
		},
		{
			name:  "total garbage",
			event: "garbage",
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      events.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			prevCredsFile, err := os.CreateTemp("", "prevcreds*")
			require.NoError(t, err, "failed to create temporary file")
			_, err = prevCredsFile.WriteString("also-not-our-secret\ntest")
			require.NoError(t, err, "failed to write to temporary file")
			defer os.Remove(prevCredsFile.Name())

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
					CreatedAt: &github.Timestamp{},
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

			var mockStore *mockdb.MockStore
			if tt.mockStoreFunc != nil {
				mockStore = tt.mockStoreFunc(ctrl)
			} else {
				mockStore = mockdb.NewMockStore(ctrl)
			}

			srv, evt := newDefaultServer(t, mockStore, nil)
			srv.cfg.WebhookConfig.WebhookSecret = "not-our-secret"
			srv.cfg.WebhookConfig.PreviousWebhookSecretFile = prevCredsFile.Name()
			srv.providerManager = providerSetup(ctrl)
			defer evt.Close()

			pq := testqueue.NewPassthroughQueue(t)
			queued := pq.GetQueue()

			evt.Register(tt.topic, pq.Pass)

			go func() {
				err := evt.Run(context.Background())
				require.NoError(t, err, "failed to run eventer")
			}()

			<-evt.Running()

			ts := httptest.NewServer(http.HandlerFunc(srv.HandleGitHubWebHook()))
			defer ts.Close()

			var packageJson []byte
			if tt.payload != nil {
				packageJson, err = json.Marshal(tt.payload)
				require.NoError(t, err, "failed to marshal package event")
			} else {
				packageJson = tt.rawPayload
			}

			expectedMAC := sign(packageJson, "test")

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
			require.Equal(t, tt.statusCode, resp.StatusCode, "unexpected status code")

			require.Len(t, queued, 0)
			if tt.queued != nil {
				tt.queued(t, tt.event, queued)
			}

			// TODO: assert payload is Repository protobuf
		})
	}
}

func (s *UnitTestSuite) TestHandleGitHubAppWebHook() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name          string
		event         string
		payload       any
		rawPayload    []byte
		mockStoreFunc func(*gomock.Controller) *mockdb.MockStore
		statusCode    int
		topic         string
		queued        func(*testing.T, string, <-chan *message.Message)
	}{
		{
			name: "ping",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: github.Int64(54321),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
		},
		{
			name: "ping no hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#ping
			event: "ping",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PingEvent
			payload: &github.PingEvent{
				HookID: nil,
				Repo: newGitHubRepo(
					12345,
					"minder",
					"stacklok/minder",
					"https://github.com/stacklok/minder",
				),
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://example.com/random/url"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
		},
		{
			name: "installation created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationEvent{
				Action: github.String("created"),
				Repositories: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"stacklok/minder",
						"https://github.com/stacklok/minder",
					),
				},
				// Installation field is left blank on
				// purpose, to attest the fact that
				// this particolar event/action
				// combination does not use it.
				Installation: &github.Installation{},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "installation deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationEvent{
				Action: github.String("deleted"),
				Repositories: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"stacklok/minder",
						"https://github.com/stacklok/minder",
					),
				},
				Installation: &github.Installation{
					ID: github.Int64(12345),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			//nolint:thelper
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
			},
		},
		{
			name: "installation new_permissions_accepted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationEvent{
				Action: github.String("new_permissions_accepted"),
				Repositories: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"stacklok/minder",
						"https://github.com/stacklok/minder",
					),
				},
				// Installation field is left blank on
				// purpose, to attest the fact that
				// this particolar event/action
				// combination does not use it.
				Installation: &github.Installation{},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "installation suspend",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationEvent{
				Action: github.String("suspend"),
				Repositories: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"stacklok/minder",
						"https://github.com/stacklok/minder",
					),
				},
				// Installation field is left blank on
				// purpose, to attest the fact that
				// this particolar event/action
				// combination does not use it.
				Installation: &github.Installation{},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued:        nil,
		},
		{
			name: "installation unsuspend",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationEvent{
				Action: github.String("unsuspend"),
				Repositories: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"stacklok/minder",
						"https://github.com/stacklok/minder",
					),
				},
				// Installation field is left blank on
				// purpose, to attest the fact that
				// this particolar event/action
				// combination does not use it.
				Installation: &github.Installation{},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: newMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued:        nil,
		},

		// garbage
		{
			name:  "garbage",
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      installations.ProviderInstallationTopic,
			statusCode: http.StatusOK,
		},
		{
			name:  "total garbage",
			event: "garbage",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      installations.ProviderInstallationTopic,
			statusCode: http.StatusOK,
		},
		{
			name: "more garbage",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload: []byte("ceci n'est pas une JSON"),
			topic:      installations.ProviderInstallationTopic,
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			srv, evt := newDefaultServer(t, mockStore, nil)
			srv.cfg.WebhookConfig.WebhookSecret = "test"

			pq := testqueue.NewPassthroughQueue(t)
			queued := pq.GetQueue()

			evt.Register(tt.topic, pq.Pass)

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
				}, nil).
				AnyTimes()

			ts := httptest.NewServer(http.HandlerFunc(srv.HandleGitHubAppWebhook()))
			defer ts.Close()

			var packageJson []byte
			if tt.payload != nil {
				var err error
				packageJson, err = json.Marshal(tt.payload)
				require.NoError(t, err, "failed to marshal package event")
			} else {
				packageJson = tt.rawPayload
			}

			expectedMAC := sign(packageJson, "test")
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
			require.Equal(t, tt.statusCode, resp.StatusCode, "unexpected status code")

			require.Len(t, queued, 0)
			if tt.queued != nil {
				tt.queued(t, tt.event, queued)
			}

			// TODO: assert payload is Repository protobuf
		})
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	RunUnitTestSuite(t)
	// Call other test runner functions for additional test suites
}

//nolint:unparam
func withTimeout(ch <-chan *message.Message, timeout time.Duration) *message.Message {
	wrapper := make(chan *message.Message, 1)
	go func() {
		select {
		case item := <-ch:
			wrapper <- item
		case <-time.After(timeout):
			wrapper <- nil
		}
	}()
	return <-wrapper
}

//nolint:unparam
func newGitHubRepo(id int, name, fullname, url string) *github.Repository {
	return &github.Repository{
		ID:       github.Int64(int64(id)),
		Name:     github.String(name),
		FullName: github.String(fullname),
		HTMLURL:  github.String(url),
	}
}

//nolint:unparam
func newRepo(id int, fullname, url string) *repo {
	return &repo{
		ID:       github.Int64(int64(id)),
		FullName: github.String(fullname),
		HTMLURL:  github.String(url),
	}
}

//nolint:unparam
func sign(payload []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
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

func withSuccessfulGetRepositoryByRepoID(repository db.Repository) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetRepositoryByRepoID(gomock.Any(), gomock.Any()).
			Return(repository, nil)
	}
}

func withSuccessfulUpsertPullRequest(pullRequest db.PullRequest) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			UpsertPullRequest(gomock.Any(), gomock.Any()).
			Return(pullRequest, nil)
	}
}

func withSuccessfulDeletePullRequest() func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			DeletePullRequest(gomock.Any(), gomock.Any()).
			Return(nil)
	}
}

func withSuccessfulGetArtifactByID(artifact db.Artifact) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetArtifactByID(gomock.Any(), gomock.Any()).
			Return(artifact, nil)
	}
}

func withSuccessfulUpsertArtifact(artifact db.Artifact) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			UpsertArtifact(gomock.Any(), gomock.Any()).
			Return(artifact, nil)
	}
}

func newMockStore(funcs ...func(*mockdb.MockStore)) func(*gomock.Controller) *mockdb.MockStore {
	return func(ctrl *gomock.Controller) *mockdb.MockStore {
		mockStore := mockdb.NewMockStore(ctrl)

		for _, fn := range funcs {
			fn(mockStore)
		}

		return mockStore
	}
}
