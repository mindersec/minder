// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package auth contains the authentication logic for the control plane
package auth

import (
	"context"
	"errors"
	"fmt"
)

// GetUserForGitHubId looks up a user in the identity provider by their GitHub ID.
//
// If the user is found, it returns their subject suitable for use in
// the `sub` claim of a JWT, and in OpenFGA's user field. Note that this function may
// return a user of "" with no error if no users were found matching the GitHub ID.
func GetUserForGitHubId(ctx context.Context, idClient Resolver, ghUser int64) (string, error) {
	// look up the user in the identity provider
	id, err := idClient.ResolveFederated(ctx, "github", fmt.Sprintf("%d", ghUser))
	if err != nil {
		// If the user is not found, return an empty string and no error
		if errors.Is(err, ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return id.UserID, nil
}
