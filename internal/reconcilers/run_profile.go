// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcilers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/events"
)

// ProfileInitEvent is an event that is sent to the reconciler topic
// when a new profile is created. It is used to initialize the profile
// by iterating over all registered entities for the relevant project
// and sending a profile evaluation event for each one.
type ProfileInitEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
}

// NewProfileInitMessage creates a new repos init event
func NewProfileInitMessage(projectID uuid.UUID) (*message.Message, error) {
	evt := &ProfileInitEvent{
		Project: projectID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling init event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	return msg, nil
}

// handleProfileInitEvent handles a profile init event.
// It is responsible for iterating over all registered repositories
// for the project and sending a profile evaluation event for each one.
func (r *Reconciler) handleProfileInitEvent(msg *message.Message) error {
	ctx := msg.Context()

	var evt ProfileInitEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error unmarshalling event")
		return nil
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		zerolog.Ctx(ctx).Error().Err(err).Msg("error validating event")
		return nil
	}

	zerolog.Ctx(ctx).Debug().Msg("handling profile init event")
	if err := r.publishProfileInitEvents(ctx, evt.Project); err != nil {
		// We don't return an error since watermill will retry
		// the message.
		zerolog.Ctx(ctx).Error().Msg("error publishing profile events")
		return nil
	}

	return nil
}

func (r *Reconciler) publishProfileInitEvents(
	ctx context.Context,
	projectID uuid.UUID,
) error {
	ents, err := r.store.GetEntitiesByProjectHierarchy(ctx, []uuid.UUID{projectID})
	if err != nil {
		// we retry in case the database is having a bad day
		return fmt.Errorf("cannot get entities: %w", err)
	}

	for _, ent := range ents {
		entRefresh := entityMessage.NewEntityRefreshAndDoMessage().
			WithEntityID(ent.ID)

		m := message.NewMessage(uuid.New().String(), nil)
		m.SetContext(ctx)

		if err := entRefresh.ToMessage(m); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error marshalling message")
			// no point in retrying, so we return nil
			return nil
		}

		if err := r.evt.Publish(events.TopicQueueRefreshEntityByIDAndEvaluate, m); err != nil {
			// we retry in case watermill is having a bad day
			return fmt.Errorf("error publishing message: %w", err)
		}
	}

	return nil
}
