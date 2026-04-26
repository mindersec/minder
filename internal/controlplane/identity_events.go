// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/projects"
)

const (
	eventFetchInterval     = "@every 5m"
	deleteAccountEventType = "DELETE_ACCOUNT"
)

// SubscribeToIdentityEvents starts a cron job that periodically fetches events from the identity provider
func SubscribeToIdentityEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	idManager auth.IdentityManager,
	projectDeleter projects.ProjectDeleter,
) error {
	c := cron.New()
	_, err := c.AddFunc(eventFetchInterval, func() {
		HandleEvents(ctx, store, authzClient, idManager, projectDeleter)
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
	idManager auth.IdentityManager,
	projectDeleter projects.ProjectDeleter,
) {
	d := time.Now().Add(time.Duration(5) * time.Minute)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	events, err := idManager.GetEvents(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events chrono: error getting events: %v", err)
		return
	}

	for _, event := range events {
		if event.Type == auth.DeleteAccountEvent {
			err := DeleteUser(ctx, store, authzClient, projectDeleter, event.UserId)
			if err != nil {
				zerolog.Ctx(ctx).Error().Msgf("events chrono: error deleting user account: %v", err)
			}
		}
	}
}

// SubscribeToAdminEvents starts a cron job that periodicalyl fetches admin events from Keycloak.
// Users who are deleted through the Keycloak API show up as admin events, not normal identity events.
func SubscribeToAdminEvents(
	ctx context.Context,
	store db.Store,
	authzClient authz.Client,
	idManager auth.IdentityManager,
	projectDeleter projects.ProjectDeleter,
) error {
	c := cron.New()
	_, err := c.AddFunc(eventFetchInterval, func() {
		HandleAdminEvents(ctx, store, authzClient, idManager, projectDeleter)
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
	idManager auth.IdentityManager,
	projectDeleter projects.ProjectDeleter,
) {
	d := time.Now().Add(time.Duration(5) * time.Minute)
	ctx, cancel := context.WithDeadline(ctx, d)
	defer cancel()

	events, err := idManager.GetAdminEvents(ctx, []string{"DELETE"}, []string{"USER"})
	if err != nil {
		zerolog.Ctx(ctx).Error().Msgf("events cron: error getting admin events: %v", err)
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
