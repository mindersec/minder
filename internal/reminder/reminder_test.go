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

package reminder

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/db"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
)

func Test_getEligibleRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      []repoLastUpdatedIn
		output     []*db.Repository
		setup      func(store *mockdb.MockStore, input []repoLastUpdatedIn)
		minElapsed string
		err        string
	}{
		{
			name: "all repos eligible",
			input: []repoLastUpdatedIn{
				{repo: getRepoWithId(t, 1), lastUpdated: time.Now().Add(-2 * time.Hour)},
				{repo: getRepoWithId(t, 2), lastUpdated: time.Now().Add(-2 * time.Hour)},
				{repo: getRepoWithId(t, 3), lastUpdated: time.Now().Add(-2 * time.Hour)},
			},
			output:     getRepoTillId(t, 3),
			setup:      standardLastUpdatedSetup,
			minElapsed: "1h",
			err:        "",
		},
		{
			name: "no repos eligible",
			input: []repoLastUpdatedIn{
				{repo: getRepoWithId(t, 1), lastUpdated: time.Now()},
			},
			output:     []*db.Repository{},
			setup:      standardLastUpdatedSetup,
			minElapsed: "1h",
			err:        "",
		},
		{
			name: "error with store",
			input: []repoLastUpdatedIn{
				{repo: getRepoWithId(t, 1), lastUpdated: time.Now().Add(-2 * time.Hour)},
			},
			output: nil,
			setup: func(store *mockdb.MockStore, _ []repoLastUpdatedIn) {
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrConnDone)
			},
			minElapsed: "1h",
			err:        sql.ErrConnDone.Error(),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.input)
			cfg := &reminderconfig.Config{
				RecurrenceConfig: reminderconfig.RecurrenceConfig{
					MinElapsed: test.minElapsed,
				},
			}

			r := createTestReminder(t, store, cfg)

			inputRepos := make([]*db.Repository, 0, len(test.input))
			for _, in := range test.input {
				inputRepos = append(inputRepos, in.repo)
			}

			got, err := r.getEligibleRepos(context.Background(), inputRepos)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.output)
		})
	}
}

