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
	"github.com/mindersec/minder/internal/entities/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

type organizationEvent struct {
	Action       *string       `json:"action,omitempty"`
	Organization *organization `json:"release,omitempty"`
}

func (r *organizationEvent) GetAction() string {
	if r.Action != nil {
		return *r.Action
	}
	return ""
}

type organization struct {
	ID   *int64 `json:"id,omitempty"`
	Name string `json:"login,omitempty"`
}

func (r *organization) GetID() int64 {
	if r != nil && r.ID != nil {
		return *r.ID
	}
	return 0
}

func processOrganizationEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *organizationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal release event: %w", err)
	}

	if event.GetAction() == "" {
		return nil, errors.New("release event action not found")
	}

	if event.Organization.GetID() == 0 {
		return nil, errors.New("release event ID not found")
	}

	return sendOrganizationEvent(ctx, event)
}

func sendOrganizationEvent(
	_ context.Context,
	event *organizationEvent,
) (*processingResult, error) {
	lookByProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.Organization.GetID()),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating organization properties: %w", err)
	}

	originatorProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: event.Organization.GetID(),
	})
	if err != nil {
		return nil, fmt.Errorf("error org properties for originator: %w", err)
	}

	switch event.GetAction() {
	case "created":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityAdd,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_ORGANIZATION, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_ORGANIZATION, originatorProps),
		}, nil
	case "deleted":
		return &processingResult{
			topic: constants.TopicQueueOriginatingEntityDelete,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_ORGANIZATION, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_ORGANIZATION, originatorProps),
		}, nil
	case "renamed":
		return &processingResult{
			topic: constants.TopicQueueRefreshEntityAndEvaluate,
			wrapper: entityMessage.NewEntityRefreshAndDoMessage().
				WithEntity(pb.Entity_ENTITY_ORGANIZATION, lookByProps).
				WithProviderImplementsHint(string(db.ProviderTypeGithub)).
				WithOriginator(pb.Entity_ENTITY_ORGANIZATION, originatorProps),
		}, nil
	}
	return nil, nil
}
