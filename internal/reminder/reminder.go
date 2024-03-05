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

// Package reminder sends reminders to the minder server to process entities in background.
package reminder

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/util/sets"

	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/reminderevents"
	"github.com/stacklok/minder/internal/util"
)

func init() {
	gob.Register(map[projectProviderPair]string{})
}

// Interface is an interface over the reminder service
type Interface interface {
	// Start starts the reminder by sending reminders at regular intervals
	Start(ctx context.Context) error

	// Stop stops the reminder service
	Stop()
}

// reminder sends reminders to the minder server to process entities in background.
type reminder struct {
	store    db.Store
	cfg      *reminderconfig.Config
	stop     chan struct{}
	stopOnce sync.Once

	projectListCursor string
	repoListCursor    map[projectProviderPair]string

	ticker *time.Ticker
	logger zerolog.Logger

	eventPublisher message.Publisher
	eventDBCloser  driverCloser
}

type projectProviderPair struct {
	// Exported for gob

	ProjectId uuid.UUID
	Provider  string
}

// NewReminder creates a new reminder instance
func NewReminder(store db.Store, config *reminderconfig.Config) (Interface, error) {
	level := util.ViperLogLevelToZerologLevel(config.LoggingLevel)
	r := &reminder{
		store:          store,
		cfg:            config,
		stop:           make(chan struct{}),
		repoListCursor: make(map[projectProviderPair]string),
		logger:         zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger(),
	}
	err := r.restoreCursorState()
	if err != nil {
		// Non-fatal error, if we can't restore the cursor state, we'll start from scratch.
		r.logger.Error().Err(err).Msg("error restoring cursor state")
	}

	pub, cl, err := r.setupSQLPublisher(context.Background())
	if err != nil {
		return nil, err
	}

	r.eventPublisher = pub
	r.eventDBCloser = cl
	return r, nil
}

// Start starts the reminder by sending reminders at regular intervals
func (r *reminder) Start(ctx context.Context) error {
	select {
	case <-r.stop:
		return errors.New("reminder stopped, cannot start again")
	default:
	}

	interval, err := time.ParseDuration(r.cfg.RecurrenceConfig.Interval)
	if err != nil {
		return err
	}

	if interval <= 0 {
		return fmt.Errorf("invalid interval: %s", r.cfg.RecurrenceConfig.Interval)
	}

	r.ticker = time.NewTicker(interval)
	defer r.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info().Msg("reminder stopped")
			return nil
		case <-r.stop:
			r.logger.Info().Msg("reminder stopped")
			return nil
		case <-r.ticker.C:
			// In-case sending reminders i.e. iterating over entities consumes more time than the
			// interval, the ticker will adjust the time interval or drop ticks to make up for
			// slow receivers.
			if errs := r.sendReminders(ctx); errs != nil {
				for _, err := range errs {
					r.logger.Error().Err(err).Msg("reconciliation request unsuccessful")
				}
			}
		}
	}
}

// Stop stops the reminder service
func (r *reminder) Stop() {
	if r.ticker != nil {
		defer r.ticker.Stop()
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		r.eventDBCloser()
	})
}

