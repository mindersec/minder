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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/xanzy/go-gitlab"

	"github.com/stacklok/minder/internal/entities/properties"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func (c *gitlabClient) getPropertiesForRepo(
	ctx context.Context, getByProps *properties.Properties,
) (*properties.Properties, error) {
	uid, err := getByProps.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("upstream ID not found or invalid: %w", err)
	}

	projectURLPath, err := url.JoinPath("projects", url.PathEscape(uid))
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for project using upstream ID: %w", err)
	}

	// NOTE: We're not using github.com/xanzy/go-gitlab to do the actual
	// request here because of the way they form authentication for requests.
	// It would be ideal to use it, so we should consider contributing and making
	// that part more pluggable.
	req, err := c.NewRequest("GET", projectURLPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, provifv1.ErrEntityNotFound
		}
		return nil, fmt.Errorf("failed to get projects: %s", resp.Status)
	}

	proj := &gitlab.Project{}
	if err := json.NewDecoder(resp.Body).Decode(proj); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	outProps, err := properties.NewProperties(map[string]any{
		properties.RepoPropertyIsPrivate:  proj.Visibility == gitlab.PrivateVisibility,
		properties.RepoPropertyIsArchived: proj.Archived,
		properties.RepoPropertyIsFork:     proj.ForkedFromProject != nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create properties: %w", err)
	}

	return getByProps.Merge(outProps), nil
}
