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
	ListRuleTypesReferencesByDataSource(ctx context.Context, dataSourceID uuid.UUID) ([]uuid.UUID, error)
	CreateRuleType(ctx context.Context, projectID uuid.UUID, subscriptionID uuid.UUID, ruleType *pb.RuleType) (*pb.RuleType, error)
	UpdateRuleType(ctx context.Context, projectID uuid.UUID, subscriptionID uuid.UUID, ruleType *pb.RuleType) (*pb.RuleType, error)
	DeleteRuleType(ctx context.Context, ruleTypeID uuid.UUID) error
	GetRuleTypeByName(ctx context.Context, projectIDs []uuid.UUID, name string) (*pb.RuleType, error)
}

// DeleteRuleType deletes a rule type by ID
func (q *querierType) DeleteRuleType(ctx context.Context, ruleTypeID uuid.UUID) error {
	if q.querier == nil {
		return ErrQuerierMissing
	}
	return q.querier.DeleteRuleType(ctx, ruleTypeID)
}

// ListRuleTypesByProject returns a list of rule types by project ID
func (q *querierType) ListRuleTypesByProject(ctx context.Context, projectID uuid.UUID) ([]*pb.RuleType, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	ret, err := q.querier.ListRuleTypesByProject(ctx, projectID)
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
func (q *querierType) GetRuleTypeByName(ctx context.Context, projectIDs []uuid.UUID, name string) (*pb.RuleType, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	ret, err := q.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Name:     name,
		Projects: projectIDs,
	})
	if err != nil {
		return nil, err
	}
	return ruletypes.RuleTypePBFromDB(&ret)
}

// UpdateRuleType updates a rule type
func (q *querierType) UpdateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.ruleSvc == nil {
		return nil, ErrRuleSvcMissing
	}
	return q.ruleSvc.UpdateRuleType(ctx, projectID, subscriptionID, ruleType, q.querier)
}

// CreateRuleType creates a rule type
func (q *querierType) CreateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.ruleSvc == nil {
		return nil, ErrRuleSvcMissing
	}
	return q.ruleSvc.CreateRuleType(ctx, projectID, subscriptionID, ruleType, q.querier)
}

// ListRuleTypesReferencesByDataSource returns a list of rule types using a data source
func (q *querierType) ListRuleTypesReferencesByDataSource(ctx context.Context, dataSourceID uuid.UUID) ([]uuid.UUID, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	ruleTypes, err := q.querier.ListRuleTypesReferencesByDataSource(ctx, dataSourceID)
	if err != nil {
		return nil, err
	}

	// Convert ruleTypes to a slice of strings
	ruleTypeIds := make([]uuid.UUID, len(ruleTypes))
	for i, r := range ruleTypes {
		ruleTypeIds[i] = r.RuleTypeID
	}

	return ruleTypeIds, nil
}