func Test_getReposForReconciliation(t *testing.T) {
	t.Parallel()

	type want struct {
		repos          []*db.Repository
		repoListCursor map[projectProviderPair]cursorutil.RepoCursor
	}

	tests := []struct {
		name          string
		projects      map[*db.Project][]*db.Repository
		fetchLimit    int
		recurrenceCfg reminderconfig.RecurrenceConfig
		setup         func(store *mockdb.MockStore, projects map[*db.Project][]*db.Repository)
		want          want
		err           string
	}{
		{
			name: "get repos for reconciliation",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: getRepoTillId(t, 3),
			},
			fetchLimit: 3,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  3,
			},
			setup: func(store *mockdb.MockStore, projects map[*db.Project][]*db.Repository) {
				for project, repos := range projects {
					returnedRepos := make([]db.Repository, len(repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(repos))
					for i := range repos {
						returnedRepos[i] = *repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}
			},
			want: want{
				repos: getRepoTillId(t, 3),
				// Exhausted cursors are deleted to keep size in check
				repoListCursor: map[projectProviderPair]cursorutil.RepoCursor{},
			},
			err: "",
		},
		{
			name: "get repos for reconciliation with additional repos",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: getRepoTillId(t, 2),
			},
			fetchLimit: 2,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  2,
			},
			setup: func(store *mockdb.MockStore, projects map[*db.Project][]*db.Repository) {
				for project, repos := range projects {
					returnedRepos := make([]db.Repository, len(repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(repos))
					for i := range repos {
						returnedRepos[i] = *repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					returnedRepos = append(returnedRepos, *getRepoWithId(t, 3))

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 3, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}
			},
			want: want{
				repos: getRepoTillId(t, 2),
				repoListCursor: map[projectProviderPair]cursorutil.RepoCursor{
					{
						ProjectId: generateUUIDFromNum(t, 1),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 1).String(),
						Provider:  "github",
						RepoId:    3,
					},
				},
			},
			err: "",
		},
		{
			name: "error listing repositories",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: {
					getRepoWithId(t, 1),
				},
			},
			fetchLimit: 3,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  3,
			},
			setup: func(store *mockdb.MockStore, _ map[*db.Project][]*db.Repository) {
				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("some error"))
			},
			want: want{},
			err:  "error listing repositories: some error",
		},
		{
			name: "get repos from multiple projects with additional repos",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: {
					getRepoWithId(t, 1),
				},
				{
					ID:   generateUUIDFromNum(t, 2),
					Name: "project2",
				}: {
					getRepoWithId(t, 2),
				},
			},
			fetchLimit: 1,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  2,
			},
			setup: func(store *mockdb.MockStore, projects map[*db.Project][]*db.Repository) {
				for project, repos := range projects {
					returnedRepos := make([]db.Repository, len(repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(repos))
					for i := range repos {
						returnedRepos[i] = *repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					// Add an additional repo for each project, it doesn't matter if RepoID is same
					returnedRepos = append(returnedRepos, *getRepoWithId(t, 3))

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 2, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}
			},
			want: want{
				repos: getRepoTillId(t, 2),
				repoListCursor: map[projectProviderPair]cursorutil.RepoCursor{
					{
						ProjectId: generateUUIDFromNum(t, 1),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 1).String(),
						Provider:  "github",
						RepoId:    3,
					},
					{
						ProjectId: generateUUIDFromNum(t, 2),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 2).String(),
						Provider:  "github",
						RepoId:    3,
					},
				},
			},
		},
		{
			name: "no repos found for project",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: {},
			},
			fetchLimit: 1,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  2,
			},
			setup: func(store *mockdb.MockStore, _ map[*db.Project][]*db.Repository) {
				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrNoRows)
			},
			want: want{},
			err:  "",
		},
		{
			name: "error getting eligible repos",
			projects: map[*db.Project][]*db.Repository{
				{
					ID:   generateUUIDFromNum(t, 1),
					Name: "project1",
				}: getRepoTillId(t, 3),
			},
			fetchLimit: 3,
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed: "1h",
				BatchSize:  3,
			},
			setup: func(store *mockdb.MockStore, projects map[*db.Project][]*db.Repository) {
				for project, repos := range projects {
					returnedRepos := make([]db.Repository, len(repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(repos))
					for i := range repos {
						returnedRepos[i] = *repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(returnedRepos, nil)
				}

				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrConnDone)
			},
			want: want{},
			err:  fmt.Sprintf("error getting eligible repos: %s", sql.ErrConnDone),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.projects)
			cfg := &reminderconfig.Config{
				RecurrenceConfig: test.recurrenceCfg,
			}

			r := createTestReminder(t, store, cfg)

			projectsSlice := make([]*db.Project, 0, len(test.projects))
			for project := range test.projects {
				projectsSlice = append(projectsSlice, project)
			}

			got, err := r.getReposForReconciliation(context.Background(), projectsSlice, test.fetchLimit)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.want.repos)
			require.Equal(t, len(test.want.repoListCursor), len(r.repoListCursor))
			for k, v := range test.want.repoListCursor {
				require.Equal(t, v.String(), r.repoListCursor[k])
			}
		})
	}
}

