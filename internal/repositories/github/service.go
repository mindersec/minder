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
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects/features"
	ghprov "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/manager"
	reconcilers "github.com/stacklok/minder/internal/reconcilers/messages"
	ghclient "github.com/stacklok/minder/internal/repositories/github/clients"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// RepositoryService encapsulates logic related to registering and deleting repos
// TODO: get rid of the github client from this interface
type RepositoryService interface {
	// CreateRepository registers a GitHub repository, including creating
	// a webhook in the repo in GitHub.
	CreateRepository(
		ctx context.Context,
		// TODO: this should just be ProviderID
		// Switch once we get rid of provider names from the repo table
		provider *db.Provider,
		projectID uuid.UUID,
		repoName string,
		repoOwner string,
	) (*pb.Repository, error)
	// DeleteByID removes the webhook and deletes the repo from the database.
	DeleteByID(
		ctx context.Context,
		repoID uuid.UUID,
		projectID uuid.UUID,
	) error
	// DeleteByName removes the webhook and deletes the repo from the database.
	// Ideally, we would take provider ID instead of name. Name is used for
	// backwards compatibility with the API endpoint which calls it.
	DeleteByName(
		ctx context.Context,
		repoOwner string,
		repoName string,
		projectID uuid.UUID,
		providerName string,
	) error

	// ListRepositories retrieves all repositories for the
	// specific provider and project. Ideally, we would take
	// provider ID instead of name. Name is used for backwards
	// compatibility with the API endpoint which calls it.
	ListRepositories(
		ctx context.Context,
		projectID uuid.UUID,
		providerName string,
	) ([]db.Repository, error)

	// GetRepositoryById retrieves a repository by its ID and project.
	GetRepositoryById(ctx context.Context, repositoryID uuid.UUID, projectID uuid.UUID) (db.Repository, error)
	// GetRepositoryByName retrieves a repository by its name, owner, project and provider (if specified).
	GetRepositoryByName(
		ctx context.Context,
		repoOwner string,
		repoName string,
		projectID uuid.UUID,
		providerName string,
	) (db.Repository, error)
}

var (
	// ErrPrivateRepoForbidden is returned when creation fails due to an
	// attempt to register a private repo in a project which does not allow
	// private repos
	ErrPrivateRepoForbidden = errors.New("private repos cannot be registered in this project")
	// ErrArchivedRepoForbidden is returned when creation fails due to an
	// attempt to register an archived repo
	ErrArchivedRepoForbidden = errors.New("archived repos cannot be registered in this project")
)

type repositoryService struct {
	webhookManager  webhooks.WebhookManager
	store           db.Store
	eventProducer   events.Publisher
	providerManager manager.ProviderManager
}

// NewRepositoryService creates an instance of the RepositoryService interface
func NewRepositoryService(
	webhookManager webhooks.WebhookManager,
	store db.Store,
	eventProducer events.Publisher,
	providerManager manager.ProviderManager,
) RepositoryService {
	return &repositoryService{
		webhookManager:  webhookManager,
		store:           store,
		eventProducer:   eventProducer,
		providerManager: providerManager,
	}
}

func (r *repositoryService) CreateRepository(
	ctx context.Context,
	provider *db.Provider,
	projectID uuid.UUID,
	repoOwner string,
	repoName string,
) (*pb.Repository, error) {
	// instantiate the GitHub client
	p, err := r.providerManager.InstantiateFromID(ctx, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("error instantiating provider: %w", err)
	}

	client, err := provifv1.As[provifv1.GitHub](p)
	if err != nil {
		return nil, fmt.Errorf("error instantiating github client: %w", err)
	}

	propClient, err := provifv1.As[provifv1.PropertiesFetcher](p)
	if err != nil {
		return nil, fmt.Errorf("error instantiating properties client: %w", err)
	}

	repoProperties, err := propClient.FetchAllProperties(
		ctx,
		fmt.Sprintf("%s/%s", repoOwner, repoName),
		pb.Entity_ENTITY_REPOSITORIES)
	if err != nil {
		return nil, fmt.Errorf("error fetching properties for repository: %w", err)
	}

	isArchived, err := repoProperties.GetProperty(properties.RepoPropertyIsArchived).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
	}

	// skip if this is an archived repo
	if isArchived {
		return nil, ErrArchivedRepoForbidden
	}

	isPrivate, err := repoProperties.GetProperty(properties.RepoPropertyIsPrivate).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
	}

	// skip if this is a private repo, and private repos are not enabled
	if isPrivate && !features.ProjectAllowsPrivateRepos(ctx, r.store, projectID) {
		return nil, ErrPrivateRepoForbidden
	}

	// create a webhook to capture events from the repository
	hookUUID, githubHook, err := r.webhookManager.CreateWebhook(ctx, client, repoOwner, repoName)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook in repo: %w", err)
	}

	// insert the repository into the DB
	dbID, pbRepo, err := r.persistRepository(
		ctx,
		repoProperties,
		githubHook,
		hookUUID,
		projectID,
		provider,
	)
	if err != nil {
		log.Printf("error creating repository '%s/%s' in database: %v", repoOwner, repoName, err)
		// Attempt to clean up the webhook we created earlier. This is a
		// best-effort attempt: If it fails, the customer either has to delete
		// the hook manually, or it will be deleted the next time the customer
		// attempts to register a repo.
		cleanupErr := r.webhookManager.DeleteWebhook(ctx, client, repoOwner, repoName, *githubHook.ID)
		if cleanupErr != nil {
			log.Printf("error deleting new webhook: %v", cleanupErr)
		}
		return nil, fmt.Errorf("error creating repository in database: %w", err)
	}

	// publish a reconciling event for the registered repositories
	if err = r.pushReconcilerEvent(pbRepo, projectID, provider.ID); err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = provider.ID
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Repository = dbID

	return pbRepo, nil
}

