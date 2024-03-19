// Copyright 2024 Stacklok, Inc
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

// Package github contains logic relating to the management of github repos
package github

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	ghprovider "github.com/stacklok/minder/internal/providers/github/oauth"
	ghclient "github.com/stacklok/minder/internal/repositories/github/clients"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
)

// RepositoryService encapsulates logic related to registering and deleting repos
type RepositoryService interface {
	// DeleteRepositoryByName removes the webhook and deletes the repo from the
	// database. The repo is identified by its name and project.
	DeleteRepositoryByName(
		ctx context.Context,
		client ghclient.GitHubRepoClient,
		projectID uuid.UUID,
		repoOwner string,
		repoName string,
	) error
	// DeleteRepositoryByID removes the webhook and deletes the repo from the
	// database. The repo is identified by its database ID and project.
	DeleteRepositoryByID(
		ctx context.Context,
		client ghclient.GitHubRepoClient,
		projectID uuid.UUID,
		repoID uuid.UUID,
	) error
}

type repositoryService struct {
	webhookManager webhooks.WebhookManager
	store          db.Store
	eventProducer  events.Interface
}

// NewRepositoryService creates an instance of the RepositoryService interface
func NewRepositoryService(
	webhookManager webhooks.WebhookManager,
	store db.Store,
	eventProducer events.Interface,
) RepositoryService {
	return &repositoryService{
		webhookManager: webhookManager,
		store:          store,
		eventProducer:  eventProducer,
	}
}

func (r *repositoryService) DeleteRepositoryByName(
	ctx context.Context,
	client ghclient.GitHubRepoClient,
	projectID uuid.UUID,
	repoOwner string,
	repoName string,
) error {
	// assumption: provider name should always be `ghprovider.Github` for this Github-specific code
	params := db.GetRepositoryByRepoNameParams{
		Provider:  ghprovider.Github,
		RepoOwner: repoOwner,
		RepoName:  repoName,
		ProjectID: projectID,
	}
	repo, err := r.store.GetRepositoryByRepoName(ctx, params)
	if err != nil {
		return err
	}
	return r.deleteRepository(ctx, client, &repo)
}

func (r *repositoryService) DeleteRepositoryByID(
	ctx context.Context,
	client ghclient.GitHubRepoClient,
	projectID uuid.UUID,
	repoID uuid.UUID,
) error {
	repo, err := r.store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        repoID,
		ProjectID: projectID,
	})
	if err != nil {
		log.Printf("error is %v", err)
		return err
	}
	return r.deleteRepository(ctx, client, &repo)
}

func (r *repositoryService) deleteRepository(ctx context.Context, client ghclient.GitHubRepoClient, repo *db.Repository) error {
	var err error

	// Cleanup any webhook we created for this project
	// `webhooksManager.DeleteWebhook` is idempotent, so there are no issues
	// re-running this if we have to retry the request.
	webhookID := repo.WebhookID
	if webhookID.Valid {
		err = r.webhookManager.DeleteWebhook(ctx, client, repo.RepoOwner, repo.RepoName, webhookID.Int64)
		if err != nil {
			return fmt.Errorf("error creating webhook from: %w", err)
		}
	}

	// then remove the entry in the DB
	if err = r.store.DeleteRepository(ctx, repo.ID); err != nil {
		return fmt.Errorf("error deleting repository from DB: %w", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = repo.Provider
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

	return nil
}
