// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// projectsJSON renders a GitLab projects list response for the given IDs.
func projectsJSON(ids ...int) string {
	projects := make([]string, 0, len(ids))
	for _, id := range ids {
		project := fmt.Sprintf(`{"id": %d, "name": "project-%d", "namespace": {"path": "group"}}`, id, id)
		projects = append(projects, project)
	}
	return "[" + strings.Join(projects, ",") + "]"
}

// pagesUpTo returns the page numbers 1 through n.
func pagesUpTo(n int) []int {
	pages := make([]int, 0, n)
	for page := 1; page <= n; page++ {
		pages = append(pages, page)
	}
	return pages
}

// pageResponse is how the fake GitLab server answers a request for one page.
type pageResponse struct {
	status   int
	body     string
	nextPage string
}

func TestListAllRepositories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		respond    func(page int) pageResponse
		wantPages  []int
		wantRepos  int
		wantNames  []string
		wantErrIs  error
		wantErrMsg string
	}{
		{
			name: "follows X-Next-Page across pages",
			respond: func(page int) pageResponse {
				if page == 1 {
					return pageResponse{body: projectsJSON(1, 2), nextPage: "2"}
				}
				return pageResponse{body: projectsJSON(3)}
			},
			wantPages: []int{1, 2},
			wantRepos: 3,
			wantNames: []string{"project-1", "project-2", "project-3"},
		},
		{
			name: "single page without a next page",
			respond: func(_ int) pageResponse {
				return pageResponse{body: projectsJSON(1)}
			},
			wantPages: []int{1},
			wantRepos: 1,
			wantNames: []string{"project-1"},
		},
		{
			name: "empty listing",
			respond: func(_ int) pageResponse {
				return pageResponse{body: "[]", nextPage: "2"}
			},
			wantPages: []int{1},
			wantRepos: 0,
		},
		{
			// A server that keeps reporting the page it just served must not
			// keep us looping, even when every page has items.
			name: "non-advancing next page ends the listing",
			respond: func(page int) pageResponse {
				return pageResponse{body: projectsJSON(page), nextPage: "2"}
			},
			wantPages: []int{1, 2},
			wantRepos: 2,
		},
		{
			name: "next page pointing backwards ends the listing",
			respond: func(page int) pageResponse {
				if page == 1 {
					return pageResponse{body: projectsJSON(1), nextPage: "2"}
				}
				return pageResponse{body: projectsJSON(2), nextPage: "1"}
			},
			wantPages: []int{1, 2},
			wantRepos: 2,
		},
		{
			name: "malformed next page ends the listing",
			respond: func(_ int) pageResponse {
				return pageResponse{body: projectsJSON(1), nextPage: "not-a-page"}
			},
			wantPages: []int{1},
			wantRepos: 1,
		},
		{
			// The listing is bounded: hitting the page limit is an error, and
			// the projects fetched so far are returned alongside it.
			name: "page limit is an error",
			respond: func(page int) pageResponse {
				return pageResponse{body: projectsJSON(page), nextPage: strconv.Itoa(page + 1)}
			},
			wantPages:  pagesUpTo(maxPages),
			wantRepos:  maxPages,
			wantErrMsg: "too many pages",
		},
		{
			name: "not found",
			respond: func(_ int) pageResponse {
				return pageResponse{status: http.StatusNotFound}
			},
			wantPages: []int{1},
			wantRepos: 0,
			wantErrIs: provifv1.ErrEntityNotFound,
		},
		{
			// A failure partway through returns the earlier pages' projects
			// with the error, so the caller can decide what to do with them.
			name: "failure on a later page returns earlier pages with the error",
			respond: func(page int) pageResponse {
				if page == 1 {
					return pageResponse{body: projectsJSON(1, 2), nextPage: "2"}
				}
				return pageResponse{status: http.StatusInternalServerError}
			},
			wantPages:  []int{1, 2},
			wantRepos:  2,
			wantNames:  []string{"project-1", "project-2"},
			wantErrMsg: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var mu sync.Mutex
			var pages []int
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				// The caller's own query parameters must survive pagination.
				assert.Equal(t, strconv.Itoa(minAccessLevelControl), q.Get("min_access_level"))
				assert.Equal(t, strconv.Itoa(perPage), q.Get("per_page"))

				page, err := strconv.Atoi(q.Get("page"))
				if !assert.NoError(t, err, "page parameter must be numeric") {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				mu.Lock()
				pages = append(pages, page)
				mu.Unlock()

				resp := tt.respond(page)
				if resp.nextPage != "" {
					w.Header().Set(nextPageHeader, resp.nextPage)
				}
				if resp.status != 0 {
					w.WriteHeader(resp.status)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, err = fmt.Fprint(w, resp.body)
				assert.NoError(t, err)
			}))
			defer ts.Close()

			repos, err := newTestGitlabProvider(ts.URL).ListAllRepositories(context.Background())

			switch {
			case tt.wantErrIs != nil:
				require.ErrorIs(t, err, tt.wantErrIs)
			case tt.wantErrMsg != "":
				require.ErrorContains(t, err, tt.wantErrMsg)
			default:
				require.NoError(t, err)
			}

			require.Len(t, repos, tt.wantRepos)
			if tt.wantNames != nil {
				names := make([]string, 0, len(repos))
				for _, r := range repos {
					names = append(names, r.GetName())
				}
				assert.Equal(t, tt.wantNames, names)
			}

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tt.wantPages, pages)
		})
	}
}
