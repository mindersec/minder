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
	"strings"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/events"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// repoEvent represents any event related to a repository.
type repoEvent struct {
	Action *string `json:"action,omitempty"`
	Repo   *repo   `json:"repository,omitempty"`
	HookID *int64  `json:"hook_id,omitempty"`
}

func (r *repoEvent) GetAction() string {
	if r.Action != nil {
		return *r.Action
	}
	return ""
}

func (r *repoEvent) GetRepo() *repo {
	return r.Repo
}

func (r *repoEvent) GetHookID() int64 {
	if r.HookID != nil {
		return *r.HookID
	}
	return 0
}

type repo struct {
	ID       *int64  `json:"id,omitempty"`
	Name     *string `json:"name,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	HTMLURL  *string `json:"html_url,omitempty"`
	Private  *bool   `json:"private,omitempty"`
}

func (r *repo) GetID() int64 {
	if r.ID != nil {
		return *r.ID
	}
	return 0
}

func (r *repo) GetName() string {
	if r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *repo) GetFullName() string {
	if r.FullName != nil {
		return *r.FullName
	}
	return ""
}

func (r *repo) GetHTMLURL() string {
	if r.HTMLURL != nil {
		return *r.HTMLURL
	}
	return ""
}

func (r *repo) GetPrivate() bool {
	if r.Private != nil {
		return *r.Private
	}
	return false
}

func (r *repo) GetOwner() string {
	if r.FullName != nil {
		parts := strings.SplitN(*r.FullName, "/", 2)
		// It is ok to always return the first item since it
		// defaults to empty string in case the string has no
		// separators.
		return parts[0]
	}
	return ""
}

func processRepositoryEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *repoEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetRepo() == nil {
		return nil, errRepoNotFound
	}

	l := zerolog.Ctx(ctx).With().
		Str("github-event-action", event.GetAction()).
		Int64("github-repository-id", event.GetRepo().GetID()).
		Str("github-repository-url", event.GetRepo().GetHTMLURL()).
		Logger()

	if event.GetRepo().GetID() == 0 {
		return nil, errors.New("invalid repo: id is 0")
	}

	l.Info().Msg("handling event for repository")

	return sendEvaluateRepoMessage(event.GetRepo(), events.TopicQueueRefreshEntityAndEvaluate)
}

func sendEvaluateRepoMessage(
	repo *repo,
	handler string,
) (*processingResult, error) {
	lookByProps, err := properties.NewProperties(map[string]any{
		// the PropertyUpstreamID is always a string
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(repo.GetID()),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository properties: %w", err)
	}

	entRefresh := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntity(pb.Entity_ENTITY_REPOSITORIES, lookByProps).
		WithProviderImplementsHint(string(db.ProviderTypeGithub))

	return &processingResult{
			topic:   handler,
			wrapper: entRefresh},
		nil
}

func processRelevantRepositoryEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *repoEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetRepo() == nil {
		return nil, errRepoNotFound
	}

	l := zerolog.Ctx(ctx).With().
		Str("github-event-action", event.GetAction()).
		Int64("github-repository-id", event.GetRepo().GetID()).
		Str("github-repository-url", event.GetRepo().GetHTMLURL()).
		Logger()

	if event.GetRepo().GetID() == 0 {
		return nil, errors.New("invalid repo: id is 0")
	}

	l.Info().Msg("handling event for repository")

	lookByProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.GetRepo().GetID()),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository lookup properties: %w", err)
	}

	msg := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntity(pb.Entity_ENTITY_REPOSITORIES, lookByProps).
		WithProviderImplementsHint(string(db.ProviderTypeGithub))

	// This only makes sense for "meta" event type
	if event.GetHookID() != 0 {
		// Check if the payload webhook ID matches the one we
		// have stored in the DB for this repository
		// If not, this means we got a deleted event for a
		// webhook ID that doesn't correspond to the
		// one we have stored in the DB.
		matchHookProps, err := properties.NewProperties(map[string]any{
			ghprop.RepoPropertyHookId: event.GetHookID(),
		})
		if err != nil {
			return nil, fmt.Errorf("error creating hook match properties: %w", err)
		}
		msg = msg.WithMatchProps(matchHookProps)
	}

	// For all other events exept deletions we issue a refresh event.
	topic := events.TopicQueueRefreshEntityAndEvaluate

	// For webhook deletions, repository deletions, and repository
	// transfers, we issue a delete event with the correct message
	// type.
	if event.GetAction() == webhookActionEventDeleted ||
		event.GetAction() == webhookActionEventTransferred {
		topic = events.TopicQueueGetEntityAndDelete
	}

	return &processingResult{
		topic:   topic,
		wrapper: msg,
	}, nil
}
