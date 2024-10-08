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

func (m *providerClassManager) handleRepoPush(l zerolog.Logger, r *http.Request) error {
	l.Debug().Msg("handling push event")

	pushEvent := gitlablib.PushEvent{}
	if err := decodeJSONSafe(r.Body, &pushEvent); err != nil {
		l.Error().Err(err).Msg("error decoding push event")
		return fmt.Errorf("error decoding push event: %w", err)
	}

	rawID := pushEvent.ProjectID
	if rawID == 0 {
		l.Error().Msg("push event missing project ID")
		return fmt.Errorf("push event missing project ID")
	}

	return m.publishRefreshAndEvalForGitlabProject(l, rawID)
}

func (m *providerClassManager) handleTagPush(l zerolog.Logger, r *http.Request) error {
	l.Debug().Msg("handling tag push event")

	tagPushEvent := gitlablib.TagEvent{}
	if err := decodeJSONSafe(r.Body, &tagPushEvent); err != nil {
		l.Error().Err(err).Msg("error decoding tag push event")
		return fmt.Errorf("error decoding tag push event: %w", err)
	}

	rawID := tagPushEvent.ProjectID
	if rawID == 0 {
		l.Error().Msg("tag push event missing project ID")
		return fmt.Errorf("tag push event missing project ID")
	}

	return m.publishRefreshAndEvalForGitlabProject(l, rawID)
}

func (m *providerClassManager) publishRefreshAndEvalForGitlabProject(
	l zerolog.Logger, rawProjectID int) error {
	upstreamID := gitlab.FormatRepositoryUpstreamID(rawProjectID)

	// Form identifying properties
	identifyingProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: upstreamID,
	})
	if err != nil {
		l.Error().Err(err).Msg("error creating identifying properties")
		return fmt.Errorf("error creating identifying properties: %w", err)
	}

	// Form message to publish
	outm := entmsg.NewEntityRefreshAndDoMessage()
	outm.WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, identifyingProps)
	outm.WithProviderClassHint(gitlab.Class)

	// Convert message for publishing
	msgID := uuid.New().String()
	msg := message.NewMessage(msgID, nil)
	if err := outm.ToMessage(msg); err != nil {
		l.Error().Err(err).Msg("error converting message to protobuf")
		return fmt.Errorf("error converting message to protobuf: %w", err)
	}

	// Publish message
	l.Debug().Str("msg_id", msgID).Msg("publishing refresh and eval message")
	if err := m.pub.Publish(events.TopicQueueRefreshEntityAndEvaluate, msg); err != nil {
		l.Error().Err(err).Msg("error publishing refresh and eval message")
		return fmt.Errorf("error publishing refresh and eval message: %w", err)
	}

	return nil
}