func (r *repositoryService) ListRepositories(
	ctx context.Context,
	projectID uuid.UUID,
	providerName string,
) ([]db.Repository, error) {
	return r.store.ListRepositoriesByProjectID(
		ctx,
		db.ListRepositoriesByProjectIDParams{
			ProjectID: projectID,
			Provider: sql.NullString{
				String: providerName,
				Valid:  providerName != "",
			},
		},
	)
}

func (r *repositoryService) GetRepositoryById(
	ctx context.Context,
	repositoryID uuid.UUID,
	projectID uuid.UUID,
) (db.Repository, error) {
	return r.store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        repositoryID,
		ProjectID: projectID,
	})
}

func (r *repositoryService) GetRepositoryByName(
	ctx context.Context,
	repoOwner string,
	repoName string,
	projectID uuid.UUID,
	providerName string,
) (db.Repository, error) {
	providerFilter := sql.NullString{
		String: providerName,
		Valid:  providerName != "",
	}
	params := db.GetRepositoryByRepoNameParams{
		Provider:  providerFilter,
		RepoOwner: repoOwner,
		RepoName:  repoName,
		ProjectID: projectID,
	}
	return r.store.GetRepositoryByRepoName(ctx, params)
}

func (r *repositoryService) DeleteByID(ctx context.Context, repositoryID uuid.UUID, projectID uuid.UUID) error {
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Repository = repositoryID

	repo, err := r.GetRepositoryById(ctx, repositoryID, projectID)
	if err != nil {
		return fmt.Errorf("error retrieving repository %s in project %s: %w", repositoryID, projectID, err)
	}

	logger.BusinessRecord(ctx).ProviderID = repo.ProviderID

	client, err := r.instantiateGithubProvider(ctx, repo.ProviderID)
	if err != nil {
		return err
	}

	return r.deleteRepository(ctx, client, &repo)
}

func (r *repositoryService) DeleteByName(
	ctx context.Context,
	repoOwner string,
	repoName string,
	projectID uuid.UUID,
	providerName string,
) error {
	logger.BusinessRecord(ctx).Project = projectID

	repo, err := r.store.GetRepositoryByRepoName(ctx, db.GetRepositoryByRepoNameParams{
		RepoOwner: repoOwner,
		RepoName:  repoName,
		ProjectID: projectID,
		Provider: sql.NullString{
			String: providerName,
			Valid:  providerName != "",
		},
	})
	if err != nil {
		return fmt.Errorf("error retrieving repository %s/%s in project %s: %w", repoOwner, repoName, projectID, err)
	}

	logger.BusinessRecord(ctx).Repository = repo.ID

	client, err := r.instantiateGithubProvider(ctx, repo.ProviderID)
	if err != nil {
		return err
	}

	return r.deleteRepository(ctx, client, &repo)
}

