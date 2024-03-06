// Copyright 2024 Stacklok, Inc
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

package webhooks_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/config/server"
	ghprovider "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	mockghhook "github.com/stacklok/minder/internal/repositories/github/webhooks/mock"
	"github.com/stacklok/minder/internal/util/ptr"
)

func TestWebhookManager_DeleteWebhook(t *testing.T) {
	t.Parallel()

	deletionScenarios := []struct {
		Name          string
		ClientSetup   mockBuilder
		ShouldSucceed bool
		ExpectedError string
	}{
		{
			Name:          "DeleteWebhook returns error when deletion request fails",
			ClientSetup:   newClient(withFailedDeletion),
			ShouldSucceed: false,
			ExpectedError: "error deleting hook",
		},
		{
			Name:          "DeleteWebhook skips webhooks which cannot be found",
			ClientSetup:   newClient(withNotFoundDeletion),
			ShouldSucceed: true,
		},
		{
			Name:          "DeleteWebhook successfully deletes a webhook",
			ClientSetup:   newClient(withSuccessfulDeletion),
			ShouldSucceed: true,
		},
	}

	for i := range deletionScenarios {
		scenario := deletionScenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			client := scenario.ClientSetup(ctrl)
			err := webhooks.NewWebhookManager(webhookConfig).
				DeleteWebhook(ctx, client, repoOwner, repoName, hookID)

			if scenario.ShouldSucceed {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

func TestWebhookManager_CreateWebhook(t *testing.T) {
	t.Parallel()

	creationScenarios := []struct {
		Name          string
		ClientSetup   mockBuilder
		ShouldSucceed bool
		ExpectedError string
	}{
		{
			Name:          "CreateWebhook returns error when listing request fails",
			ClientSetup:   newClient(withFailedList),
			ShouldSucceed: false,
			ExpectedError: "error listing hooks",
		},
		{
			Name:          "CreateWebhook returns error when webhook config cannot be parsed",
			ClientSetup:   newClient(withSuccessfulList("")),
			ShouldSucceed: false,
			ExpectedError: "unexpected hook config structure",
		},
		{
			Name:          "CreateWebhook returns error when stale hook deletion fails",
			ClientSetup:   newClient(withSuccessfulList(webhookURL), withFailedDeletion),
			ShouldSucceed: false,
			ExpectedError: "error deleting hook",
		},
		{
			Name:          "CreateWebhook returns error when creation request fails",
			ClientSetup:   newClient(withSuccessfulList(webhookURL), withSuccessfulDeletion, withFailedCreation),
			ShouldSucceed: false,
			ExpectedError: "error creating hook",
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook for repo with no previous webhooks",
			ClientSetup:   newClient(withNotFoundList, withSuccessfulCreation),
			ShouldSucceed: true,
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook, ignoring other projects' webhooks",
			ClientSetup:   newClient(withSuccessfulList("http://hook.foo.com/67890"), withSuccessfulCreation),
			ShouldSucceed: true,
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook, and deletes stale hook",
			ClientSetup:   newClient(withSuccessfulList(webhookURL), withSuccessfulDeletion, withSuccessfulCreation),
			ShouldSucceed: true,
		},
	}

	for i := range creationScenarios {
		scenario := creationScenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			client := scenario.ClientSetup(ctrl)
			resultID, hook, err := webhooks.NewWebhookManager(webhookConfig).
				CreateWebhook(ctx, client, repoOwner, repoName)

			if scenario.ShouldSucceed {
				require.NoError(t, err)
				require.Equal(t, result, hook)
				// can't do much with the ID since it is a random UUID
				// assert that it is in fact a string of a UUID
				_, err := uuid.Parse(resultID)
				require.NoError(t, err)
			} else {
				require.Equal(t, "", resultID)
				require.Nil(t, hook)
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

type clientMock = *mockghhook.MockGitHubWebhookClient
type mockBuilder = func(*gomock.Controller) clientMock

const (
	repoOwner = "acme"
	repoName  = "api-gateway"
	hookID    = int64(12345)
)

var (
	errTest       = errors.New("oh no")
	webhookConfig = server.WebhookConfig{
		ExternalWebhookURL: "https://example.com/api/v1/webhook/github",
		ExternalPingURL:    "https://example.com/api/v1/health",
		WebhookSecret:      "",
	}
	result     = &github.Hook{ID: ptr.Ptr[int64](hookID)}
	webhookURL = webhookConfig.ExternalWebhookURL + "/12345"
)

func newClient(opts ...func(clientMock)) mockBuilder {
	return func(ctrl *gomock.Controller) clientMock {
		mock := mockghhook.NewMockGitHubWebhookClient(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withSuccessfulDeletion(mock *mockghhook.MockGitHubWebhookClient) {
	stubDelete(mock, nil, nil)
}

func withFailedDeletion(mock clientMock) {
	stubDelete(mock, nil, errTest)
}

func withNotFoundDeletion(mock clientMock) {
	githubResp := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	stubDelete(mock, githubResp, errTest)
}

func stubDelete(mock clientMock, resp *github.Response, err error) {
	mock.EXPECT().
		DeleteHook(gomock.Any(), gomock.Eq(repoOwner), gomock.Eq(repoName), gomock.Eq(hookID)).
		Return(resp, err)
}

func withSuccessfulList(url string) func(clientMock) {
	hookConfig := map[string]any{
		"url": url,
	}
	hooks := []*github.Hook{
		{
			ID:     ptr.Ptr(hookID),
			Config: hookConfig,
		},
	}
	return func(mock clientMock) {
		stubList(mock, hooks, nil)
	}
}

func withFailedList(mock clientMock) {
	stubList(mock, []*github.Hook{}, errTest)
}

func withNotFoundList(mock clientMock) {
	stubList(mock, []*github.Hook{}, ghprovider.ErrNotFound)
}

func stubList(mock clientMock, hooks []*github.Hook, err error) {
	mock.EXPECT().
		ListHooks(gomock.Any(), gomock.Eq(repoOwner), gomock.Eq(repoName)).
		Return(hooks, err)
}

func withFailedCreation(mock clientMock) {
	stubCreate(mock, nil, errTest)
}

func withSuccessfulCreation(mock clientMock) {
	stubCreate(mock, result, nil)
}

func stubCreate(mock clientMock, hook *github.Hook, err error) {
	// it would be nice to be able to make some assertions about the webhook
	// config which gets passed here... this requires more investigation
	mock.EXPECT().
		CreateHook(gomock.Any(), gomock.Eq(repoOwner), gomock.Eq(repoName), gomock.Any()).
		Return(hook, err)
}
