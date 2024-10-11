// Copyright 2024 Stacklok, Inc.
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

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/events"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// pullRequestEvent are events related to pull requests issued around
// a specific repository
type pullRequestEvent struct {
	Action      *string      `json:"action,omitempty"`
	Repo        *repo        `json:"repository,omitempty"`
	PullRequest *pullRequest `json:"pull_request,omitempty"`
}

func (p *pullRequestEvent) GetAction() string {
	if p.Action != nil {
		return *p.Action
	}
	return ""
}

func (p *pullRequestEvent) GetRepo() *repo {
	return p.Repo
}

func (p *pullRequestEvent) GetPullRequest() *pullRequest {
	return p.PullRequest
}

type pullRequest struct {
	ID     *int64  `json:"id,omitempty"`
	URL    *string `json:"url,omitempty"`
	Number *int64  `json:"number,omitempty"`
	User   *user   `json:"user,omitempty"`
}

func (p *pullRequest) GetID() int64 {
	if p.ID != nil {
		return *p.ID
	}
	return 0
}

func (p *pullRequest) GetURL() string {
	if p.URL != nil {
		return *p.URL
	}
	return ""
}

func (p *pullRequest) GetNumber() int64 {
	if p.Number != nil {
		return *p.Number
	}
	return 0
}

func (p *pullRequest) GetUser() *user {
	return p.User
}

func processPullRequestEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	l := zerolog.Ctx(ctx)

	var event *pullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.GetAction() == "" {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.GetRepo() == nil {
		return nil, errors.New("invalid event: repo is nil")
	}
	if event.GetPullRequest() == nil {
		return nil, errors.New("invalid event: pull request is nil")
	}
	if event.GetPullRequest().GetURL() == "" {
		return nil, errors.New("invalid pull request: URL is nil")
	}
	if event.GetPullRequest().GetNumber() == 0 {
		return nil, errors.New("invalid pull request: number is 0")
	}
	if event.GetPullRequest().GetUser() == nil {
		return nil, errors.New("invalid pull request: user is nil")
	}
	if event.GetPullRequest().GetUser().GetID() == 0 {
		return nil, errors.New("invalid user: id is 0")
	}

	ghRepo := event.GetRepo()
	pullProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.GetPullRequest().GetID()),
		ghprop.PullPropertyRepoName:   ghRepo.GetName(),
		ghprop.PullPropertyRepoOwner:  ghRepo.GetOwner(),
		ghprop.PullPropertyNumber:     event.GetPullRequest().GetNumber(),
		ghprop.PullPropertyAction:     event.GetAction(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating pull request properties: %w", err)
	}

	repoProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(ghRepo.GetID()),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository properties for PR origination: %w", err)
	}

	// it is bit of a code smell to use the fetcher here just to format the name
	name, err := ghprop.NewPullRequestFetcher().GetName(pullProps)
	if err != nil {
		return nil, fmt.Errorf("error fetching pull request name: %w", err)
	}
	nameProp, err := properties.NewProperty(name)
	if err != nil {
		return nil, fmt.Errorf("error creating property for the name: %w", err)
	}
	pullProps.SetProperty(properties.PropertyName, nameProp)

	var topic string
	switch pullProps.GetProperty(ghprop.PullPropertyAction).GetString() {
	case webhookActionEventOpened,
		webhookActionEventReopened,
		webhookActionEventSynchronize:
		topic = events.TopicQueueOriginatingEntityAdd
	case webhookActionEventClosed:
		topic = events.TopicQueueOriginatingEntityDelete
	default:
		zerolog.Ctx(ctx).Info().Msgf("action %s is not handled for pull requests",
			pullProps.GetProperty(ghprop.PullPropertyAction).GetString())
		return nil, errNotHandled
	}

	prMsg := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntity(pb.Entity_ENTITY_PULL_REQUESTS, pullProps).
		WithOriginator(pb.Entity_ENTITY_REPOSITORIES, repoProps).
		WithProviderImplementsHint(string(db.ProviderTypeGithub))

	l.Info().Msgf("evaluating PR %s\n", event.GetPullRequest().GetURL())

	return &processingResult{topic: topic, wrapper: prMsg}, nil
}
