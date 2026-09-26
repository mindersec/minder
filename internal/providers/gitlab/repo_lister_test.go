// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllRepositoriesPaginates(t *testing.T) {
	t.Parallel()

	// Two pages of projects: the server reports a second page via the
	// X-Next-Page header and returns different projects for each page.
	pageBodies := map[string]string{
		"1": `[
			{"id": 1, "name": "one", "namespace": {"path": "group"}},
			{"id": 2, "name": "two", "namespace": {"path": "group"}}
		]`,
		"2": `[
			{"id": 3, "name": "three", "namespace": {"path": "group"}}
		]`,
	}

	var requestedPages []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "25", q.Get("min_access_level"), "query params from the caller must be preserved")
		assert.Equal(t, "100", q.Get("per_page"), "pages should be requested at GitLab's maximum size")

		page := q.Get("page")
		requestedPages = append(requestedPages, page)

		body, ok := pageBodies[page]
		require.True(t, ok, "unexpected page requested: %q", page)

		if page == "1" {
			w.Header().Set("X-Next-Page", "2")
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(body))
		require.NoError(t, err)
	}))
	defer ts.Close()

	repos, err := newTestGitlabProvider(ts.URL).ListAllRepositories(context.Background())
	require.NoError(t, err)

	require.Len(t, repos, 3, "projects from every page should be returned")
	assert.Equal(t, []string{"1", "2"}, requestedPages)

	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.GetName())
	}
	assert.Equal(t, []string{"one", "two", "three"}, names)
}

func TestListAllRepositoriesSinglePage(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "1", r.URL.Query().Get("page"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": 1, "name": "solo", "namespace": {"path": "group"}}]`)
	}))
	defer ts.Close()

	repos, err := newTestGitlabProvider(ts.URL).ListAllRepositories(context.Background())
	require.NoError(t, err)
	require.Len(t, repos, 1)
}
