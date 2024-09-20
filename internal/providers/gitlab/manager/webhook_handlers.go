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
	"encoding/json"
	"fmt"
	"io"
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

const (
	// MaxBytesLimit is the maximum number of bytes to read from the response body
	// We limit to 1MB to prevent abuse
	MaxBytesLimit int64 = 1 << 20
)

// getWebhookEventDispatcher returns the appropriate webhook event dispatcher for the given event type
// It returns a function that is meant to do the actual handling of the event.
// Note that we pass the request to the handler function, so we don't even try to
// parse the request body here unless it's necessary.
func (m *providerClassManager) getWebhookEventDispatcher(
	eventType gitlablib.EventType,
) func(l zerolog.Logger, r *http.Request) error {
	//nolint:exhaustive // We only handle a subset of the possible events
	switch eventType {
	case gitlablib.EventTypePush:
		return m.handleRepoPush
	case gitlablib.EventTypeTagPush:
		return m.handleTagPush
	default:
		return m.handleNoop
	}
}

// handleNoop is a no-op handler for unhandled webhook events
func (_ *providerClassManager) handleNoop(l zerolog.Logger, _ *http.Request) error {
	l.Debug().Msg("unhandled webhook event")
	return nil
}

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

func decodeJSONSafe[T any](r io.ReadCloser, v *T) error {
	rs := wrapSafe(r)
	defer r.Close()

	dec := json.NewDecoder(rs)
	return dec.Decode(v)
}

// wrapSafe wraps the io.Reader in a LimitReader to prevent abuse
func wrapSafe(r io.Reader) io.Reader {
	return io.LimitReader(r, MaxBytesLimit)
}
