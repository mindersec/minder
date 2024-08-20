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
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/entities"
	minderlogger "github.com/stacklok/minder/internal/logger"
)

//nolint:exhaustive
func (r *Reconciler) handleEntityDeleteEvent(msg *message.Message) error {
	ctx := msg.Context()
	l := zerolog.Ctx(ctx).With().Logger()

	event, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error parsing entity event: %w", err)
	}

	eid, err := event.GetID()
	if err != nil {
		return fmt.Errorf("error getting entity id: %w", err)
	}
	l = zerolog.Ctx(ctx).With().
		Str("provider_id", event.ProviderID.String()).
		Str("project_id", event.ProjectID.String()).
		Str("repo_id", eid.String()).
		Logger()

	// Telemetry logging
	minderlogger.BusinessRecord(ctx).ProviderID = event.ProviderID
	minderlogger.BusinessRecord(ctx).Project = event.ProjectID

	l.Info().Msg("handling entity delete event")
	// Remove the entry in the DB. There's no need to clean any webhook we created for this repository, as GitHub
	// will automatically remove them when the repository is deleted.
	// TODO: Handle other types of entities
	if err := r.repos.DeleteByID(ctx, eid, event.ProjectID); err != nil {
		return fmt.Errorf("error deleting repository from DB: %w", err)
	}

	minderlogger.BusinessRecord(ctx).Repository = eid
	return nil
}
