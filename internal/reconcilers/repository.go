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
	"github.com/rs/zerolog/log"

	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/reconcilers/messages"
)

// handleRepoReconcilerEvent handles events coming from the reconciler topic
func (r *Reconciler) handleRepoReconcilerEvent(msg *message.Message) error {
	var evt messages.RepoReconcilerEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error unmarshalling event: %v", err)
		return nil
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(&evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error validating event: %v", err)
		return nil
	}

	ctx := msg.Context()
	log.Printf("handling reconciler event for project %s and repository %s", evt.Project.String(), evt.EntityID.String())
	return r.handleRepositoryReconcilerEvent(ctx, &evt)
}

// HandleArtifactsReconcilerEvent recreates the artifacts belonging to
// an specific repository
// nolint: gocyclo
func (r *Reconciler) handleRepositoryReconcilerEvent(ctx context.Context, evt *messages.RepoReconcilerEvent) error {
	entRefresh := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntityID(evt.EntityID)

	m := message.NewMessage(uuid.New().String(), nil)
	if err := entRefresh.ToMessage(m); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error marshalling message")
		// no point in retrying, so we return nil
		return nil
	}

	if evt.EntityID == uuid.Nil {
		// this might happen if we process old messages during an upgrade, but there's no point in retrying
		zerolog.Ctx(ctx).Error().Msg("entityID is nil")
		return nil
	}

	m.SetContext(ctx)
	if err := r.evt.Publish(events.TopicQueueRefreshEntityByIDAndEvaluate, m); err != nil {
		// we retry in case watermill is having a bad day
		return fmt.Errorf("error publishing message: %w", err)
	}

	return nil
}
