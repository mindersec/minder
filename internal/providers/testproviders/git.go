// Copyright 2024 Stacklok, Inc.
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

package testproviders

import (
	"github.com/stacklok/minder/internal/providers/git"
	"github.com/stacklok/minder/internal/providers/noop"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
