// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// pullRequestEvent are events related to pull requests issued around
// a specific repository
type pullRequestEvent struct {
	Action      string      `json:"action,omitempty"`
	Repo        repo        `json:"repository,omitempty"`
	PullRequest pullRequest `json:"pull_request,omitempty"`
}

type pullRequest struct {
	ID     int64  `json:"id,omitempty"`
	URL    string `json:"url,omitempty"`
	Number int64  `json:"number,omitempty"`
	User   user   `json:"user,omitempty"`
}

func processPullRequestEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	l := zerolog.Ctx(ctx)

	var event pullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Action == "" {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.Repo.ID == 0 {
		return nil, errors.New("invalid event: repo is nil")
	}
	if event.PullRequest.URL == "" {
		return nil, errors.New("invalid pull request: URL is nil")
	}
	if event.PullRequest.Number == 0 {
		return nil, errors.New("invalid pull request: number is 0")
	}
	if event.PullRequest.User.ID == 0 {
		return nil, errors.New("invalid user: id is 0")
	}

	ghRepo := event.Repo
	pullProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.PullRequest.ID),
		ghprop.PullPropertyRepoName:   ghRepo.Name,
		ghprop.PullPropertyRepoOwner:  ghRepo.GetOwner(),
		ghprop.PullPropertyNumber:     event.PullRequest.Number,
		ghprop.PullPropertyAction:     event.Action,
	})

	repoProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(ghRepo.ID),
	})

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

	topic, err := getPREventHandlingTopic(pullProps)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Str("action", event.Action).
			Msg("error getting PR event handling topic")
		return nil, err
	}

	prMsg := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntity(pb.Entity_ENTITY_PULL_REQUESTS, pullProps).
		WithOriginator(pb.Entity_ENTITY_REPOSITORIES, repoProps).
		WithProviderImplementsHint(string(db.ProviderTypeGithub))

	l.Info().Msgf("evaluating PR %s: %s => %s\n", event.PullRequest.URL, event.Action, topic)

	return &processingResult{topic: topic, wrapper: prMsg}, nil
}

func getPREventHandlingTopic(pullProps *properties.Properties) (string, error) {
	switch pullProps.GetProperty(ghprop.PullPropertyAction).GetString() {
	case webhookActionEventOpened,
		webhookActionEventReopened:
		return constants.TopicQueueOriginatingEntityAdd, nil
	case webhookActionEventSynchronize:
		return constants.TopicQueueRefreshEntityAndEvaluate, nil
	case webhookActionEventClosed:
		return constants.TopicQueueOriginatingEntityDelete, nil
	default:
		return "", errNotHandled
	}
}