// nolint:gocyclo
func Test_getRepositoryBatch(t *testing.T) {
	t.Parallel()

	type want struct {
		repos          []*db.Repository
		repoListCursor map[projectProviderPair]cursorutil.RepoCursor
		projectCursor  cursorutil.ProjectCursor
	}

	type projectAndRepos struct {
		project *db.Project
		repos   []*db.Repository
	}

	tests := []struct {
		name                         string
		projectsAndRepos             []projectAndRepos
		additionalProjectsAndRepos   []projectAndRepos
		additionalNonQueriedProjects []*db.Project
		recurrenceCfg                reminderconfig.RecurrenceConfig
		setup                        func(store *mockdb.MockStore, projectsAndRepos, additionalProjectsAndRepos []projectAndRepos, additionalNonQueriedProjects []*db.Project)
		want                         want
		err                          string
	}{
		{
			name: "get repository batch no additional repos",
			projectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 1),
						Name:      "project-1",
						CreatedAt: getCreatedAtFromNum(t, 1),
					},
					repos: getRepoTillId(t, 4),
				},
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 2),
						Name:      "project-2",
						CreatedAt: getCreatedAtFromNum(t, 2),
					},
					repos: getRepoTillId(t, 4),
				},
			},
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed:    "1h",
				BatchSize:     6,
				MaxPerProject: 3,
			},
			setup: func(store *mockdb.MockStore, projectsAndRepos, _ []projectAndRepos, _ []*db.Project) {
				for _, projAndRepos := range projectsAndRepos {
					returnedRepos := make([]db.Repository, len(projAndRepos.repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(projAndRepos.repos)-1)
					for i := range projAndRepos.repos {
						returnedRepos[i] = *projAndRepos.repos[i]

						// Magic number -1 i.e. last index is skipped as only MaxPerProject = 3 repos
						// are returned internally
						if i != len(projAndRepos.repos)-1 {
							repoWithLastUpdated[i] = repoLastUpdatedIn{
								repo:        projAndRepos.repos[i],
								lastUpdated: time.Now().Add(-2 * time.Hour),
							}
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: projAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}
			},
			want: want{
				repos: append(getRepoTillId(t, 3), getRepoTillId(t, 3)...),
				repoListCursor: map[projectProviderPair]cursorutil.RepoCursor{
					{
						ProjectId: generateUUIDFromNum(t, 1),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 1).String(),
						Provider:  "github",
						RepoId:    4,
					},
					{
						ProjectId: generateUUIDFromNum(t, 2),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 2).String(),
						Provider:  "github",
						RepoId:    4,
					},
				},
			},
		},
		{
			name: "get repository batch with additional repos with profile cursor update",
			projectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 1),
						Name:      "project-1",
						CreatedAt: getCreatedAtFromNum(t, 1),
					},
					repos: getRepoTillId(t, 3),
				},
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 2),
						Name:      "project-2",
						CreatedAt: getCreatedAtFromNum(t, 2),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			additionalProjectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 3),
						Name:      "project-3",
						CreatedAt: getCreatedAtFromNum(t, 3),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			additionalNonQueriedProjects: []*db.Project{
				{
					ID:        generateUUIDFromNum(t, 4),
					Name:      "project-4",
					CreatedAt: getCreatedAtFromNum(t, 4),
				},
			},
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed:    "1h",
				BatchSize:     6,
				MaxPerProject: 2,
			},
			setup: func(store *mockdb.MockStore, projectsAndRepos, additionalProjectsAndRepos []projectAndRepos, additionalNonQueriedProjects []*db.Project) {
				for _, projAndRepos := range projectsAndRepos {
					returnedRepos := make([]db.Repository, len(projAndRepos.repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(projAndRepos.repos)-1)
					for i := range projAndRepos.repos {
						returnedRepos[i] = *projAndRepos.repos[i]

						// Magic number -1 i.e. last index is skipped as only MaxPerProject = 2 repos
						// are returned internally
						if i != len(projAndRepos.repos)-1 {
							repoWithLastUpdated[i] = repoLastUpdatedIn{
								repo:        projAndRepos.repos[i],
								lastUpdated: time.Now().Add(-2 * time.Hour),
							}
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: projAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},

						// Magic number 3 is used as only MaxPerProject = 2 repos are returned
						// and extra repo is fetched to update the cursor
						Limit: sql.NullInt64{Int64: 3, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}

				for _, projAndRepos := range additionalProjectsAndRepos {
					returnedRepos := make([]db.Repository, len(projAndRepos.repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(projAndRepos.repos)-1)
					for i := range projAndRepos.repos {
						returnedRepos[i] = *projAndRepos.repos[i]

						// Magic number -1 i.e. last index is skipped as only MaxPerProject = 2 repos
						// are returned internally
						if i != len(projAndRepos.repos)-1 {
							repoWithLastUpdated[i] = repoLastUpdatedIn{
								repo:        projAndRepos.repos[i],
								lastUpdated: time.Now().Add(-2 * time.Hour),
							}
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: projAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},

						// Magic number 3 is used as only MaxPerProject = 2 repos are returned
						// and extra repo is fetched to update the cursor
						Limit: sql.NullInt64{Int64: 3, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}

				projects := make([]db.Project, len(additionalProjectsAndRepos)+len(additionalNonQueriedProjects))
				for i, projAndRepos := range additionalProjectsAndRepos {
					projects[i] = *projAndRepos.project
				}

				for i, proj := range additionalNonQueriedProjects {
					projects[i+len(additionalProjectsAndRepos)] = *proj
				}

				store.EXPECT().ListProjects(gomock.Any(), gomock.Any()).
					Return(projects, nil)
			},
			want: want{
				repos: append(getRepoTillId(t, 2), append(getRepoTillId(t, 2), getRepoTillId(t, 2)...)...),
				repoListCursor: map[projectProviderPair]cursorutil.RepoCursor{
					{
						ProjectId: generateUUIDFromNum(t, 1),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 1).String(),
						Provider:  "github",
						RepoId:    3,
					},
					{
						ProjectId: generateUUIDFromNum(t, 2),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 2).String(),
						Provider:  "github",
						RepoId:    3,
					},
					{
						ProjectId: generateUUIDFromNum(t, 3),
						Provider:  "github",
					}: {
						ProjectId: generateUUIDFromNum(t, 3).String(),
						Provider:  "github",
						RepoId:    3,
					},
				},
				projectCursor: cursorutil.ProjectCursor{
					Id: generateUUIDFromNum(t, 4),
				},
			},
		},
		{
			name: "error listing additional projects",
			projectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 1),
						Name:      "project-1",
						CreatedAt: getCreatedAtFromNum(t, 1),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed:    "1h",
				BatchSize:     6,
				MaxPerProject: 3,
			},
			setup: func(store *mockdb.MockStore, projectsAndRepos, _ []projectAndRepos, _ []*db.Project) {
				for _, projAndRepos := range projectsAndRepos {
					returnedRepos := make([]db.Repository, len(projAndRepos.repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(projAndRepos.repos))
					for i := range projAndRepos.repos {
						returnedRepos[i] = *projAndRepos.repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        projAndRepos.repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: projAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}

				store.EXPECT().ListProjects(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrConnDone)
			},
			want: want{},
			err:  fmt.Sprintf("error getting additional repos for reconciliation: error listing projects: %s", sql.ErrConnDone),
		},
		{
			name: "error listing additional repositories",
			projectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 1),
						Name:      "project-1",
						CreatedAt: getCreatedAtFromNum(t, 1),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			additionalProjectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 2),
						Name:      "project-2",
						CreatedAt: getCreatedAtFromNum(t, 2),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed:    "1h",
				BatchSize:     6,
				MaxPerProject: 3,
			},
			setup: func(store *mockdb.MockStore, projectsAndRepos, additionalProjectsAndRepos []projectAndRepos, _ []*db.Project) {
				for _, projAndRepos := range projectsAndRepos {
					returnedRepos := make([]db.Repository, len(projAndRepos.repos))
					repoWithLastUpdated := make([]repoLastUpdatedIn, len(projAndRepos.repos))
					for i := range projAndRepos.repos {
						returnedRepos[i] = *projAndRepos.repos[i]
						repoWithLastUpdated[i] = repoLastUpdatedIn{
							repo:        projAndRepos.repos[i],
							lastUpdated: time.Now().Add(-2 * time.Hour),
						}
					}

					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: projAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(returnedRepos, nil)

					standardLastUpdatedSetup(store, repoWithLastUpdated)
				}

				for _, additionalProjectAndRepos := range additionalProjectsAndRepos {
					store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
						Provider:  "github",
						ProjectID: additionalProjectAndRepos.project.ID,
						RepoID:    sql.NullInt64{Valid: false},
						Limit:     sql.NullInt64{Int64: 4, Valid: true},
					}).Return(nil, sql.ErrConnDone)
				}

				projects := make([]db.Project, len(additionalProjectsAndRepos))
				for i, additionalProjectAndRepos := range additionalProjectsAndRepos {
					projects[i] = *additionalProjectAndRepos.project
				}

				store.EXPECT().ListProjects(gomock.Any(), gomock.Any()).
					Return(projects, nil)
			},
			want: want{},
			err:  fmt.Sprintf("error getting additional repos for reconciliation: error getting repos for reconciliation: error listing repositories: %s", sql.ErrConnDone),
		},
		{
			name: "error getting repos for reconciliation",
			projectsAndRepos: []projectAndRepos{
				{
					project: &db.Project{
						ID:        generateUUIDFromNum(t, 1),
						Name:      "project-1",
						CreatedAt: getCreatedAtFromNum(t, 1),
					},
					repos: getRepoTillId(t, 3),
				},
			},
			recurrenceCfg: reminderconfig.RecurrenceConfig{
				MinElapsed:    "1h",
				BatchSize:     6,
				MaxPerProject: 3,
			},
			setup: func(store *mockdb.MockStore, _, _ []projectAndRepos, _ []*db.Project) {
				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrTxDone)
			},
			want: want{},
			err:  fmt.Sprintf("error getting repos for reconciliation: error listing repositories: %s", sql.ErrTxDone),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.projectsAndRepos, test.additionalProjectsAndRepos, test.additionalNonQueriedProjects)
			cfg := &reminderconfig.Config{
				RecurrenceConfig: test.recurrenceCfg,
			}

			r := createTestReminder(t, store, cfg)
			projects := make([]*db.Project, len(test.projectsAndRepos))
			for i, projAndRepos := range test.projectsAndRepos {
				projects[i] = projAndRepos.project
			}

			got, err := r.getRepositoryBatch(context.Background(), projects)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.want.repos)
			require.Equal(t, len(test.want.repoListCursor), len(r.repoListCursor))
			for k, v := range test.want.repoListCursor {
				require.Equal(t, v.String(), r.repoListCursor[k])
			}

			cursor, err := cursorutil.NewProjectCursor(r.projectListCursor)
			require.NoError(t, err)

			// We do not cover null cursor, that is tested separately
			require.NotNil(t, cursor)
			require.Equal(t, test.want.projectCursor, *cursor)
		})
	}
}

