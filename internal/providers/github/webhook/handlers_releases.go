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
	Action  *string  `json:"action,omitempty"`
	Release *release `json:"release,omitempty"`
	Repo    *repo    `json:"repository,omitempty"`
}

func (r *releaseEvent) GetAction() string {
	if r.Action != nil {
		return *r.Action
	}
	return ""
}

func (r *releaseEvent) GetRelease() *release {
	return r.Release
}

func (r *releaseEvent) GetRepo() *repo {
	return r.Repo
}

type release struct {
	ID      *int64  `json:"id,omitempty"`
	TagName *string `json:"tag_name,omitempty"`
	Target  *string `json:"target_commitish,omitempty"`
}

func (r *release) GetID() int64 {
	if r.ID != nil {
		return *r.ID
	}
	return 0
}

func (r *release) GetTagName() string {
	if r.TagName != nil {
		return *r.TagName
	}
	return ""
}

func (r *release) GetTarget() string {
	if r.Target != nil {
		return *r.Target
	}
	return ""
}

func processReleaseEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *releaseEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal release event: %w", err)
	}

	if event.GetAction() == "" {
		return nil, errors.New("release event action not found")
	}

	if event.GetRelease() == nil {
		return nil, errors.New("release event release not found")
	}

	if event.GetRepo() == nil {
		return nil, errors.New("release event repository not found")
	}

	if event.GetRelease().GetTagName() == "" {
		return nil, errors.New("release event tag name not found")
	}

	if event.GetRelease().GetTarget() == "" {
		return nil, errors.New("release event target not found")
	}

	return sendReleaseEvent(ctx, event)
}

func sendReleaseEvent(
	_ context.Context,
	event *releaseEvent,
) (*processingResult, error) {
	lookByProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.GetRelease().GetID()),
		ghprop.ReleasePropertyOwner:   event.GetRepo().GetOwner(),
		ghprop.ReleasePropertyRepo:    event.GetRepo().GetName(),
	})

	originatorProps := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.GetRepo().GetID()),
	})

	switch event.GetAction() {
	case "published":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityAdd,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}, nil
	case "unpublished", "deleted":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityDelete,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}, nil
	case "edited":
		return &processingResult{
			topic: constants.TopicQueueRefreshEntityAndEvaluate,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_RELEASE, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_REPOSITORIES, originatorProps),
		}, nil
	}
	return nil, nil
}
