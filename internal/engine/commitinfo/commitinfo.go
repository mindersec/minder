// Package commitinfo provides utilities for extracting and normalizing
// pull request commit metadata into a provider-agnostic structure.
//
// It is designed to decouple commit data handling from provider-specific
// implementations (e.g., GitHub) and avoid import cycles between engine
// components.
package commitinfo

import (
	"strings"

	"github.com/google/go-github/v63/github"
)

// CommitInfo represents normalized pull request commit metadata.
type CommitInfo struct {
	SHA     string
	Message string
	Author  string
}

// Extract normalizes a GitHub commit into a CommitInfo struct.
func Extract(c *github.RepositoryCommit) CommitInfo {
	msg := ""
	if c.GetCommit() != nil {
		msg = c.GetCommit().GetMessage()
	}

	firstLine := msg
	if idx := strings.Index(msg, "\n"); idx != -1 {
		firstLine = msg[:idx]
	}
	firstLine = strings.TrimSpace(firstLine)

	author := ""
	if c.GetCommit() != nil && c.GetCommit().Author != nil {
		author = c.GetCommit().Author.GetName()
	}

	return CommitInfo{
		SHA:     c.GetSHA(),
		Message: firstLine,
		Author:  author,
	}
}
