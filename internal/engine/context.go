// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package engine

import (
	"context"
	"fmt"

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Group is a construct relevant to an entity's context.
// This is relevant for getting the full information about an entity.
type Group struct {
	ID   int32
	Name string
}

// GetID returns the ID of the group
func (g Group) GetID() int32 {
	return g.ID
}

// GetName returns the name of the group
func (g Group) GetName() string {
	return g.Name
}

// EntityContext is the context of an entity.
// This is relevant for getting the full information about an entity.
type EntityContext struct {
	Group    Group
	Provider string
}

// GetGroup returns the group of the entity
func (c *EntityContext) GetGroup() Group {
	return c.Group
}

// GetProvider returns the provider of the entity
func (c *EntityContext) GetProvider() string {
	return c.Provider
}

// GetContextFromInput returns the context from the input. The
// input is the context from the gRPC request which merely holds
// user-friendly information about an object.
func GetContextFromInput(ctx context.Context, in *pb.Context, q db.Querier) (*EntityContext, error) {
	if in.Group == nil || *in.Group == "" {
		return nil, fmt.Errorf("invalid context: missing group")
	}

	group, err := q.GetGroupByName(ctx, *in.Group)
	if err != nil {
		return nil, fmt.Errorf("unable to get context: %w", err)
	}

	return &EntityContext{
		Group: Group{
			ID:   group.ID,
			Name: group.Name,
		},
		Provider: in.Provider,
	}, nil
}
