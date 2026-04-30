// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/github"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// BackfillOrganizations loops through GitHub app providers and ensures an organization entity is tracked for each
func BackfillOrganizations(ctx context.Context, store db.Store) error {
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Starting backfill for Organization entities...")

	provs, err := store.GlobalListProvidersByClass(ctx, db.ProviderClassGithubApp)
	if err != nil {
		return fmt.Errorf("failed to list providers: %w", err)
	}

	count := 0

	for _, prov := range provs {
		login := github.GetGithubAppOwner(prov.Name)

		_, err = db.WithTransaction(store, func(qtx db.ExtendQuerier) (any, error) {
			// Check if organization entity already exists
			_, err := qtx.GetEntityByName(ctx, db.GetEntityByNameParams{
				EntityType: db.EntitiesOrganization,
				Name:       login,
				ProviderID: prov.ID,
				ProjectID:  prov.ProjectID,
			})

			if err == nil {
				return nil, nil // already exists
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			// Entity doesn't exist, create it
			ent, err := qtx.CreateEntity(ctx, db.CreateEntityParams{
				EntityType: db.EntitiesOrganization,
				Name:       login,
				ProviderID: prov.ID,
				ProjectID:  prov.ProjectID,
			})
			if err != nil {
				return nil, err
			}

			// Set the default property (login name)
			propVal := map[string]any{
				"minder.internal.type":  "string",
				"minder.internal.value": login,
			}
			propBytes, _ := json.Marshal(propVal)

			_, err = qtx.UpsertProperty(ctx, db.UpsertPropertyParams{
				EntityID: ent.ID,
				Key:      properties.PropertyName,
				Value:    propBytes,
			})

			if err == nil {
				count++
			}
			return nil, err
		})

		if err != nil {
			l.Error().Err(err).Str("provider", prov.ID.String()).Msg("Failed to backfill organization for provider")
		}
	}

	l.Info().Int("count", count).Msg("Completed backfill for Organization entities")
	return nil
}
