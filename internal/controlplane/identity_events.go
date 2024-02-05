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
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
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

	parsedURL, err := url.Parse(cfg.Identity.Server.IssuerUrl)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chron: error parsing issuer URL: %v", err)
		return
	}

	tokenUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/token")

	clientSecret, err := cfg.Identity.Server.GetClientSecret()
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("failed to get client secret: %v", err)
		return
	}

	clientCredentials := clientcredentials.Config{
		ClientID:     cfg.Identity.Server.ClientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl.String(),
	}

	token, err := clientCredentials.Token(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chron: error getting access token: %v", err)
		return
	}

	eventsUrl := parsedURL.JoinPath("admin/realms/stacklok/events")
	request, err := http.NewRequest("GET", eventsUrl.String(), nil)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chron: error constructing events request: %v", err)
		return
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Do(request)
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
func DeleteUser(ctx context.Context, store db.Store, authzClient authz.Client, userId string) error {
	tx, err := store.BeginTransaction()
	if err != nil {
		return err
	}
	defer store.Rollback(tx)
	qtx := store.GetQuerierWithTransaction(tx)

	_, err = qtx.GetUserBySubject(ctx, userId)
	// If the user doesn't exist, we still want to clean up any associated data
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error retrieving user %v", err)
	}

	// Fetching the projects for user before the deletion was made.
	// This allows us to clean up the project from the database
	// if there are no more role assignments for the project.
	projects, err := authzClient.ProjectsForUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("error getting projects for user %v", err)
	}

	// We delete the user from the authorization system first
	if err := authzClient.DeleteUser(ctx, userId); err != nil {
		return fmt.Errorf("error deleting authorization tuple %v", err)
	}

	for _, proj := range projects {
		// Given that we've deleted the user from the authorization system,
		// we can now check if there are any role assignments for the project.
		as, err := authzClient.AssignmentsToProject(ctx, proj)
		if err != nil {
			return fmt.Errorf("error getting role assignments for project %v", err)
		}

		if len(as) == 0 {
			// no role assignments for this project
			// we can safely delete it.
			if _, err := qtx.DeleteProject(ctx, proj); err != nil {
				return fmt.Errorf("error deleting project %v", err)
			}
		}
	}

	// organizations will be cleaned up in a migration after this change

	err = store.Commit(tx)
	if err != nil {
		return fmt.Errorf("error committing account deletion %w", err)
	}
	return nil
}
