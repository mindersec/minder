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
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/stacklok/mediator/pkg/db"
)

// Repository represents a GitHub repository
type Repository struct {
	Owner string
	Repo  string
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
	HookID     int64
	HookURL    string
	DeployURL  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	HookName   string
	HookType   string
	RegistrationStatus
}

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
//
//gocyclo:ignore
func HandleGitHubWebHook(_ db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validate the payload signature. This is required for security reasons.
		// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks
		// for more information. Note that this is not required for the GitHub App
		// webhook secret, but it is required for OAuth2 App.
		segments := strings.Split(r.URL.Path, "/")
		uuid := segments[len(segments)-1]
		fmt.Println(uuid)

		payload, err := github.ValidatePayload(r, []byte(viper.GetString("github-app.app.webhook_secret")))
		if err != nil {
			fmt.Printf("Error validating webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse the payload into the appropriate event type
		// See https://pkg.go.dev/github.com/google/go-github/v53/github#ParseWebHook
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			fmt.Printf("Error parsing webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Handle the specific event type
		// Some of these may not be relevant to mediator, but lets log them for now
		// in case anyone gets creative.
		// See https://github.com/google/go-github/blob/5a9f8f7d92f20e3a16770640ce9f15694f5c03ff/github/event.go#L30-L153
		switch e := event.(type) {
		case *github.BranchProtectionRuleEvent:
			// Handle branch protection rule event
			fmt.Printf("Branch protection rule event received: %+v\n", e)
		case *github.CheckRunEvent:
			// Handle check run event
			fmt.Printf("Check run event received: %+v\n", e)
		case *github.CheckSuiteEvent:
			// Handle check suite event
			fmt.Printf("Check suite event received: %+v\n", e)
		case *github.CodeScanningAlertEvent:
			// Handle code scanning alert event
			fmt.Printf("Code scanning alert event received: %+v\n", e)
		case *github.CommitCommentEvent:
			// Handle commit comment event
			fmt.Printf("Commit comment event received: %+v\n", e)
		case *github.ContentReferenceEvent:
			// Handle content reference event
			fmt.Printf("Content reference event received: %+v\n", e)
		case *github.CreateEvent:
			// Handle create event
			fmt.Printf("Create event received: %+v\n", e)
		case *github.DeployKeyEvent:
			// Handle deploy key event
			fmt.Printf("Deploy key event received: %+v\n", e)
		case *github.DeploymentEvent:
			// Handle deployment event
			fmt.Printf("Deployment event received: %+v\n", e)
		case *github.DeploymentStatusEvent:
			// Handle deployment status event
			fmt.Printf("Deployment status event received: %+v\n", e)
		case *github.DeploymentProtectionRuleEvent:
			// Handle deployment protection rule event
			fmt.Printf("Deployment protection rule event received: %+v\n", e)
		case *github.DiscussionEvent:
			// Handle discussion event
			fmt.Printf("Discussion event received: %+v\n", e)
		case *github.DiscussionCommentEvent:
			// Handle discussion comment event
			fmt.Printf("Discussion comment event received: %+v\n", e)
		case *github.ForkEvent:
			// Handle fork event
			fmt.Printf("Fork event received: %+v\n", e)
		case *github.GitHubAppAuthorizationEvent:
			// Handle GitHub App authorization event
			fmt.Printf("GitHub App authorization event received: %+v\n", e)
		case *github.GollumEvent:
			// Handle Gollum event
			fmt.Printf("Gollum event received: %+v\n", e)
		case *github.InstallationEvent:
			// Handle installation event
			fmt.Printf("Installation event received: %+v\n", e)
		case *github.InstallationRepositoriesEvent:
			// Handle installation repositories event
			fmt.Printf("Installation repositories event received: %+v\n", e)
		case *github.IssueCommentEvent:
			// Handle issue comment event
			fmt.Printf("Issue comment event received: %+v\n", e)
		case *github.IssuesEvent:
			// Handle issues event
			fmt.Printf("Issues event received: %+v\n", e)
		case *github.LabelEvent:
			// Handle label event
			fmt.Printf("Label event received: %+v\n", e)
		case *github.MarketplacePurchaseEvent:
			// Handle marketplace purchase event
			fmt.Printf("Marketplace purchase event received: %+v\n", e)
		case *github.MemberEvent:
			// Handle member event
			fmt.Printf("Member event received: %+v\n", e)
		case *github.MembershipEvent:
			// Handle membership event
			fmt.Printf("Membership event received: %+v\n", e)
		case *github.MergeGroupEvent:
			// Handle merge group event
			fmt.Printf("Merge group event received: %+v\n", e)
		case *github.MetaEvent:
			// Handle meta event
			fmt.Printf("Meta event received: %+v\n", e)
		case *github.MilestoneEvent:
			// Handle milestone event
			fmt.Printf("Milestone event received: %+v\n", e)
		case *github.OrganizationEvent:
			// Handle organization event
			fmt.Printf("Organization event received: %+v\n", e)
		case *github.OrgBlockEvent:
			// Handle org block event
			fmt.Printf("Org block event received: %+v\n", e)
		case *github.PackageEvent:
			// Handle package event
			fmt.Printf("Package event received: %+v\n", e)
		case *github.PageBuildEvent:
			// Handle page build event
			fmt.Printf("Page build event received: %+v\n", e)
		case *github.PingEvent:
			// Handle ping event
			fmt.Printf("Ping event received: %+v\n", e)
		case *github.ProjectEvent:
			// Handle project event
			fmt.Printf("Project event received: %+v\n", e)
		case *github.ProjectCardEvent:
			// Handle project card event
			fmt.Printf("Project card event received: %+v\n", e)
		case *github.ProjectColumnEvent:
			// Handle project column event
			fmt.Printf("Project column event received: %+v\n", e)
		case *github.PublicEvent:
			// Handle public event
			fmt.Printf("Public event received: %+v\n", e)
		case *github.PullRequestEvent:
			// Handle pull request event
			fmt.Printf("Pull request event received: %+v\n", e)
		case *github.PullRequestReviewEvent:
			// Handle pull request review event
			fmt.Printf("Pull request review event received: %+v\n", e)
		case *github.PullRequestReviewCommentEvent:
			// Handle pull request review comment event
			fmt.Printf("Pull request review comment event received: %+v\n", e)
		case *github.PullRequestTargetEvent:
			// Handle pull request target event
			fmt.Printf("Pull request target event received: %+v\n", e)
		case *github.PushEvent:
			// Handle push event
			fmt.Printf("Push event received: %+v\n", e)
		case *github.ReleaseEvent:
			// Handle release event
			fmt.Printf("Release event received: %+v\n", e)
		case *github.RepositoryEvent:
			// Handle repository event
			fmt.Printf("Repository event received: %+v\n", e)
		case *github.RepositoryDispatchEvent:
			// Handle repository dispatch event
			fmt.Printf("Repository dispatch event received: %+v\n", e)
		case *github.RepositoryImportEvent:
			// Handle repository import event
			fmt.Printf("Repository import event received: %+v\n", e)
		case *github.RepositoryVulnerabilityAlertEvent:
			// Handle repository vulnerability alert event
			fmt.Printf("Repository vulnerability alert event received: %+v\n", e)
		case *github.SecretScanningAlertEvent:
			// Handle secret scanning alert event
			fmt.Printf("Secret scanning alert event received: %+v\n", e)
		case *github.SecurityAdvisoryEvent:
			// Handle security advisory event
			fmt.Printf("Security advisory event received: %+v\n", e)
		case *github.StarEvent:
			// Handle star event
			fmt.Printf("Star event received: %+v\n", e)
		case *github.StatusEvent:
			// Handle status event
			fmt.Printf("Status event received: %+v\n", e)
		case *github.TeamEvent:
			// Handle team event
			fmt.Printf("Team event received: %+v\n", e)
		case *github.TeamAddEvent:
			// Handle team add event
			fmt.Printf("Team add event received: %+v\n", e)
		case *github.UserEvent:
			// Handle user event
			fmt.Printf("User event received: %+v\n", e)
		case *github.WatchEvent:
			// Handle watch event
			fmt.Printf("Watch event received: %+v\n", e)
		case *github.WorkflowDispatchEvent:
			// Handle workflow dispatch event
			fmt.Printf("Workflow dispatch event received: %+v\n", e)
		case *github.WorkflowJobEvent:
			// Handle workflow job event
			fmt.Printf("Workflow job event received: %+v\n", e)
		case *github.WorkflowRunEvent:
			// Handle workflow run event
			fmt.Printf("Workflow run event received: %+v\n", e)
		default:
			// Unknown event
			fmt.Printf("Unknown event type received: %+v\n", e)
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

	fmt.Println()

	for _, repo := range repositories {
		result := RegistrationStatus{
			Success: true,
			Error:   nil,
		}
		urlUUID := uuid.New().String()

		url := viper.GetString("github-app.external_webhook_url")
		webhookUrl := fmt.Sprintf("%s/%s", url, urlUUID)
		hook := &github.Hook{
			Config: map[string]interface{}{
				"url":          webhookUrl,
				"content_type": "json",
				"ping_url":     viper.GetString("github-app.app.external_ping_url"),
				"secret":       viper.GetString("github-app.app.webhook_secret"),
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
			HookID:     mhook.GetID(),
			HookURL:    mhook.GetURL(),
			DeployURL:  webhookUrl,
			CreatedAt:  mhook.GetCreatedAt().Time,
			UpdatedAt:  mhook.GetUpdatedAt().Time,
			HookType:   mhook.GetType(),
			HookName:   mhook.GetName(),
			RegistrationStatus: RegistrationStatus{
				Success: result.Success,
				Error:   result.Error,
			},
		}

		registerData = append(registerData, regResult)

		// iterate over registerData and print out the results

	}

	return registerData, nil
}