func (r *reminder) restoreCursorState() error {
	if _, err := os.Stat(r.cfg.CursorFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(r.cfg.CursorFile)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	cursorData := make(map[string]interface{})

	if err := dec.Decode(&cursorData); err != nil {
		return err
	}

	r.projectListCursor = cursorData["projectListCursor"].(string)
	r.repoListCursor = cursorData["repoListCursor"].(map[projectProviderPair]string)

	return nil
}

func (r *reminder) sendReminders(ctx context.Context) []error {
	listProjectsResp, err := r.listProjects(ctx, listProjectsRequest{
		cursor: r.projectListCursor,
		limit:  r.cfg.RecurrenceConfig.MinProjectFetchLimit,
	})
	if err != nil {
		return []error{fmt.Errorf("error listing projects: %w", err)}
	}

	r.logger.Info().Msgf("fetched project list of size: %d", len(listProjectsResp.projects))

	// Update the cursor for the next iteration
	r.updateProjectListCursor(listProjectsResp.cursor)

	repos, err := r.getRepositoryBatch(ctx, listProjectsResp.projects)
	if err != nil {
		return []error{fmt.Errorf("error listing projects: %w", err)}
	}

	r.logger.Info().Msgf("created repository batch of size: %d", len(repos))

	errorSlice := make([]error, 0)
	messages := make([]*message.Message, 0, len(repos))

	for _, repo := range repos {
		repoReconcilerMessage, err := reminderevents.NewRepoReminderMessage(repo.Provider,
			repo.RepoID, repo.ProjectID)
		if err != nil {
			errorSlice = append(errorSlice, err)
			continue
		}

		messages = append(messages, repoReconcilerMessage)
	}

	if len(messages) != 0 {
		err = r.eventPublisher.Publish(reminderevents.RepoReminderEventTopic, messages...)
		if err != nil {
			errorSlice = append(errorSlice, fmt.Errorf("error publishing messages: %w", err))
		}
		r.logger.Info().Msgf("sent %d reminders", len(messages))
	}

	return errorSlice
}

func (r *reminder) getRepositoryBatch(ctx context.Context, projects []*db.Project) ([]*db.Repository, error) {
	repos, err := r.getReposForReconciliation(ctx, projects, r.cfg.RecurrenceConfig.MaxPerProject)
	if err != nil {
		return nil, fmt.Errorf("error getting repos for reconciliation: %w", err)
	}

	if len(repos) < r.cfg.RecurrenceConfig.BatchSize {
		remCap := r.cfg.RecurrenceConfig.BatchSize - len(repos)
		r.logger.Debug().Msgf("fetched %d repos, trying to fetch %d additional repos", len(repos), remCap)
		additionalRepos, err := r.getAdditionalReposForReconciliation(ctx, remCap)
		if err != nil {
			return nil, fmt.Errorf("error getting additional repos for reconciliation: %w", err)
		}

		repos = append(repos, additionalRepos...)
	}
	return repos, nil
}

func (r *reminder) getReposForReconciliation(
	ctx context.Context,
	projects []*db.Project,
	fetchLimit int,
) ([]*db.Repository, error) {
	repos := make([]*db.Repository, 0)

	// Instead of querying which providers are registered for a project, we can simply test all
	// providers we support. Performance shouldn't be an issue as ListRepositories endpoint is
	// indexed using provider.
	githubProvider := string(db.ProviderTypeGithub)

	for _, project := range projects {
		cursorKey := projectProviderPair{
			ProjectId: project.ID,
			Provider:  githubProvider,
		}
		listRepoResp, err := r.listRepositories(ctx, listRepoRequest{
			projectId: project.ID,
			provider:  githubProvider,
			limit:     fetchLimit,
			// Use the cursor from the last iteration. If the cursor is empty, it will fetch
			// the first page.
			cursor: r.repoListCursor[cursorKey],
		})

		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug().Msgf("no repositories found for project: %s", project.ID)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error listing repositories: %w", err)
		}

		r.updateRepoListCursor(cursorKey, listRepoResp.cursor)

		eligibleRepos, err := r.getEligibleRepos(ctx, listRepoResp.results)
		if err != nil {
			return nil, fmt.Errorf("error getting eligible repos: %w", err)
		}

		repos = append(repos, eligibleRepos...)
	}
	return repos, nil
}

func (r *reminder) updateRepoListCursor(cursorKey projectProviderPair, newCursor string) {
	r.logger.Debug().
		Str("newCursor", newCursor).
		Str("project", cursorKey.ProjectId.String()).
		Str("provider", cursorKey.Provider).
		Msg("updating repo list cursor")

	// Update the cursor for the next iteration
	r.repoListCursor[cursorKey] = newCursor
	if newCursor == "" {
		// Remove the cursor from the map if it's empty. This keeps the map size in check.
		// Default empty cursor is used to fetch the first page in the subsequent iterations.
		delete(r.repoListCursor, cursorKey)
	}
	err := r.storeCursorState()
	if err != nil {
		r.logger.Error().Err(err).Msg("error storing cursor state")
	}
}

