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

package webhooks

import (
	"context"

	"github.com/google/go-github/v56/github"
)

// GitHubWebhookClient defines the methods from the GitHub client which we
// need. This allows us to create a stub for testing.
type GitHubWebhookClient interface {
	CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, error)
	DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error)
	ListHooks(ctx context.Context, owner, repo string) ([]*github.Hook, error)
}
