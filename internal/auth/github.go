//
// Copyright 2023 Stacklok, Inc.
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

// Package auth contains the authentication logic for the control plane
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/stacklok/minder/internal/config/server"
)

// GetUserForGitHubId looks up a user in Keycloak by their GitHub ID.  This is a temporary
// implementation until we have a proper interface in front of IDP implementations.
//
// If the user is found, it returns their subject _in Keycloak_, suitable for use in
// the `sub` claim of a JWT, and in OpenFGA's user field.  Note that this function may
// return a user of "" with no error if no users were found matching the GitHub ID.
func GetUserForGitHubId(ctx context.Context, sic server.IdentityConfigWrapper, ghUser int64) (string, error) {
	// look up the user in the identity provider (keycloak-specific for now)
	q := url.Values{
		"q": {fmt.Sprintf("gh_id:%d", ghUser)},
		// TODO: add idpAlias and configuration for same
	}
	resp, err := sic.Server.Do(ctx, "GET", "admin/realms/stacklok/users", q, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	type kcUser struct {
		Id         string
		Username   string
		Attributes map[string][]string
	}
	users := []kcUser{}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return "", err
	}
	if len(users) == 0 {
		// No user found, that's okay.
		return "", nil
	}
	if len(users) > 1 {
		return "", fmt.Errorf("expected 1 user, got %d", len(users))
	}
	return users[0].Id, nil
}
