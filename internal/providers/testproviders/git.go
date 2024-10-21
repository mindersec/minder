// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package testproviders

import (
	"github.com/mindersec/minder/internal/providers/git"
	"github.com/mindersec/minder/internal/providers/noop"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// GitProvider is a test implementation of the Git provider
// interface
type GitProvider struct {
	noop.Provider
	*git.Git
}

// NewGitProvider creates a new Git provider with credentials and options
func NewGitProvider(credential provifv1.GitCredential, opts ...git.Options) *GitProvider {
	return &GitProvider{
		Git: git.NewGit(credential, opts...),
	}
}

// Ensure GitProvider implements the Provider interface
var _ provifv1.Provider = (*GitProvider)(nil)

// CanImplement implements the Provider interface
func (_ *GitProvider) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_GIT
}