func Test_cursorStateBackup(t *testing.T) {
	t.Parallel()

	tempDirPath := t.TempDir()
	cursorFilePath := filepath.Join(tempDirPath, "cursor")
	repoListCursor := map[projectProviderPair]string{
		{
			ProjectId: generateUUIDFromNum(t, 1),
			Provider:  "github",
		}: "repo-cursor-1",
		{
			ProjectId: generateUUIDFromNum(t, 2),
			Provider:  "gitlab",
		}: "repo-cursor-2",
	}
	projectCursor := "project-cursor"

	r := &reminder{
		cfg: &reminderconfig.Config{
			CursorFile: cursorFilePath,
		},
		projectListCursor: projectCursor,
		repoListCursor:    repoListCursor,
	}

	err := r.storeCursorState()
	require.NoError(t, err)

	// Set cursors to empty values to check if they are restored
	r.projectListCursor = ""
	r.repoListCursor = nil

	err = r.restoreCursorState()
	require.NoError(t, err)

	require.Equal(t, projectCursor, r.projectListCursor)
	require.Equal(t, len(repoListCursor), len(r.repoListCursor))
	for k, v := range repoListCursor {
		require.Equal(t, v, r.repoListCursor[k])
	}
}

func getCreatedAtFromNum(t *testing.T, numFromZeroToSixty int) time.Time {
	t.Helper()

	creationTime := time.Date(2023, time.February, 23, 10, 0, numFromZeroToSixty, 0, time.UTC)
	return creationTime
}

