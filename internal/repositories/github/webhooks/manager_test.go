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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/config/server"
	cf "github.com/stacklok/minder/internal/repositories/github/fixtures"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
)

func TestWebhookManager_DeleteWebhook(t *testing.T) {
	t.Parallel()

	deletionScenarios := []struct {
		Name          string
		ClientSetup   cf.ClientMockBuilder
		ShouldSucceed bool
		ExpectedError string
	}{
		{
			Name:          "DeleteWebhook returns error when deletion request fails",
			ClientSetup:   cf.NewClientMock(cf.WithFailedDeletion),
			ShouldSucceed: false,
			ExpectedError: "error deleting hook",
		},
		{
			Name:          "DeleteWebhook skips webhooks which cannot be found",
			ClientSetup:   cf.NewClientMock(cf.WithNotFoundDeletion),
			ShouldSucceed: true,
		},
		{
			Name:          "DeleteWebhook successfully deletes a webhook",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulDeletion),
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
				DeleteWebhook(ctx, client, cf.RepoOwner, cf.RepoName, cf.HookID)

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
		ClientSetup   cf.ClientMockBuilder
		ShouldSucceed bool
		ExpectedError string
	}{
		{
			Name:          "CreateWebhook returns error when listing request fails",
			ClientSetup:   cf.NewClientMock(cf.WithFailedList),
			ShouldSucceed: false,
			ExpectedError: "error listing hooks",
		},
		{
			Name:          "CreateWebhook returns error when webhook config cannot be parsed",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulList("")),
			ShouldSucceed: false,
			ExpectedError: "unexpected hook config structure",
		},
		{
			Name:          "CreateWebhook returns error when stale hook deletion fails",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulList(webhookURL), cf.WithFailedDeletion),
			ShouldSucceed: false,
			ExpectedError: "error deleting hook",
		},
		{
			Name:          "CreateWebhook returns error when creation request fails",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulList(webhookURL), cf.WithSuccessfulDeletion, cf.WithFailedCreation),
			ShouldSucceed: false,
			ExpectedError: "error creating hook",
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook for repo cf.With no previous webhooks",
			ClientSetup:   cf.NewClientMock(cf.WithNotFoundList, cf.WithSuccessfulCreation),
			ShouldSucceed: true,
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook, ignoring other projects' webhooks",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulList("http://hook.foo.com/67890"), cf.WithSuccessfulCreation),
			ShouldSucceed: true,
		},
		{
			Name:          "CreateWebhook successfully creates a new webhook, and deletes stale hook",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulList(webhookURL), cf.WithSuccessfulDeletion, cf.WithSuccessfulCreation),
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
				CreateWebhook(ctx, client, cf.RepoOwner, cf.RepoName)

			if scenario.ShouldSucceed {
				require.NoError(t, err)
				require.Equal(t, cf.ResultHook, hook)
				// can't do much cf.With the ID since it is a random UUID
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

var (
	webhookConfig = server.WebhookConfig{
		ExternalWebhookURL: "https://example.com/api/v1/webhook/github",
		ExternalPingURL:    "https://example.com/api/v1/health",
		WebhookSecret:      "",
	}
	webhookURL = webhookConfig.ExternalWebhookURL + "/12345"
)
