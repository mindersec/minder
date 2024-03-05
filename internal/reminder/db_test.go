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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
)

func Test_listProjects(t *testing.T) {
	t.Parallel()

	type want struct {
		projects []db.Project
		cursorId uuid.UUID
	}

	tests := []struct {
		name  string
		req   listProjectsRequest
		want  want
		setup func(store *mockdb.MockStore, req listProjectsRequest, proj []db.Project)
		err   string
	}{
		{
			name: "extra project as cursor",
			req: listProjectsRequest{
				cursor: "",
				limit:  3,
			},
			want: want{
				projects: []db.Project{
					{
						Name: "project1",
						ID:   generateUUIDFromNum(t, 1),
					},
					{
						Name: "project2",
						ID:   generateUUIDFromNum(t, 2),
					},
					{
						Name: "project3",
						ID:   generateUUIDFromNum(t, 3),
					},
				},
				cursorId: generateUUIDFromNum(t, 4),
			},
			setup: func(store *mockdb.MockStore, req listProjectsRequest, proj []db.Project) {
				extraResp := []db.Project{
					{
						Name: "project4",
						ID:   generateUUIDFromNum(t, 4),
					},
				}

				returnedResp := append(proj, extraResp...)

				store.EXPECT().ListProjects(gomock.Any(), db.ListProjectsParams{
					ID: uuid.NullUUID{
						UUID:  uuid.Nil,
						Valid: false,
					},
					Limit: sql.NullInt64{
						Int64: int64(req.limit + 1),
						Valid: true,
					},
				}).Return(returnedResp, nil)
			},
			err: "",
		},
		{
			name: "error with wrong cursor",
			req: listProjectsRequest{
				cursor: "wrong-cursor",
				limit:  3,
			},
			want: want{},
			setup: func(_ *mockdb.MockStore, _ listProjectsRequest, _ []db.Project) {
			},
			err: "error decoding cursor: illegal base64 data at input byte 5",
		},
		{
			name: "error with store",
			req: listProjectsRequest{
				cursor: "",
				limit:  3,
			},
			want: want{},
			setup: func(store *mockdb.MockStore, _ listProjectsRequest, _ []db.Project) {
				store.EXPECT().ListProjects(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrConnDone)
			},
			err: sql.ErrConnDone.Error(),
		},
		{
			name: "no extra project as cursor",
			req: listProjectsRequest{
				cursor: "",
				limit:  1,
			},
			want: want{
				projects: []db.Project{
					{
						Name: "project1",
						ID:   generateUUIDFromNum(t, 1),
					},
				},
				cursorId: uuid.Nil,
			},
			setup: func(store *mockdb.MockStore, _ listProjectsRequest, proj []db.Project) {
				store.EXPECT().ListProjects(gomock.Any(), gomock.Any()).
					Return(proj, nil)
			},
			err: "",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.req, test.want.projects)

			r := &reminder{
				store: store,
			}

			ctx := context.Background()
			got, err := r.listProjects(ctx, test.req)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)

			wanted := make([]*db.Project, len(test.want.projects))
			for i := range test.want.projects {
				wanted[i] = &test.want.projects[i]
			}

			require.ElementsMatch(t, wanted, got.projects)

			cursor, err := cursorutil.NewProjectCursor(got.cursor)
			require.NoError(t, err)

			require.Equal(t, test.want.cursorId, cursor.Id)
		})
	}
}

