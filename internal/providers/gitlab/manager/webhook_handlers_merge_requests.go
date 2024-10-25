// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	gitlablib "github.com/xanzy/go-gitlab"

	entmsg "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/providers/gitlab"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

func (m *providerClassManager) handleMergeRequest(l zerolog.Logger, r *http.Request) error {
	l.Debug().Msg("handling merge request event")

	mergeRequestEvent := gitlablib.MergeEvent{}
	if err := decodeJSONSafe(r.Body, &mergeRequestEvent); err != nil {
		l.Error().Err(err).Msg("error decoding merge request event")
		return fmt.Errorf("error decoding merge request event: %w", err)
	}

	mrID := mergeRequestEvent.ObjectAttributes.ID
	if mrID == 0 {
		return fmt.Errorf("merge request event missing ID")
	}

	mrIID := mergeRequestEvent.ObjectAttributes.IID
	if mrIID == 0 {
		return fmt.Errorf("merge request event missing IID")
	}

	rawProjectID := mergeRequestEvent.Project.ID
	if rawProjectID == 0 {
		return fmt.Errorf("merge request event missing project ID")
	}

	switch {
	case mergeRequestEvent.ObjectAttributes.Action == "open",
		mergeRequestEvent.ObjectAttributes.Action == "reopen":
		return m.publishMergeRequestMessage(mrID, mrIID, rawProjectID,
			constants.TopicQueueOriginatingEntityAdd)
	case mergeRequestEvent.ObjectAttributes.Action == "close":
		return m.publishMergeRequestMessage(mrID, mrIID, rawProjectID,
			constants.TopicQueueOriginatingEntityDelete)
	case mergeRequestEvent.ObjectAttributes.Action == "update":
		return m.publishMergeRequestMessage(mrID, mrIID, rawProjectID,
			constants.TopicQueueRefreshEntityAndEvaluate)
	default:
		return nil
	}
}

func (m *providerClassManager) publishMergeRequestMessage(
	mrID, mrIID, rawProjectID int, queueTopic string) error {
	mrUpstreamID := gitlab.FormatPullRequestUpstreamID(mrID)
	mrformattedIID := gitlab.FormatPullRequestUpstreamID(mrIID)
	mrProjectID := gitlab.FormatRepositoryUpstreamID(rawProjectID)

	// Form identifying properties
	identifyingProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: mrUpstreamID,
		gitlab.PullRequestNumber:      mrformattedIID,
		gitlab.PullRequestProjectID:   mrProjectID,
	})
	if err != nil {
		return fmt.Errorf("error creating identifying properties: %w", err)
	}

	repoIdentifyingProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: mrProjectID,
	})
	if err != nil {
		return fmt.Errorf("error creating repo identifying properties: %w", err)
	}

	// Form message to publish
	outm := entmsg.NewEntityRefreshAndDoMessage()
	outm.WithEntity(minderv1.Entity_ENTITY_PULL_REQUESTS, identifyingProps)
	outm.WithOriginator(minderv1.Entity_ENTITY_REPOSITORIES, repoIdentifyingProps)
	outm.WithProviderClassHint(gitlab.Class)

	// Convert message for publishing
	msgID := uuid.New().String()
	msg := message.NewMessage(msgID, nil)
	if err := outm.ToMessage(msg); err != nil {
		return fmt.Errorf("error converting message to protobuf: %w", err)
	}

	// Publish message
	if err := m.pub.Publish(queueTopic, msg); err != nil {
		return fmt.Errorf("error publishing refresh and eval message: %w", err)
	}

	return nil
}
