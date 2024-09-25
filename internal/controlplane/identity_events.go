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
	"net/url"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/projects"
)

const (
	eventFetchInterval     = "@hourly"
	deleteAccountEventType = "DELETE_ACCOUNT"
)

// AccountEvent is an event returned by Keycloak for user events
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
	projectDeleter projects.ProjectDeleter,
) error {
	c := cron.New()
	_, err := c.AddFunc(eventFetchInterval, func() {
		HandleEvents(ctx, store, authzClient, cfg, projectDeleter)
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
	projectDeleter projects.ProjectDeleter,
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
			err := DeleteUser(ctx, store, authzClient, projectDeleter, event.UserId)
			if err != nil {
				zerolog.Ctx(ctx).Error().Msgf("events chron: error deleting user account: %v", err)
			}
		}
	}
}

// AdminEvent is an event returned by Keycloak for admin events -- note the completely different structure
type AdminEvent struct {
	Time          int64  `json:"time"`
	RealmId       string `json:"realmId"`
	OperationType string `json:"operationType"`
	ResourceType  string `json:"resourceType"`
	ResourcePath  string `json:"resourcePath"`
}

// SubscribeToAdminEvents starts a cron job that periodicalyl fetches admin events from Keycloak.
// Users who are deleted through the Keycloak API show up as admin events, not normal identity events.
func SubscribeToAdminEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	cfg *serverconfig.Config,
	projectDeleter projects.ProjectDeleter,
) error {
	c := cron.New()
	_, err := c.AddFunc(eventFetchInterval, func() {
		HandleAdminEvents(ctx, store, authzClient, cfg, projectDeleter)
	})
	if err != nil {
		return err
	}
	c.Start()
	return nil
}

// HandleAdminEvents deletes users where the deletion occurred through the Keycloak API.
func HandleAdminEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	cfg *serverconfig.Config,
	projectDeleter projects.ProjectDeleter,
) {
	d := time.Now().Add(time.Duration(10) * time.Minute)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	query := url.Values{
		"operationTypes": []string{"DELETE"},
		"resourceTypes":  []string{"USER"},
	}
	resp, err := cfg.Identity.Server.Do(ctx, "GET", "admin/realms/stacklok/admin-events", query, nil)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events cron: error getting admin events: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		zerolog.Ctx(ctx).Error().Msgf("events cron: unexpected status code when fetching admin events: %d", resp.StatusCode)
		return
	}

	var events []AdminEvent
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events cron: error decoding admin events: %v", err)
		return
	}
	for _, event := range events {
		if event.OperationType == "DELETE" && event.ResourceType == "USER" {
			userId := strings.TrimPrefix(event.ResourcePath, "users/")
			err := DeleteUser(ctx, store, authzClient, projectDeleter, userId)
			if err != nil {
				zerolog.Ctx(ctx).Error().Msgf("events cron: error deleting user account from admin event: %v", err)
			}
		}
	}
}

// DeleteUser deletes a user and all their associated data from the minder database
func DeleteUser(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	projectDeleter projects.ProjectDeleter,
	userId string,
) error {
	l := zerolog.Ctx(ctx).With().
		Str("operation", "delete").
		Str("subject", userId).
		Logger()

	// Get the projects the user is associated with. We'll want to clean up any projects
	// that the user is the only member of. Because projects are identified by UUIDs, if
	// we end up deleting the project but for some reason fail to delete the role assignments
	// in openFGA, we'll just end up with dangling role assignments to UUIDs that don't exist.
	projs, err := authzClient.ProjectsForUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("error getting projects for user %v", err)
	}
	l.Debug().Int("projects", len(projs)).Msg("projects for user")

	dbUser, err := db.WithTransaction(store, func(qtx db.ExtendQuerier) (db.User, error) {
		usr, err := qtx.GetUserBySubject(ctx, userId)
		// If the user doesn't exist, we still want to clean up any associated data
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return db.User{}, fmt.Errorf("error retrieving user %v", err)
		}

		for _, proj := range projs {
			l.Debug().Str("project_id", proj.String()).Msg("cleaning up project")
			if err := projectDeleter.CleanUpUnmanagedProjects(l.WithContext(ctx), userId, proj, qtx); err != nil {
				return db.User{}, fmt.Errorf("error deleting project %s: %v", proj.String(), err)
			}
		}

		// We only delete the user if it still exists in the database
		if usr.IdentitySubject != "" {
			l = l.With().Int32("user_id", usr.ID).Logger()
			if err := qtx.DeleteUser(ctx, usr.ID); err != nil {
				return db.User{}, fmt.Errorf("error deleting user %v", err)
			}
		}

		return usr, nil
	})
	if err != nil {
		return err
	}

	l.Debug().Msg("deleting user from authorization system")
	// We delete the user from the authorization system last
	if err := authzClient.DeleteUser(ctx, userId); err != nil {
		return fmt.Errorf("error deleting authorization tuple %v", err)
	}

	zerolog.Ctx(ctx).Info().Str("subject", dbUser.IdentitySubject).Msg("user account deleted")
	return nil
}