func getRepoTillId(t *testing.T, tillId int) []*db.Repository {
	t.Helper()

	repos := make([]*db.Repository, 0, tillId)
	for i := 1; i <= tillId; i++ {
		repoName := fmt.Sprintf("repo-%d", i)
		repos = append(repos, &db.Repository{
			RepoID:   int64(i),
			RepoName: repoName,
			ID:       generateUUIDFromNum(t, i),
		})
	}
	return repos
}

func getRepoWithId(t *testing.T, id int) *db.Repository {
	t.Helper()
	repoName := fmt.Sprintf("repo-%d", id)
	return &db.Repository{
		RepoID:   int64(id),
		RepoName: repoName,
		ID:       generateUUIDFromNum(t, id),
	}
}

func generateUUIDFromNum(t *testing.T, num int) uuid.UUID {
	t.Helper()

	numberStr := fmt.Sprintf("%d", num)

	uuidStr := fmt.Sprintf("00000000-0000-0000-0000-%012s", numberStr)

	u, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Errorf("error parsing UUID: %v", err)
	}

	return u
}

func createTestReminder(t *testing.T, store db.Store, config *reminderconfig.Config) *reminder {
	t.Helper()

	return &reminder{
		store:          store,
		cfg:            config,
		repoListCursor: make(map[projectProviderPair]string),
	}
}

type repoLastUpdatedIn struct {
	repo        *db.Repository
	lastUpdated time.Time
}

func standardLastUpdatedSetup(store *mockdb.MockStore, input []repoLastUpdatedIn) {
	repoIds := make([]uuid.UUID, 0, len(input))
	for _, in := range input {
		repoIds = append(repoIds, in.repo.ID)
	}

	res := make([]db.ListOldestRuleEvaluationsByRepositoryIdRow, len(input))
	for _, in := range input {
		res = append(res, db.ListOldestRuleEvaluationsByRepositoryIdRow{
			RepositoryID:      in.repo.ID,
			OldestLastUpdated: in.lastUpdated,
		})
	}

	store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), repoIds).
		Return(res, nil)
}