func (r *reminder) getAdditionalReposForReconciliation(ctx context.Context, additionalSpaces int) ([]*db.Repository, error) {
	additionalRepos := make([]*db.Repository, 0, additionalSpaces)

	// TODO: Is this optimization necessary?
	// minProjectsToFetch := math.Ceil(float64(additionalSpaces) / float64(r.cfg.RecurrenceConfig.MaxPerProject))

	// A good assumption is that every project has at least one repository. So even if every
	// project contributed only one repository, we will be able to fetch additionalSpaces number
	// of repositories. This prevents this loop from running indefinitely or for a long time if
	// a lot of projects don't have any repositories.
	roundsRemaining := additionalSpaces

	for additionalSpaces > 0 && roundsRemaining > 0 {
		roundsRemaining--

		// Fetch the next project (one-by-one) to get the repos
		listProjectsResp, err := r.listProjects(ctx, listProjectsRequest{
			cursor: r.projectListCursor,
			limit:  1,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing projects: %w", err)
		}

		fetchLimit := additionalSpaces
		if additionalSpaces >= r.cfg.RecurrenceConfig.MaxPerProject {
			fetchLimit = r.cfg.RecurrenceConfig.MaxPerProject

			// Update the cursor for the next iteration. If additionalSpaces < MaxPerProject, then
			// we will fetch the same project again in the next iteration. This is done to prevent
			// evaluating only small number of entities from a project in an iteration.
			r.updateProjectListCursor(listProjectsResp.cursor)
		}

		repos, err := r.getReposForReconciliation(ctx, listProjectsResp.projects, fetchLimit)
		if err != nil {
			return nil, fmt.Errorf("error getting repos for reconciliation: %w", err)
		}

		additionalSpaces -= len(repos)
		additionalRepos = append(additionalRepos, repos...)
	}

	return additionalRepos, nil
}

func (r *reminder) updateProjectListCursor(newCursor string) {
	r.logger.Debug().Str("newCursor", newCursor).Msg("updating project list cursor")

	r.projectListCursor = newCursor
	err := r.storeCursorState()
	if err != nil {
		r.logger.Error().Err(err).Msg("error storing cursor state")
	}
}

func (r *reminder) storeCursorState() error {
	r.logger.Debug().Msg("storing cursor state")

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	data := map[string]interface{}{
		"projectListCursor": r.projectListCursor,
		"repoListCursor":    r.repoListCursor,
	}
	if err := enc.Encode(data); err != nil {
		return err
	}

	return os.WriteFile(r.cfg.CursorFile, buf.Bytes(), 0600)
}

func (r *reminder) getEligibleRepos(ctx context.Context, repos []*db.Repository) ([]*db.Repository, error) {
	repoIds := make([]uuid.UUID, len(repos))
	for i, repo := range repos {
		repoIds[i] = repo.ID
	}

	eligibleRepoIds, err := r.getEligibleRepoIds(ctx, repoIds)
	if err != nil {
		return nil, err
	}

	eligibleRepos := make([]*db.Repository, 0, len(repos))
	for _, repo := range repos {
		if eligibleRepoIds.Has(repo.ID) {
			eligibleRepos = append(eligibleRepos, repo)
		}
	}

	return eligibleRepos, nil
}

func (r *reminder) getEligibleRepoIds(ctx context.Context, repoId []uuid.UUID) (sets.Set[uuid.UUID], error) {
	// MinElapsed is validated in the config, so we can safely parse it here.
	minElapsed, _ := time.ParseDuration(r.cfg.RecurrenceConfig.MinElapsed)

	oldestRuleEvaluations, err := r.listOldestRuleEvaluationsByIds(ctx, repoId)
	if err != nil {
		return nil, err
	}

	eligibleRepoIds := sets.New[uuid.UUID]()

	// If a repo has no rule evaluations, it is not eligible for reconciliation.
	for _, ruleEvaluation := range oldestRuleEvaluations.results {
		if ruleEvaluation.oldestRuleEvaluation.Add(minElapsed).Before(time.Now()) {
			eligibleRepoIds.Insert(ruleEvaluation.repoId)
		}
	}
	return eligibleRepoIds, nil
}
