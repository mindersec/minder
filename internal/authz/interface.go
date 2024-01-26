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
)

// ErrNotAuthorized is the error returned when a user is not authorized to perform an action
var ErrNotAuthorized = fmt.Errorf("not authorized")

// Role is the role a user can have on a project
type Role string

const (
	// AuthzRoleAdmin is the admin role
	AuthzRoleAdmin Role = "admin"
	// AuthzRoleEditor is the editor role
	AuthzRoleEditor Role = "editor"
	// AuthzRoleViewer is the viewer role
	AuthzRoleViewer Role = "viewer"
	// AuthzRolePolicyWriter is the `policy_writer` role
	AuthzRolePolicyWriter Role = "policy_writer"
)

func (r Role) String() string {
	return string(r)
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

	// PrepareForRun allows for any preflight configurations to be done before
	// the server is started.
	PrepareForRun(ctx context.Context) error

	// MigrateUp runs the authz migrations
	MigrateUp(ctx context.Context) error
}
