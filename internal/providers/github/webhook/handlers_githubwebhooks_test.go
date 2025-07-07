// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	df "github.com/mindersec/minder/database/mock/fixtures"
	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	entMsg "github.com/mindersec/minder/internal/entities/handlers/message"
	mock_service "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/providers/github/installations"
	gf "github.com/mindersec/minder/internal/providers/github/mock/fixtures"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	ghService "github.com/mindersec/minder/internal/providers/github/service"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	"github.com/mindersec/minder/internal/util/testqueue"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

//go:embed test-payloads/installation-deleted.json
var rawInstallationDeletedEvent string

//go:embed test-payloads/package-published.json
var rawPackageEventPublished string

//go:embed test-payloads/push.json
var rawPushEvent string

//go:embed test-payloads/branch-protection-configuration-disabled.json
var rawBranchProtectionConfigurationDisabledEvent string

var timeout time.Duration = 10 * time.Millisecond

// MockClient is a mock implementation of the GitHub client.
type MockClient struct {
	mock.Mock
}

type propSvcMock = *mock_service.MockPropertiesService
type propSvcMockBuilder = func(*gomock.Controller) propSvcMock

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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")
	defer evt.Close()

	whSecretFile, err := os.CreateTemp("", "webhooksecret*")
	require.NoError(t, err, "failed to create temporary file")
	_, err = whSecretFile.WriteString("test")
	require.NoError(t, err, "failed to write to temporary file")
	defer os.Remove(whSecretFile.Name())

	cfg := &serverconfig.WebhookConfig{}
	cfg.WebhookSecretFile = whSecretFile.Name()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(constants.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	handler := HandleWebhookEvent(metrics.NewNoopMetrics(), evt, cfg)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	event := github.PingEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal ping event")

	resp, err := httpDoWithRetry(ts.Client(), func() (*http.Request, error) {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
		if err != nil {
			return nil, err
		}

		req.Header.Add("X-GitHub-Event", "ping")
		req.Header.Add("X-GitHub-Delivery", "12345")
		// the ping event has an empty body ({}), the value below is a SHA256 hmac of the empty body with the shared key "test"
		req.Header.Add("X-Hub-Signature-256", "sha256=5f5863b9805ad4e66e954a260f9cab3f2e95718798dec0bb48a655195893d10e")
		req.Header.Add("Content-Type", "application/json")
		return req, nil
	})
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

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	defer pq.Close()
	queued := pq.GetQueue()

	evt.Register(constants.TopicQueueRefreshEntityAndEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	cfg := &serverconfig.WebhookConfig{}
	handler := HandleWebhookEvent(metrics.NewNoopMetrics(), evt, cfg)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	event := github.MetaEvent{
		Action: github.String("deleted"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("mindersec/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	resp, err := httpDoWithRetry(ts.Client(), func() (*http.Request, error) {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
		if err != nil {
			return nil, err
		}

		req.Header.Add("X-GitHub-Event", "meta")
		req.Header.Add("X-GitHub-Delivery", "12345")
		req.Header.Add("Content-Type", "application/json")
		return req, nil
	})
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

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	// This changes because "meta" event can only trigger a
	// deletion

	evt.Register(constants.TopicQueueGetEntityAndDelete, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	prevCredsFile, err := os.CreateTemp("", "prevcreds*")
	require.NoError(t, err, "failed to create temporary file")
	_, err = prevCredsFile.WriteString("also-not-our-secret\ntest")
	require.NoError(t, err, "failed to write to temporary file")
	defer os.Remove(prevCredsFile.Name())

	cfg := &serverconfig.WebhookConfig{}
	cfg.WebhookSecret = "not-our-secret"
	cfg.PreviousWebhookSecretFile = prevCredsFile.Name()

	handler := HandleWebhookEvent(metrics.NewNoopMetrics(), evt, cfg)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	event := github.MetaEvent{
		Action: github.String("deleted"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("mindersec/minder"),
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
	var inner entMsg.HandleEntityAndDoMessage
	err = json.Unmarshal(received.Payload, &inner)
	require.NoError(t, err)
	require.NoError(t, validator.New().Struct(&inner))

	require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, inner.Entity.Type)
	require.Equal(t, "12345", inner.Entity.GetByProps[properties.PropertyUpstreamID])
	require.Equal(t, "github", inner.Hint.ProviderImplementsHint)

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

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")
	defer evt.Close()

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt.Register(constants.TopicQueueEntityEvaluate, pq.Pass)

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	cfg := &serverconfig.WebhookConfig{}

	handler := HandleWebhookEvent(metrics.NewNoopMetrics(), evt, cfg)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	event := github.PackageEvent{
		Action: github.String("published"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("mindersec/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	resp, err := httpDoWithRetry(ts.Client(), func() (*http.Request, error) {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
		if err != nil {
			return nil, err
		}

		req.Header.Add("X-GitHub-Event", "package")
		req.Header.Add("X-GitHub-Delivery", "12345")
		req.Header.Add("Content-Type", "application/json")
		return req, nil
	})
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

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")
	defer evt.Close()

	go func() {
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	handler := NoopWebhookHandler(metrics.NewNoopMetrics())
	ts := httptest.NewServer(handler)
	defer ts.Close()

	event := github.MarketplacePurchaseEvent{}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal marketplace event")

	resp, err := httpDoWithRetry(ts.Client(), func() (*http.Request, error) {
		req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
		if err != nil {
			return nil, err
		}

		req.Header.Add("X-GitHub-Event", "marketplace_purchase")
		req.Header.Add("X-GitHub-Delivery", "12345")
		req.Header.Add("Content-Type", "application/json")
		return req, nil
	})

	require.NoError(t, err, "failed to make request")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code")
}

func (s *UnitTestSuite) TestHandleGitHubWebHook() {
	t := s.T()
	t.Parallel()

	tests := []struct {
		name         string
		event        string
		payload      any
		rawPayload   []byte
		mockPropsBld propSvcMockBuilder
		ghMocks      []func(hubMock gf.GitHubMock)
		statusCode   int
		topic        string
		queued       func(*testing.T, string, <-chan *message.Message)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Org: &github.Organization{
					Login: github.String("stacklok"),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://example.com/random/url"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
		},
		{
			name: "package published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			payload: &packageEvent{
				Action: github.String("published"),
				Package: &pkg{
					ID:          github.Int64(123),
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("login/package-name"),
			},
			topic:      constants.TopicQueueOriginatingEntityAdd,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "package published raw payload",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload: []byte(rawPackageEventPublished),
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder"),
			},
			topic:      constants.TopicQueueOriginatingEntityAdd,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
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
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
		},
		{
			name: "package garbage",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload: []byte("ceci n'est pas une JSON"),
			statusCode: http.StatusInternalServerError,
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
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
			queued:     nil,
		},
		{
			name: "package garbage",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#package
			event: "package",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PackageEvent
			rawPayload: []byte("ceci n'est pas une JSON"),
			statusCode: http.StatusInternalServerError,
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)

				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
				matchProps := properties.NewProperties(evt.MatchProps)
				require.Equal(t, int64(54321), matchProps.GetProperty(ghprop.RepoPropertyHookId).GetInt64())

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)

				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
				matchProps := properties.NewProperties(evt.MatchProps)
				require.Equal(t, int64(54321), matchProps.GetProperty(ghprop.RepoPropertyHookId).GetInt64())

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "meta bad hook",
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)

				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
				matchProps := properties.NewProperties(evt.MatchProps)
				require.Equal(t, int64(54321), matchProps.GetProperty(ghprop.RepoPropertyHookId).GetInt64())

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, event, received.Metadata["type"])

				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)

				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository deleted had hook",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("deleted"),
				Repo: &github.Repository{
					ID:       github.Int64(12345),
					Name:     github.String("minder"),
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)

				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)

				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)

				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)

				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, _ string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository private repos not enabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("transferred"),
				Repo: &github.Repository{
					ID:       github.Int64(12345),
					Name:     github.String("minder"),
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
					Private:  github.Bool(true),
				},
			},
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			// the message is passed on to events.TopicQueueRefreshEntityAndEvaluate
			// which should discard it (see test there)
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				timeout := 1 * time.Second
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
			},
		},
		{
			name: "repository private repos enabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository
			event: "repository",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#RepositoryEvent
			payload: &github.RepositoryEvent{
				Action: github.String("created"),
				Repo: &github.Repository{
					ID:       github.Int64(12345),
					Name:     github.String("minder"),
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
					Private:  github.Bool(true),
				},
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			// the message is passed on to events.TopicQueueRefreshEntityAndEvaluate
			// which should discard it (see test there)
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.NotNilf(t, received, "no event received after waiting %s", timeout)

				var evt entMsg.HandleEntityAndDoMessage
				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				require.Equal(t, "12345", evt.Entity.GetByProps[properties.PropertyUpstreamID])
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "org_block",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#org_block
			event: "org_block",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#OrgBlockEvent
			payload:    &github.OrgBlockEvent{},
			topic:      constants.TopicQueueEntityEvaluate,
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
					FullName: github.String("mindersec/minder"),
					HTMLURL:  github.String("https://github.com/mindersec/minder"),
				},
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "push raw payload",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
			event: "push",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PushEvent
			rawPayload: []byte(rawPushEvent),
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "branch_protection_configuration disabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event: "branch_protection_configuration",
			payload: &repoEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "branch_protection_configuration disabled raw event",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#branch_protection_configuration
			event:      "branch_protection_configuration",
			rawPayload: []byte(rawBranchProtectionConfigurationDisabledEvent),
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository_advisory published",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repoEvent{
				Action: github.String("disabled"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository_advisory reported",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_advisory
			event: "repository_advisory",
			payload: &repoEvent{
				Action: github.String("reported"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository_ruleset created",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository_ruleset deleted",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("deleted"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "repository_ruleset edited",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#repository_ruleset
			event: "repository_ruleset",
			payload: &repoEvent{
				Action: github.String("edited"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "secret_scanning_alert_location",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "secret_scanning_alert_location",
			payload: &repoEvent{
				Action: github.String("created"),
				Repo: newRepo(
					12345,
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					ID:     github.Int64(1234542),
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder/42"),
			},
			topic:      constants.TopicQueueOriginatingEntityAdd,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "pull_request reopened",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("reopened"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					ID:     github.Int64(1234542),
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder/42"),
			},
			topic:      constants.TopicQueueOriginatingEntityAdd,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "pull_request synchronize",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#secret_scanning_alert_location
			event: "pull_request",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
			payload: &github.PullRequestEvent{
				Action: github.String("synchronize"),
				Repo: newGitHubRepo(
					12345,
					"minder",
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
				PullRequest: &github.PullRequest{
					ID:     github.Int64(1234542),
					URL:    github.String("url"),
					Number: github.Int(42),
					User: &github.User{
						ID: github.Int64(42),
					},
				},
			},
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder/42"),
			},
			topic:      constants.TopicQueueRefreshEntityAndEvaluate,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
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
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder/42"),
			},
			topic:      constants.TopicQueueOriginatingEntityDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
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
			ghMocks: []func(hubMock gf.GitHubMock){
				gf.WithSuccessfulGetEntityName("mindersec/minder/42"),
			},
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Organization: &github.Organization{
					Login: github.String("stacklok"),
				},
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusInternalServerError,
			queued:     nil,
		},

		// garbage
		{
			name:  "garbage",
			event: "repository",
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
		},
		{
			name:  "total garbage",
			event: "garbage",
			payload: &garbage{
				Action:  github.String("created"),
				Garbage: github.String("garbage"),
			},
			topic:      constants.TopicQueueEntityEvaluate,
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")
			defer evt.Close()

			prevCredsFile, err := os.CreateTemp("", "prevcreds*")
			require.NoError(t, err, "failed to create temporary file")
			_, err = prevCredsFile.WriteString("also-not-our-secret\ntest")
			require.NoError(t, err, "failed to write to temporary file")
			defer os.Remove(prevCredsFile.Name())

			cfg := &serverconfig.WebhookConfig{}
			cfg.WebhookSecret = "not-our-secret"
			cfg.PreviousWebhookSecretFile = prevCredsFile.Name()
			defer evt.Close()

			pq := testqueue.NewPassthroughQueue(t)
			queued := pq.GetQueue()

			evt.Register(tt.topic, pq.Pass)

			go func() {
				err := evt.Run(context.Background())
				require.NoError(t, err, "failed to run eventer")
				require.NoError(t, pq.Close(), "failed to close queue")

			}()

			<-evt.Running()

			handler := HandleWebhookEvent(metrics.NewNoopMetrics(), evt, cfg)
			ts := httptest.NewServer(handler)
			t.Cleanup(ts.Close)

			var packageJson []byte
			if tt.payload != nil {
				packageJson, err = json.Marshal(tt.payload)
				require.NoError(t, err, "failed to marshal package event")
			} else {
				packageJson = tt.rawPayload
			}

			expectedMAC := sign(packageJson, "test")

			client := &http.Client{}
			resp, err := httpDoWithRetry(client, func() (*http.Request, error) {
				req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
				if err != nil {
					return nil, err
				}

				req.Header.Add("X-GitHub-Event", tt.event)
				req.Header.Add("X-GitHub-Delivery", "12345")
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", expectedMAC))
				return req, nil
			})
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

	projectID := uuid.New()
	providerID := uuid.New()

	autoregConfigEnabled := `{"github-app": {}, "auto_registration": {"entities": {"repository": {"enabled": true}}}}`
	autoregConfigDisabled := `{"github-app": {}, "auto_registration": {"entities": {"repository": {"enabled": false}}}}`

	tests := []struct {
		name          string
		event         string
		payload       any
		rawPayload    []byte
		mockStoreFunc df.MockStoreBuilder
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: df.NewMockStore(),
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
					"mindersec/minder",
					"https://github.com/mindersec/minder",
				),
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://example.com/random/url"),
				},
			},
			mockStoreFunc: df.NewMockStore(),
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
						"mindersec/minder",
						"https://github.com/mindersec/minder",
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
			mockStoreFunc: df.NewMockStore(),
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
						"mindersec/minder",
						"https://github.com/mindersec/minder",
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
			mockStoreFunc: df.NewMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, "provider_instance_removed", received.Metadata["event"])
				require.Equal(t, "github-app", received.Metadata["class"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "installation deleted raw payload",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			rawPayload:    []byte(rawInstallationDeletedEvent),
			mockStoreFunc: df.NewMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()
				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])
				require.Equal(t, "provider_instance_removed", received.Metadata["event"])
				require.Equal(t, "github-app", received.Metadata["class"])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
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
						"mindersec/minder",
						"https://github.com/mindersec/minder",
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
			mockStoreFunc: df.NewMockStore(),
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
						"mindersec/minder",
						"https://github.com/mindersec/minder",
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
			mockStoreFunc: df.NewMockStore(),
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
						"mindersec/minder",
						"https://github.com/mindersec/minder",
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
			mockStoreFunc: df.NewMockStore(),
			topic:         installations.ProviderInstallationTopic,
			statusCode:    http.StatusOK,
			queued:        nil,
		},

		// installation repositories events
		{
			name: "installation_repositories added",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation_repositories",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationRepositoriesEvent{
				Action: github.String("added"),
				RepositoriesAdded: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"mindersec/minder",
						"https://github.com/mindersec/minder",
					),
					newGitHubRepo(
						67890,
						"trusty",
						"stacklok/trusty",
						"https://github.com/stacklok/trusty",
					),
				},
				Installation: &github.Installation{
					ID: github.Int64(54321),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:         providerID,
						Definition: json.RawMessage(autoregConfigEnabled),
					},
					providerID,
				),
				df.WithSuccessfulGetInstallationIDByAppID(
					db.ProviderGithubAppInstallation{
						ProjectID: uuid.NullUUID{
							UUID:  projectID,
							Valid: true,
						},
						ProviderID: uuid.NullUUID{
							UUID:  providerID,
							Valid: true,
						},
					},
					54321),
			),
			topic:      constants.TopicQueueReconcileEntityAdd,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()

				var evt messages.MinderEvent

				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, providerID, evt.ProviderID)
				require.Equal(t, projectID, evt.ProjectID)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.EntityType)

				// the name can be either mindersec/minder or stacklok/trusty
				require.Contains(t, []string{"mindersec/minder", "stacklok/trusty"}, evt.Properties[properties.PropertyName])

				received = withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				err = json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, providerID, evt.ProviderID)
				require.Equal(t, projectID, evt.ProjectID)

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
		},
		{
			name: "installation_repositories autoreg disabled",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation_repositories",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationRepositoriesEvent{
				Action: github.String("added"),
				RepositoriesAdded: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"mindersec/minder",
						"https://github.com/mindersec/minder",
					),
					newGitHubRepo(
						67890,
						"trusty",
						"stacklok/trusty",
						"https://github.com/stacklok/trusty",
					),
				},
				Installation: &github.Installation{
					ID: github.Int64(54321),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:         providerID,
						Definition: json.RawMessage(autoregConfigDisabled),
					},
					providerID,
				),
				df.WithSuccessfulGetInstallationIDByAppID(
					db.ProviderGithubAppInstallation{
						ProjectID: uuid.NullUUID{
							UUID:  projectID,
							Valid: true,
						},
						ProviderID: uuid.NullUUID{
							UUID:  providerID,
							Valid: true,
						},
					},
					54321),
			),
			topic:      constants.TopicQueueReconcileEntityAdd,
			statusCode: http.StatusOK,
			//nolint:thelper
			queued: nil,
		},
		{
			name: "installation_repositories removed",
			// https://docs.github.com/en/webhooks/webhook-events-and-payloads#installation
			event: "installation_repositories",
			// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#InstallationEvent
			payload: &github.InstallationRepositoriesEvent{
				Action: github.String("removed"),
				RepositoriesRemoved: []*github.Repository{
					newGitHubRepo(
						12345,
						"minder",
						"mindersec/minder",
						"https://github.com/mindersec/minder",
					),
					newGitHubRepo(
						67890,
						"trusty",
						"stacklok/trusty",
						"https://github.com/stacklok/trusty",
					),
				},
				Installation: &github.Installation{
					ID: github.Int64(54321),
				},
				Sender: &github.User{
					Login:   github.String("stacklok"),
					HTMLURL: github.String("https://github.com/apps"),
				},
			},
			mockStoreFunc: df.NewMockStore(
				df.WithSuccessfulGetProviderByID(
					db.Provider{
						ID:         providerID,
						Definition: json.RawMessage(autoregConfigEnabled),
					},
					providerID,
				),
				df.WithSuccessfulGetInstallationIDByAppID(
					db.ProviderGithubAppInstallation{
						ProjectID: uuid.NullUUID{
							UUID:  projectID,
							Valid: true,
						},
						ProviderID: uuid.NullUUID{
							UUID:  providerID,
							Valid: true,
						},
					},
					54321),
			),
			topic:      constants.TopicQueueGetEntityAndDelete,
			statusCode: http.StatusOK,
			queued: func(t *testing.T, event string, ch <-chan *message.Message) {
				t.Helper()

				var evt entMsg.HandleEntityAndDoMessage

				received := withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				err := json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				// we use contains here because the messages can arrive in any order
				require.Contains(t, []string{"12345", "67890"}, evt.Entity.GetByProps[properties.PropertyUpstreamID])

				received = withTimeout(ch, timeout)
				require.NotNilf(t, received, "no event received after waiting %s", timeout)
				require.Equal(t, "12345", received.Metadata["id"])
				require.Equal(t, event, received.Metadata["type"])
				require.Equal(t, "https://api.github.com/", received.Metadata["source"])

				err = json.Unmarshal(received.Payload, &evt)
				require.NoError(t, err)
				require.Equal(t, "github", evt.Hint.ProviderImplementsHint)
				require.Equal(t, v1.Entity_ENTITY_REPOSITORIES, evt.Entity.Type)
				// we use contains here because the messages can arrive in any order
				require.Contains(t, []string{"12345", "67890"}, evt.Entity.GetByProps[properties.PropertyUpstreamID])

				received = withTimeout(ch, timeout)
				require.Nil(t, received)
			},
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

			var mockStore *mockdb.MockStore
			if tt.mockStoreFunc != nil {
				mockStore = tt.mockStoreFunc(ctrl)
			} else {
				mockStore = mockdb.NewMockStore(ctrl)
			}

			evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")
			defer evt.Close()

			cfg := &serverconfig.WebhookConfig{}
			cfg.WebhookSecret = "test"

			pq := testqueue.NewPassthroughQueue(t)
			defer pq.Close()
			queued := pq.GetQueue()

			evt.Register(tt.topic, pq.Pass)

			go func() {
				err := evt.Run(context.Background())
				require.NoError(t, err, "failed to run eventer")
			}()

			<-evt.Running()

			var c *serverconfig.Config
			tokenKeyPath := generateTokenKey(t)
			c = &serverconfig.Config{
				Auth: serverconfig.AuthConfig{
					TokenKey: tokenKeyPath,
				},
			}

			eng, err := crypto.NewEngineFromConfig(c)
			require.NoError(t, err)
			ghClientService := ghService.NewGithubProviderService(
				mockStore,
				eng,
				metrics.NewNoopMetrics(),
				// These nil dependencies do not matter for the current tests
				&serverconfig.ProviderConfig{
					GitHubApp: &serverconfig.GitHubAppConfig{
						WebhookSecret: "test",
					},
				},
				nil,
				nil,
			)

			handler := HandleGitHubAppWebhook(mockStore, ghClientService, metrics.NewNoopMetrics(), evt)
			ts := httptest.NewServer(http.HandlerFunc(handler))
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
			resp, err := httpDoWithRetry(client, func() (*http.Request, error) {
				req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer(packageJson))
				if err != nil {
					return nil, err
				}

				req.Header.Add("X-GitHub-Event", tt.event)
				req.Header.Add("X-GitHub-Delivery", "12345")
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("X-Hub-Signature-256", fmt.Sprintf("sha256=%s", expectedMAC))
				return req, nil
			})
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

// TODO: deduplicate this function with the one in the other test file
func generateTokenKey(t *testing.T) string {
	t.Helper()

	tmpdir := t.TempDir()

	tokenKeyPath := filepath.Join(tmpdir, "/token_key")

	// generate 256-bit key
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)
	encodedKey := base64.StdEncoding.EncodeToString(key)

	// Write token key to file
	err = os.WriteFile(tokenKeyPath, []byte(encodedKey), 0600)
	require.NoError(t, err, "failed to write token key to file")

	return tokenKeyPath
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

// httpDoWithRetry takes a createRequest function rather than a request
// to avoid reusing the req.Body io.Reader for a second request.
func httpDoWithRetry(client *http.Client, createRequest func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response

	err := backoff.Retry(func() error {
		req, err := createRequest()
		if err != nil {
			return err
		}

		resp, err = client.Do(req)
		return err
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 3))

	return resp, err
}

type garbage struct {
	Action  *string `json:"action,omitempty"`
	Garbage *string `json:"garbage,omitempty"`
}
