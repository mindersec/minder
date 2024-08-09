//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitlab

import (
	"context"

	"github.com/go-git/go-git/v5"

	gitclient "github.com/stacklok/minder/internal/providers/git"
)

// Implements the Git interface
func (c *gitlabClient) Clone(ctx context.Context, cloneUrl string, branch string) (*git.Repository, error) {
	g := gitclient.NewGit(c.GetCredential(), gitclient.WithConfig(c.gitConfig))
	return g.Clone(ctx, cloneUrl, branch)
}
