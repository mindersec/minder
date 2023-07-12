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

// Package controlplane contains the control plane API for the mediator.
package controlplane

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v53/github"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// Repository represents a GitHub repository
type Repository struct {
	Owner  string
	Repo   string
	RepoID int32
}

// RegistrationStatus gathers the status of the webhook call for each repository
type RegistrationStatus struct {
	Success bool
	Error   error
}

// RepositoryResult represents the result of the webhook registration
type RepositoryResult struct {
	Owner      string
	Repository string
	RepoID     int32
	HookID     int64
	HookURL    string
	DeployURL  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	HookName   string
	HookType   string
	HookUUID   string
	RegistrationStatus
}

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
//
//gocyclo:ignore
func HandleGitHubWebHook(p message.Publisher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate the payload signature. This is required for security reasons.
		// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks
		// for more information. Note that this is not required for the GitHub App
		// webhook secret, but it is required for OAuth2 App.
		// it returns a uuid for the webhook, but we are not currently using it
		segments := strings.Split(r.URL.Path, "/")
		_ = segments[len(segments)-1]

		payload, err := github.ValidatePayload(r, []byte(viper.GetString("webhook-config.webhook_secret")))
		if err != nil {
			fmt.Printf("Error validating webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// TODO: extract sender and event time from payload portably
		m := message.NewMessage(uuid.New().String(), payload)
		m.Metadata.Set("id", github.DeliveryID(r))
		m.Metadata.Set("source", "https://api.github.com/") // TODO: handle other sources
		m.Metadata.Set("type", github.WebHookType(r))
		// m.Metadata.Set("subject", ghEvent.GetRepo().GetFullName())
		// m.Metadata.Set("time", ghEvent.GetCreatedAt().String())
		log.Printf("publishing of type: %s", m.Metadata["type"])

		if err := p.Publish(m.Metadata["type"], m); err != nil {
			fmt.Printf("Error publishing message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// RegisterWebHook registers a webhook for the given repositories and events
// and returns the registration result for each repository.
// If an error occurs, the registration is aborted and the error is returned.
// https://docs.github.com/en/rest/reference/repos#create-a-repository-webhook
func RegisterWebHook(
	ctx context.Context,
	token oauth2.Token,
	repositories []Repository,
	events []string,
) ([]RepositoryResult, error) {

	var registerData []RepositoryResult

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	for _, repo := range repositories {
		result := RegistrationStatus{
			Success: true,
			Error:   nil,
		}
		urlUUID := uuid.New().String()

		viper.SetDefault("webhook-config.external_webhook_url", "")
		viper.SetDefault("webhook-config.external_ping_url", "")
		viper.SetDefault("webhook-config.webhook_secret", "")

		url := viper.GetString("webhook-config.external_webhook_url")
		ping := viper.GetString("webhook-config.external_ping_url")
		secret := viper.GetString("webhook-config.webhook_secret")
		if url == "" || ping == "" || secret == "" {
			result.Success = false
			result.Error = fmt.Errorf("github app incorrectly configured")
		}
		webhookUrl := fmt.Sprintf("%s/%s", url, urlUUID)
		hook := &github.Hook{
			Config: map[string]interface{}{
				"url":          webhookUrl,
				"content_type": "json",
				"ping_url":     ping,
				"secret":       secret,
			},
			Events: events,
		}

		// Attempt to register webhook
		mhook, _, err := client.Repositories.CreateHook(ctx, repo.Owner, repo.Repo, hook)
		if err != nil {
			result.Success = false
			result.Error = err
		}

		regResult := RepositoryResult{
			Repository: repo.Repo,
			Owner:      repo.Owner,
			RepoID:     repo.RepoID,
			HookID:     mhook.GetID(),
			HookURL:    mhook.GetURL(),
			DeployURL:  webhookUrl,
			CreatedAt:  mhook.GetCreatedAt().Time,
			UpdatedAt:  mhook.GetUpdatedAt().Time,
			HookType:   mhook.GetType(),
			HookName:   mhook.GetName(),
			HookUUID:   urlUUID,
			RegistrationStatus: RegistrationStatus{
				Success: result.Success,
				Error:   result.Error,
			},
		}

		registerData = append(registerData, regResult)

	}

	return registerData, nil
}
