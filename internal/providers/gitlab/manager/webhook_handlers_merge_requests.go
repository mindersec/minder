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

package manager

import (
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	gitlablib "github.com/xanzy/go-gitlab"

	entmsg "github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/gitlab"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
			events.TopicQueueOriginatingEntityAdd)
	case mergeRequestEvent.ObjectAttributes.Action == "close":
		return m.publishMergeRequestMessage(mrID, mrIID, rawProjectID,
			events.TopicQueueOriginatingEntityDelete)
	case mergeRequestEvent.ObjectAttributes.Action == "update":
		return m.publishMergeRequestMessage(mrID, mrIID, rawProjectID,
			events.TopicQueueRefreshEntityAndEvaluate)
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
