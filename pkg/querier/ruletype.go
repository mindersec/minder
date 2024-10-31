// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// RuleTypeHandlers interface provides functions to interact with ruletypes
type RuleTypeHandlers interface {
	ListRuleTypesByProject(ctx context.Context, projectID uuid.UUID) ([]*pb.RuleType, error)
	CreateRuleType(ctx context.Context, projectID uuid.UUID, subscriptionID uuid.UUID, ruleType *pb.RuleType) (*pb.RuleType, error)
	UpdateRuleType(ctx context.Context, projectID uuid.UUID, subscriptionID uuid.UUID, ruleType *pb.RuleType) (*pb.RuleType, error)
	DeleteRuleType(ctx context.Context, ruleTypeID uuid.UUID) error
	GetRuleTypeByName(ctx context.Context, projectIDs []uuid.UUID, name string) (*pb.RuleType, error)
}

// DeleteRuleType deletes a rule type by ID
func (t *Type) DeleteRuleType(ctx context.Context, ruleTypeID uuid.UUID) error {
	return t.db.querier.DeleteRuleType(ctx, ruleTypeID)
}

// ListRuleTypesByProject returns a list of rule types by project ID
func (t *Type) ListRuleTypesByProject(ctx context.Context, projectID uuid.UUID) ([]*pb.RuleType, error) {
	ret, err := t.db.querier.ListRuleTypesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ruleTypes := make([]*pb.RuleType, len(ret))
	for i, r := range ret {
		rt, err := ruletypes.RuleTypePBFromDB(&r)
		if err != nil {
			return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", r.Name, err)
		}
		ruleTypes[i] = rt
	}
	return ruleTypes, nil
}

// GetRuleTypeByName returns a rule type by name and project IDs
func (t *Type) GetRuleTypeByName(ctx context.Context, projectIDs []uuid.UUID, name string) (*pb.RuleType, error) {
	ret, err := t.db.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Name:     name,
		Projects: projectIDs,
	})
	if err != nil {
		return nil, err
	}
	return ruletypes.RuleTypePBFromDB(&ret)
}

// UpdateRuleType updates a rule type
func (t *Type) UpdateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	return t.ruleSvc.UpdateRuleType(ctx, projectID, subscriptionID, ruleType, t.db.querier)
}

// CreateRuleType creates a rule type
func (t *Type) CreateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	return t.ruleSvc.CreateRuleType(ctx, projectID, subscriptionID, ruleType, t.db.querier)
}
