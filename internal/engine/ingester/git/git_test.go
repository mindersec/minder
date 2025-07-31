// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	gitengine "github.com/mindersec/minder/internal/engine/ingester/git"
	"github.com/mindersec/minder/internal/providers/credentials"
	gitclient "github.com/mindersec/minder/internal/providers/git"
	"github.com/mindersec/minder/internal/providers/testproviders"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestGitIngestWithCloneURLFromRepo(t *testing.T) {
	t.Parallel()

	cfg := server.GitConfig{
		MaxFiles: 100,
		MaxBytes: 1_000_000,
	}
	gitClient := testproviders.NewGitProvider(
		credentials.NewEmptyCredential(),
		gitclient.WithConfig(cfg),
	)

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		gitClient,
	)
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{
		CloneUrl: "https://github.com/octocat/Hello-World.git",
	}, map[string]interface{}{})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
	require.NotNil(t, got.Fs, "expected non-nil fs")
	require.NotNil(t, got.Checkpoint, "expected non-nil checkpoint")

	fs := got.Fs
	f, err := fs.Open("README")
	require.NoError(t, err, "expected no error")

	// should contain hello world
	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(f)
	require.NoError(t, err, "expected no error")

	require.Contains(t, buf.String(), "Hello World", "expected README.md to contain Hello World")

	require.NotNil(t, got.Checkpoint.Checkpoint.Branch, "expected non-nil branch")
	require.Equal(t, "master", *got.Checkpoint.Checkpoint.Branch, "expected branch to be master")

	require.NotNil(t, got.Checkpoint.Checkpoint.CommitHash, "expected non-nil commit")
}

func TestGitIngestWithCloneURLFromParams(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		testproviders.NewGitProvider(credentials.NewEmptyCredential()),
	)
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
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

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		testproviders.NewGitProvider(credentials.NewEmptyCredential()),
	)
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
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

func TestGitIngestWithBranchFromRepoEntity(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(
		//		&pb.GitType{Branch: "master"},
		&pb.GitType{},
		testproviders.NewGitProvider(credentials.NewEmptyCredential()),
	)
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{
		DefaultBranch: "master",
	}, map[string]any{
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

func TestGitIngestWithUnexistentBranchFromParams(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		testproviders.NewGitProvider(credentials.NewEmptyCredential()),
	)

	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
		"clone_url": "https://github.com/octocat/Hello-World.git",
		"branch":    "unexistent-branch",
	})
	require.Error(t, err, "expected error")
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "expected ErrActionFailed")
	require.Nil(t, got, "expected non-nil result")
}

func TestGitIngestFailsBecauseOfAuthorization(t *testing.T) {
	t.Parallel()

	// foobar is not a valid token
	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		testproviders.NewGitProvider(credentials.NewGitHubTokenCredential("foobar")),
	)

	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
		"clone_url": "https://github.com/mindersec/minder.git",
	})
	require.Error(t, err, "expected error")
	require.Nil(t, got, "expected nil result")
}

func TestGitIngestFailsBecauseOfUnexistentCloneUrl(t *testing.T) {
	t.Parallel()

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{}, testproviders.NewGitProvider(credentials.NewEmptyCredential()))
	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
		"clone_url": "https://github.com/octocat/unexistent-git-repo.git",
	})
	require.Error(t, err, "expected error")
	require.Nil(t, got, "expected nil result")
}

func TestGitIngestFailsWhenRepoTooLarge(t *testing.T) {
	t.Parallel()

	// set size limit to 1 byte
	cfg := server.GitConfig{
		MaxFiles: 100,
		MaxBytes: 1,
	}
	gitClient := testproviders.NewGitProvider(
		credentials.NewEmptyCredential(),
		gitclient.WithConfig(cfg),
	)

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		gitClient,
	)

	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
		"clone_url": "https://github.com/octocat/Hello-World.git",
	})
	require.Error(t, err, "expected error")
	require.ErrorIs(t, err, v1.ErrRepositoryTooLarge, "expected ErrRepositoryTooLarge")
	require.Nil(t, got, "expected non-nil result")
}

func TestGitIngestFailsWhenRepoHasTooManyFiles(t *testing.T) {
	t.Parallel()

	// will fail because of files in .git
	cfg := server.GitConfig{
		MaxFiles: 1,
		MaxBytes: 1_000_000,
	}
	gitClient := testproviders.NewGitProvider(
		credentials.NewEmptyCredential(),
		gitclient.WithConfig(cfg),
	)

	gi, err := gitengine.NewGitIngester(
		&pb.GitType{Branch: "master"},
		gitClient,
	)

	require.NoError(t, err, "expected no error")

	got, err := gi.Ingest(context.Background(), &pb.Repository{}, map[string]any{
		"clone_url": "https://github.com/octocat/Hello-World.git",
	})
	require.Error(t, err, "expected error")
	require.ErrorIs(t, err, v1.ErrRepositoryTooLarge, "expected ErrRepositoryTooLarge")
	require.Nil(t, got, "expected non-nil result")
}
