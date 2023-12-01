// Copyright 2023 Stacklok, Inc
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

package controlplane

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"golang.org/x/exp/slices"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
)

// checks if an user is superadmin
func isSuperadmin(claims auth.UserPermissions) bool {
	return claims.IsStaff
}

func containsSuperadminRole(openIdToken openid.Token) bool {
	if realmAccess, ok := openIdToken.Get("realm_access"); ok {
		if realms, ok := realmAccess.(map[string]interface{}); ok {
			if roles, ok := realms["roles"]; ok {
				if userRoles, ok := roles.([]interface{}); ok {
					if slices.Contains(userRoles, "superadmin") {
						return true
					}
				}
			}
		}
	}
	return false
}

// lookupUserPermissions returns the user permissions from the database for the given user
func lookupUserPermissions(ctx context.Context, store db.Store, tok openid.Token) (auth.UserPermissions, error) {
	emptyPermissions := auth.UserPermissions{}

	// read all information for user claims
	userInfo, err := store.GetUserBySubject(ctx, tok.Subject())
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to read user")
	}

	// read groups and add id to claims
	gs, err := store.GetUserProjects(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to get groups")
	}
	var groups []uuid.UUID
	for _, g := range gs {
		groups = append(groups, g.ID)
	}

	// read roles and add details to claims
	rs, err := store.GetUserRoles(ctx, userInfo.ID)
	if err != nil {
		return emptyPermissions, fmt.Errorf("failed to get roles")
	}

	var roles []auth.RoleInfo
	for _, r := range rs {
		rif := auth.RoleInfo{
			RoleID:         r.ID,
			IsAdmin:        r.IsAdmin,
			OrganizationID: r.OrganizationID,
		}
		if r.ProjectID.Valid {
			pID := r.ProjectID.UUID
			rif.ProjectID = &pID
		}
		roles = append(roles, rif)
	}

	claims := auth.UserPermissions{
		UserId:         userInfo.ID,
		Roles:          roles,
		ProjectIds:     groups,
		OrganizationId: userInfo.OrganizationID,
		IsStaff:        containsSuperadminRole(tok),
	}

	return claims, nil
}
