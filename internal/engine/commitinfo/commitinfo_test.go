// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package commitinfo

import (
	"testing"

	"github.com/google/go-github/v63/github"
)

func TestExtract_Basic(t *testing.T) {
	t.Parallel()

	msg := "feat: add new feature\n\nmore details"
	author := "Sachin"

	c := &github.RepositoryCommit{
		SHA: github.String("abc123"),
		Commit: &github.Commit{
			Message: github.String(msg),
			Author: &github.CommitAuthor{
				Name: github.String(author),
			},
		},
	}

	info := Extract(c)

	if info.SHA != "abc123" {
		t.Fatalf("expected SHA abc123, got %s", info.SHA)
	}

	if info.Message != "feat: add new feature" {
		t.Fatalf("unexpected message: %s", info.Message)
	}

	if info.Author != author {
		t.Fatalf("expected author %s, got %s", author, info.Author)
	}
}