func (r *repositoryService) instantiateGithubProvider(ctx context.Context, providerID uuid.UUID) (provifv1.GitHub, error) {
	provider, err := r.providerManager.InstantiateFromID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("error while instantiating provider: %w", err)
	}

	gh, err := provifv1.As[provifv1.GitHub](provider)
	if err != nil {
		return nil, fmt.Errorf("error while instantiating provider: %w", err)
	}

	return gh, nil
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
			return fmt.Errorf("error deleting webhook: %w", err)
		}
	}

	_, err = db.WithTransaction(r.store, func(t db.ExtendQuerier) (*pb.Repository, error) {
		// then remove the entry in the DB
		if err := t.DeleteRepository(ctx, repo.ID); err != nil {
			return nil, fmt.Errorf("error deleting repository from DB: %w", err)
		}

		if err := t.DeleteEntity(ctx, db.DeleteEntityParams{
			ID:        repo.ID,
			ProjectID: repo.ProjectID,
		}); err != nil {
			return nil, fmt.Errorf("error deleting entity from DB: %w", err)
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("error deleting repository: %w", err)
	}

	return nil
}

func (r *repositoryService) pushReconcilerEvent(pbRepo *pb.Repository, projectID uuid.UUID, providerID uuid.UUID) error {
	log.Printf("publishing register event for repository: %s/%s", pbRepo.Owner, pbRepo.Name)

	msg, err := reconcilers.NewRepoReconcilerMessage(providerID, pbRepo.RepoId, projectID)
	if err != nil {
		return fmt.Errorf("error creating reconciler event: %v", err)
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err = r.eventProducer.Publish(events.TopicQueueReconcileRepoInit, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	return nil
}

// returns DB PK along with protobuf representation of a repo
func (r *repositoryService) persistRepository(
	ctx context.Context,
	repoProperties *properties.Properties,
	githubHook *github.Hook,
	hookUUID string,
	projectID uuid.UUID,
	provider *db.Provider,
) (uuid.UUID, *pb.Repository, error) {
	var outid uuid.UUID
	pbr, err := db.WithTransaction(r.store, func(t db.ExtendQuerier) (*pb.Repository, error) {
		// instantiate the response object
		pbRepo, err := pbRepoFromProperties(repoProperties, githubHook, hookUUID)
		if err != nil {
			return nil, fmt.Errorf("error creating repository object: %w", err)
		}

		License := sql.NullString{}
		if pbRepo.License != "" {
			License.String = pbRepo.License
			License.Valid = true
		}

		// update the database
		dbRepo, err := t.CreateRepository(ctx, db.CreateRepositoryParams{
			Provider:   provider.Name,
			ProviderID: provider.ID,
			ProjectID:  projectID,
			RepoOwner:  pbRepo.Owner,
			RepoName:   pbRepo.Name,
			RepoID:     pbRepo.RepoId,
			IsPrivate:  pbRepo.IsPrivate,
			IsFork:     pbRepo.IsFork,
			WebhookID: sql.NullInt64{
				Int64: pbRepo.HookId,
				Valid: true,
			},
			CloneUrl:   pbRepo.CloneUrl,
			WebhookUrl: pbRepo.HookUrl,
			DeployUrl:  pbRepo.DeployUrl,
			DefaultBranch: sql.NullString{
				String: pbRepo.DefaultBranch,
				Valid:  true,
			},
			License: License,
		})
		if err != nil {
			return pbRepo, err
		}

		outid = dbRepo.ID
		pbRepo.Id = ptr.Ptr(dbRepo.ID.String())

		// TODO: Replace with CreateEntity call
		_, err = t.CreateEntityWithID(ctx, db.CreateEntityWithIDParams{
			ID:         dbRepo.ID,
			EntityType: db.EntitiesRepository,
			Name:       fmt.Sprintf("%s/%s", pbRepo.Owner, pbRepo.Name),
			ProjectID:  projectID,
			ProviderID: provider.ID,
		})

		return pbRepo, err
	})
	if err != nil {
		return uuid.Nil, nil, err
	}

	return outid, pbr, nil
}

func pbRepoFromProperties(
	repoProperties *properties.Properties,
	githubHook *github.Hook,
	hookUUID string,
) (*pb.Repository, error) {
	name, err := repoProperties.GetProperty(ghprov.RepoPropertyName).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching name property: %w", err)
	}

	owner, err := repoProperties.GetProperty(ghprov.RepoPropertyOwner).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching owner property: %w", err)
	}

	repoId, err := repoProperties.GetProperty(ghprov.RepoPropertyId).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error fetching repo_id property: %w", err)
	}

	isPrivate, err := repoProperties.GetProperty(properties.RepoPropertyIsPrivate).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
	}

	isFork, err := repoProperties.GetProperty(properties.RepoPropertyIsFork).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
	}

	pbRepo := &pb.Repository{
		Name:          name,
		Owner:         owner,
		RepoId:        repoId,
		HookId:        githubHook.GetID(),
		HookUrl:       githubHook.GetURL(),
		DeployUrl:     repoProperties.GetProperty(ghprov.RepoPropertyDeployURL).GetString(),
		CloneUrl:      repoProperties.GetProperty(ghprov.RepoPropertyCloneURL).GetString(),
		HookType:      githubHook.GetType(),
		HookName:      githubHook.GetName(),
		HookUuid:      hookUUID,
		IsPrivate:     isPrivate,
		IsFork:        isFork,
		DefaultBranch: repoProperties.GetProperty(ghprov.RepoPropertyDefaultBranch).GetString(),
		License:       repoProperties.GetProperty(ghprov.RepoPropertyLicense).GetString(),
	}

	return pbRepo, nil
}
