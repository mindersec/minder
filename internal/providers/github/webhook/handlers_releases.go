// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

type releaseEvent struct {
	Action  string  `json:"action,omitempty"`
	Release release `json:"release,omitempty"`
	Repo    repo    `json:"repository,omitempty"`
}

type release struct {
	ID      int64  `json:"id,omitempty"`
	TagName string `json:"tag_name,omitempty"`
	Target  string `json:"target_commitish,omitempty"`
}

func processReleaseEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event releaseEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal release event: %w", err)
	}

	if event.Action == "" {
		return nil, errors.New("release event action not found")
	}

	if event.Release.Target == "" {
		return nil, errors.New("release event target not found")
	}

	if event.Repo.ID == 0 {
		return nil, errors.New("release event repository not found")
	}

	if event.Release.TagName == "" {
		return nil, errors.New("release event tag name not found")
	}

	return sendReleaseEvent(ctx, event), nil
}

func sendReleaseEvent(
	_ context.Context,
	event releaseEvent,
) *processingResult {
	lookByProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.Release.ID),
		ghprop.ReleasePropertyOwner:   event.Repo.GetOwner(),
		ghprop.ReleasePropertyRepo:    event.Repo.Name,
	})

	originatorProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.Repo.ID),
	})

	switch event.Action {
	case "published":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityAdd,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}
	case "unpublished", "deleted":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityDelete,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}
	case "edited":
		return &processingResult{
			topic: constants.TopicQueueRefreshEntityAndEvaluate,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}
	}
	return nil
}
