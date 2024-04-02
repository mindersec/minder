//
// Copyright 2023 Stacklok, Inc.
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

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/projects"
)

const (
	eventFetchInterval     = "@daily"
	deleteAccountEventType = "DELETE_ACCOUNT"
)

// AccountEvent is an event returned by the identity provider
type AccountEvent struct {
	Time     int64  `json:"time"`
	Type     string `json:"type"`
	RealmId  string `json:"realmId"`
	ClientId string `json:"clientId"`
	UserId   string `json:"userId"`
}

// SubscribeToIdentityEvents starts a cron job that periodically fetches events from the identity provider
func SubscribeToIdentityEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	cfg *serverconfig.Config,
) error {
	c := cron.New()
	_, err := c.AddFunc(eventFetchInterval, func() {
		HandleEvents(ctx, store, authzClient, cfg)
	})
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

// HandleEvents fetches events from the identity provider and performs any related changes to the minder database
func HandleEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	cfg *serverconfig.Config,
) {
	d := time.Now().Add(time.Duration(10) * time.Minute)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	resp, err := cfg.Identity.Server.Do(ctx, "GET", "admin/realms/stacklok/events", nil, nil)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chron: error getting events: %v", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zerolog.Ctx(ctx).Error().Msgf("events chron: unexpected status code when fetching events: %d", resp.StatusCode)
		return
	}

	var events []AccountEvent
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chron: error decoding events: %v", err)
		return
	}
	for _, event := range events {
		if event.Type == deleteAccountEventType {
			err := DeleteUser(ctx, store, authzClient, event.UserId)
			zerolog.Ctx(ctx).Error().Msgf("events chron: error deleting user account: %v", err)
		}
	}
}

// DeleteUser deletes a user and all their associated data from the minder database
func DeleteUser(ctx context.Context, store db.Store, authzClient authz.Client, userId string) (retErr error) {
	l := zerolog.Ctx(ctx).With().
		Str("operation", "delete").
		Str("subject", userId).
		Logger()

	tx, err := store.BeginTransaction()
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			if err := store.Rollback(tx); err != nil && !errors.Is(err, sql.ErrTxDone) {
				l.Debug().Msgf("error rolling back transaction: %v", err)
			}
		}
	}()
	qtx := store.GetQuerierWithTransaction(tx)

	var userDBID *int32

	usr, err := qtx.GetUserBySubject(ctx, userId)
	// If the user doesn't exist, we still want to clean up any associated data
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error retrieving user %v", err)
	} else if err == nil {
		userDBID = &usr.ID
		l = l.With().Int32("user_id", usr.ID).Logger()
	}

	// Fetching the projects for user before the deletion was made.
	// This allows us to clean up the project from the database
	// if there are no more role assignments for the project.
	projs, err := authzClient.ProjectsForUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("error getting projects for user %v", err)
	}

	l.Debug().Msg("deleting user from authorization system")
	// We delete the user from the authorization system first
	if err := authzClient.DeleteUser(ctx, userId); err != nil {
		return fmt.Errorf("error deleting authorization tuple %v", err)
	}

	// We only delete the user if it still exists in the database
	if userDBID != nil {
		l.Debug().Msg("deleting user from database")
		if err := qtx.DeleteUser(ctx, *userDBID); err != nil {
			return fmt.Errorf("error deleting user %v", err)
		}
	}

	for _, proj := range projs {
		l.Debug().Str("project_id", proj.String()).Msg("cleaning up project")
		if err := projects.CleanUpUnmanagedProjects(ctx, proj, qtx, authzClient, l); err != nil {
			return fmt.Errorf("error deleting project %v", err)
		}
	}

	// organizations will be cleaned up in a migration after this change

	l.Debug().Msg("committing account deletion")
	if err = store.Commit(tx); err != nil {
		return fmt.Errorf("error committing account deletion %w", err)
	}
	return nil
}
