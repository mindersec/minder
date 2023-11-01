// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package git_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/internal/db"
	gitengine "github.com/stacklok/mediator/internal/engine/ingester/git"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

func TestGitIngestWithCloneURLFromRepo(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(&pb.GitType{
		Branch: "master",
	}, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				"git",
				"rest",
				"github",
			},
		},
		db.ProviderAccessToken{},
		"",
	))
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{
		CloneUrl: "https://github.com/octocat/Hello-World.git",
	}, map[string]interface{}{})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
	require.NotNil(t, got.Fs, "expected non-nil fs")

	fs := got.Fs
	f, err := fs.Open("README")
	require.NoError(t, err, "expected no error")

	// should contain hello world
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(f)
	require.NoError(t, err, "expected no error")

	require.Contains(t, buf.String(), "Hello World", "expected README.md to contain Hello World")
}

func TestGitIngestWithCloneURLFromParams(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(&pb.GitType{
		Branch: "master",
	}, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				"git",
				"rest",
				"github",
			},
		},
		db.ProviderAccessToken{},
		"",
	))
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Artifact{}, map[string]any{
		"clone_url": "https://github.com/octocat/Hello-World.git",
	})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
	require.NotNil(t, got.Fs, "expected non-nil fs")

	fs := got.Fs
	f, err := fs.Open("README")
	require.NoError(t, err, "expected no error")

	// should contain hello world
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(f)
	require.NoError(t, err, "expected no error")

	require.Contains(t, buf.String(), "Hello World", "expected README.md to contain Hello World")
}

func TestGitIngestWithCustomBranchFromParams(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(&pb.GitType{
		Branch: "master",
	}, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				"git",
				"rest",
				"github",
			},
		},
		db.ProviderAccessToken{},
		"",
	))
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Artifact{}, map[string]any{
		"clone_url": "https://github.com/octocat/Hello-World.git",
		"branch":    "test",
	})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
	require.NotNil(t, got.Fs, "expected non-nil fs")

	fs := got.Fs
	f, err := fs.Open("README")
	require.NoError(t, err, "expected no error")

	// should contain hello world
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(f)
	require.NoError(t, err, "expected no error")

	require.Contains(t, buf.String(), "Hello World", "expected README.md to contain Hello World")
}

func TestGitIngestFailsBecauseOfAuthorization(t *testing.T) {
	t.Parallel()

	// foobar is not a valid token
	gi, err := gitengine.NewGitIngester(&pb.GitType{
		Branch: "master",
	}, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				"git",
				"rest",
				"github",
			},
		},
		db.ProviderAccessToken{},
		"foobar",
	),
	)
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Artifact{}, map[string]any{
		"clone_url": "https://github.com/stacklok/mediator.git",
	})
	require.Error(t, err, "expected error")
	require.Nil(t, got, "expected nil result")
}

func TestGitIngestFailsBecauseOfUnexistentCloneUrl(t *testing.T) {
	t.Parallel()

	// foobar is not a valid token
	gi, err := gitengine.NewGitIngester(&pb.GitType{}, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				"git",
				"rest",
				"github",
			},
		},
		db.ProviderAccessToken{},
		// No authentication is the right thing in this case.
		"",
	))
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Artifact{}, map[string]any{
		"clone_url": "https://github.com/octocat/unexistent-git-repo.git",
	})
	require.Error(t, err, "expected error")
	require.Nil(t, got, "expected nil result")
}
