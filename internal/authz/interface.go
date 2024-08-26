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

// Package authz provides the authorization utilities for minder
package authz

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ErrNotAuthorized is the error returned when a user is not authorized to perform an action
var ErrNotAuthorized = fmt.Errorf("not authorized")

// Role is the role a user can have on a project
type Role string

const (
	// RoleAdmin is the admin role
	RoleAdmin Role = "admin"
	// RoleEditor is the editor role
	RoleEditor Role = "editor"
	// RoleViewer is the viewer role
	RoleViewer Role = "viewer"
	// RolePolicyWriter is the `policy_writer` role
	RolePolicyWriter Role = "policy_writer"
	// RolePermissionsManager is the `permissions_manager` role
	RolePermissionsManager Role = "permissions_manager"
)

// nolint:lll
var (
	// AllRolesDescriptions is a list of all roles
	AllRolesDescriptions = map[Role]string{
		RoleAdmin:              "Admins have full permissions on the project. In addition to the editor permissions, users with this role can modify the project, enroll additional providers, and manage roles for other users within the project.",
		RoleEditor:             "In addition to the viewer permissions, editors can author profiles and rule types, as well as add resources to manage. Editors cannot enroll additional providers or change or delete projects.",
		RoleViewer:             "Provides read-only access to the project. Users with this role can view associated resources such as enrolled repositories, rule types, profiles and the status of rule evaluations.",
		RolePolicyWriter:       "Allows users to create rule types and profiles. Unlike editors, policy writers cannot add or remove resources from the project.",
		RolePermissionsManager: "Allows users to manage roles for other users within the project.",
	}
	// AllRolesDisplayName is a list of all roles with their display names
	AllRolesDisplayName = map[Role]string{
		RoleAdmin:              "Admin",
		RoleEditor:             "Editor",
		RoleViewer:             "Viewer",
		RolePolicyWriter:       "Policy Writer",
		RolePermissionsManager: "Permissions Manager",
	}
	// AllRolesSorted is a list of all roles sorted
	AllRolesSorted = []Role{RoleAdmin, RoleEditor, RoleViewer, RolePolicyWriter, RolePermissionsManager}
	// AllRolesVerbs is a list of all roles with their verbs
	AllRolesVerbs = map[Role]string{
		RoleAdmin:              "administer",
		RoleEditor:             "edit",
		RoleViewer:             "view",
		RolePolicyWriter:       "write policies for",
		RolePermissionsManager: "manage permissions for",
	}
)

func (r Role) String() string {
	return string(r)
}

// ParseRole parses a string into a Role
func ParseRole(r string) (Role, error) {
	if r == "" {
		return "", fmt.Errorf("role cannot be empty")
	}
	rr := Role(r)
	if _, ok := AllRolesDescriptions[rr]; !ok {
		return "", fmt.Errorf("invalid role %s", r)
	}

	return rr, nil
}

// Client provides an abstract interface which simplifies interacting with
// OpenFGA and supports no-op and fake implementations.
type Client interface {
	// Check returns a NotAuthorized if the action is not allowed on the resource, or nil if it is allowed
	Check(ctx context.Context, action string, project uuid.UUID) error

	// Write stores an authorization tuple allowing user (an OAuth2 subject) to
	// act in the specified role on the project.
	//
	// NOTE: this method _DOES NOT CHECK_ that the current user in the context
	// has permissions to update the project.
	Write(ctx context.Context, user string, role Role, project uuid.UUID) error
	// Delete removes an authorization from user (an OAuth2 subject) to act in
	// the specified role on the project.
	//
	// NOTE: this method _DOES NOT CHECK_ that the current user in the context
	// has permissions to update the project.
	Delete(ctx context.Context, user string, role Role, project uuid.UUID) error

	// DeleteUser removes all authorizations for the given user.
	DeleteUser(ctx context.Context, user string) error

	// AssignmentsToProject outputs the existing role assignments for a given project.
	AssignmentsToProject(ctx context.Context, project uuid.UUID) ([]*minderv1.RoleAssignment, error)

	// ProjectsForUser outputs the projects a user has access to.
	ProjectsForUser(ctx context.Context, sub string) ([]uuid.UUID, error)

	// PrepareForRun allows for any preflight configurations to be done before
	// the server is started.
	PrepareForRun(ctx context.Context) error

	// MigrateUp runs the authz migrations
	MigrateUp(ctx context.Context) error

	// Adopt stores an authorization relationship from one project to another
	Adopt(ctx context.Context, parent, child uuid.UUID) error

	// Orphan removes an authorization relationship from one project to another
	Orphan(ctx context.Context, parent, child uuid.UUID) error
}