func Test_listRepositories(t *testing.T) {
	t.Parallel()

	type want struct {
		repositories []db.Repository
		cursor       cursorutil.RepoCursor
	}

	tests := []struct {
		name  string
		req   listRepoRequest
		want  want
		setup func(store *mockdb.MockStore, req listRepoRequest, repos []db.Repository)
		err   string
	}{
		{
			name: "extra repository as cursor",
			req: listRepoRequest{
				projectId: generateUUIDFromNum(t, 1),
				provider:  "github",
				cursor:    "",
				limit:     3,
			},
			want: want{
				repositories: []db.Repository{
					{RepoID: 1},
					{RepoID: 2},
					{RepoID: 3},
				},
				cursor: cursorutil.RepoCursor{
					ProjectId: generateUUIDFromNum(t, 1).String(),
					Provider:  "github",
					RepoId:    4,
				},
			},
			setup: func(store *mockdb.MockStore, req listRepoRequest, repos []db.Repository) {
				extraResp := []db.Repository{
					{RepoID: 4},
				}

				returnedResp := append(repos, extraResp...)

				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
					Provider:  req.provider,
					ProjectID: req.projectId,
					RepoID:    sql.NullInt64{},
					Limit:     sql.NullInt64{Int64: int64(req.limit + 1), Valid: true},
				}).Return(returnedResp, nil)
			},
			err: "",
		},
		{
			name: "error with wrong cursor",
			req: listRepoRequest{
				projectId: uuid.New(),
				provider:  "github",
				cursor:    "wrong-cursor",
				limit:     3,
			},
			want: want{},
			setup: func(_ *mockdb.MockStore, _ listRepoRequest, _ []db.Repository) {
			},
			err: "error decoding cursor: illegal base64 data at input byte 5",
		},
		{
			name: "error with store",
			req: listRepoRequest{
				projectId: uuid.New(),
				provider:  "github",
				cursor:    "",
				limit:     3,
			},
			want: want{},
			setup: func(store *mockdb.MockStore, req listRepoRequest, _ []db.Repository) {
				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
					Provider:  req.provider,
					ProjectID: req.projectId,
					RepoID:    sql.NullInt64{},
					Limit:     sql.NullInt64{Int64: int64(req.limit + 1), Valid: true},
				}).Return(nil, sql.ErrConnDone)
			},
			err: sql.ErrConnDone.Error(),
		},
		{
			name: "no extra repository as cursor",
			req: listRepoRequest{
				projectId: uuid.New(),
				provider:  "github",
				cursor:    "",
				limit:     1,
			},
			want: want{
				repositories: []db.Repository{
					{RepoID: 1},
				},
				cursor: cursorutil.RepoCursor{},
			},
			setup: func(store *mockdb.MockStore, req listRepoRequest, repos []db.Repository) {
				store.EXPECT().ListRepositoriesByProjectID(gomock.Any(), db.ListRepositoriesByProjectIDParams{
					Provider:  req.provider,
					ProjectID: req.projectId,
					RepoID:    sql.NullInt64{},
					Limit:     sql.NullInt64{Int64: int64(req.limit + 1), Valid: true},
				}).Return(repos, nil)
			},
			err: "",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.req, test.want.repositories)

			r := &reminder{
				store: store,
			}

			ctx := context.Background()
			got, err := r.listRepositories(ctx, test.req)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)

			wanted := make([]*db.Repository, len(test.want.repositories))
			for i := range test.want.repositories {
				wanted[i] = &test.want.repositories[i]
			}

			require.ElementsMatch(t, wanted, got.results)

			cursor, err := cursorutil.NewRepoCursor(got.cursor)
			require.NoError(t, err)

			require.Equal(t, test.want.cursor, *cursor)
		})
	}
}

func Test_getOldestRepoRuleEvaluation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  []repoOldestRuleEvaluation
		input []uuid.UUID
		setup func(store *mockdb.MockStore, input []uuid.UUID, want []repoOldestRuleEvaluation)
		err   string
	}{
		{
			name: "success",
			want: []repoOldestRuleEvaluation{
				{
					oldestRuleEvaluation: time.Now(),
					repoId:               uuid.New(),
				},
				{
					oldestRuleEvaluation: time.Now().Add(-time.Hour),
					repoId:               uuid.New(),
				},
			},
			setup: func(store *mockdb.MockStore, input []uuid.UUID, want []repoOldestRuleEvaluation) {
				res := make([]db.ListOldestRuleEvaluationsByRepositoryIdRow, len(want))
				for i, w := range want {
					res[i] = db.ListOldestRuleEvaluationsByRepositoryIdRow{
						RepositoryID:      w.repoId,
						OldestLastUpdated: w.oldestRuleEvaluation,
					}
				}

				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), input).
					Return(res, nil)
			},
			err: "",
		},
		{
			name: "error with store",
			want: nil,
			setup: func(store *mockdb.MockStore, _ []uuid.UUID, _ []repoOldestRuleEvaluation) {
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrConnDone)
			},
			err: sql.ErrConnDone.Error(),
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.input, test.want)

			r := &reminder{
				store: store,
			}

			ctx := context.Background()
			got, err := r.listOldestRuleEvaluationsByIds(ctx, test.input)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				return
			}
			require.NoError(t, err)

			expectedOutput := &listOldestRuleEvaluationsByIdsResponse{
				results: test.want,
			}

			require.Equal(t, expectedOutput, got)
		})
	}
}
