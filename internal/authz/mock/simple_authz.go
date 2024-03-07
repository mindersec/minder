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

// Package mock provides a no-op implementation of the minder the authorization client
package mock

import (
	"context"
	"slices"
	"sync/atomic"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/authz"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// SimpleClient maintains a list of authorized projects, suitable for use in tests.
type SimpleClient struct {
	Allowed     []uuid.UUID
	Assignments map[uuid.UUID][]*minderv1.RoleAssignment

	// Adoptions is a map of child project to parent project
	Adoptions map[uuid.UUID]uuid.UUID

	// OrphanCalls is a counter for the number of times Orphan is called
	OrphanCalls atomic.Int32
}

var _ authz.Client = &SimpleClient{}

// Check implements authz.Client
func (n *SimpleClient) Check(_ context.Context, _ string, project uuid.UUID) error {
	if slices.Contains(n.Allowed, project) {
		return nil
	}
	return authz.ErrNotAuthorized
}

// Write implements authz.Client
func (n *SimpleClient) Write(_ context.Context, _ string, _ authz.Role, project uuid.UUID) error {
	n.Allowed = append(n.Allowed, project)
	return nil
}

// Delete implements authz.Client
func (n *SimpleClient) Delete(_ context.Context, _ string, _ authz.Role, project uuid.UUID) error {
	index := slices.Index(n.Allowed, project)
	if index != -1 {
		n.Allowed[index] = n.Allowed[len(n.Allowed)-1]
		n.Allowed = n.Allowed[:len(n.Allowed)-1]
	}
	return nil
}

// DeleteUser implements authz.Client
func (n *SimpleClient) DeleteUser(_ context.Context, _ string) error {
	n.Assignments = nil
	n.Allowed = nil
	return nil
}

// AssignmentsToProject implements authz.Client
func (n *SimpleClient) AssignmentsToProject(_ context.Context, p uuid.UUID) ([]*minderv1.RoleAssignment, error) {
	if n.Assignments == nil {
		return nil, nil
	}

	if _, ok := n.Assignments[p]; !ok {
		return nil, nil
	}

	return n.Assignments[p], nil
}

// ProjectsForUser implements authz.Client
func (n *SimpleClient) ProjectsForUser(_ context.Context, _ string) ([]uuid.UUID, error) {
	return n.Allowed, nil
}

// PrepareForRun implements authz.Client
func (_ *SimpleClient) PrepareForRun(_ context.Context) error {
	return nil
}

// MigrateUp implements authz.Client
func (_ *SimpleClient) MigrateUp(_ context.Context) error {
	return nil
}

// Adopt implements authz.Client
func (n *SimpleClient) Adopt(_ context.Context, p, c uuid.UUID) error {

	if n.Adoptions == nil {
		n.Adoptions = make(map[uuid.UUID]uuid.UUID)
	}

	n.Adoptions[c] = p
	return nil
}

// Orphan implements authz.Client
func (n *SimpleClient) Orphan(_ context.Context, _, c uuid.UUID) error {
	n.OrphanCalls.Add(int32(1))
	if n.Adoptions == nil {
		return nil
	}

	delete(n.Adoptions, c)

	return nil
}
