// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ruletypes contains logic relating to the management of rule types in
// minder
package ruletypes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/schemaupdate"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleTypeService encapsulates the creation and update of rule types
// TODO: in future, other operations such as delete should be moved here
type RuleTypeService interface {
	// CreateRuleType creates rule types in the database
	// the new rule type is validated
	// if the rule type already exists - this will return an error
	// returns the pb definition of the new rule type on success
	CreateRuleType(
		ctx context.Context,
		provider db.Provider,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)

	// UpdateRuleType updates rule types in the database
	// the new rule type is validated, and backwards compatibility verified
	// if the rule does not already exist - this will return an error
	// returns the pb definition of the updated rule type on success
	UpdateRuleType(
		ctx context.Context,
		provider db.Provider,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)
}

type ruleTypeService struct {
	store db.Store
}

// NewRuleTypeService creates a new instance of RuleTypeService
func NewRuleTypeService(store db.Store) RuleTypeService {
	return &ruleTypeService{store: store}
}

var (
	// ErrRuleNotFound is returned by the update method if the rule does not
	// already exist
	ErrRuleNotFound = errors.New("rule type not found")
	// ErrRuleAlreadyExists is returned by the create method if the rule
	// already exists
	ErrRuleAlreadyExists = errors.New("rule type already exists")
	// ErrRuleTypeInvalid is returned by both create and update if validation
	// fails
	ErrRuleTypeInvalid = errors.New("rule type validation failed")
)

func (r *ruleTypeService) CreateRuleType(
	ctx context.Context,
	provider db.Provider,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	_, err := r.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  provider.Name,
		ProjectID: provider.ProjectID,
		Name:      ruleTypeName,
	})
	if err == nil {
		return nil, ErrRuleAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get rule type: %w", err)
	}

	serializedRule, err := util.GetBytesFromProto(ruleTypeDef)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule definition to db: %w", err)
	}

	severity, err := getRuleTypeSeverity(ruleType.GetSeverity())
	if err != nil {
		return nil, err
	}

	newDBRecord, err := r.store.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:          ruleTypeName,
		Provider:      provider.Name,
		ProviderID:    provider.ID,
		ProjectID:     provider.ProjectID,
		Description:   ruleType.GetDescription(),
		Definition:    serializedRule,
		Guidance:      ruleType.GetGuidance(),
		SeverityValue: *severity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rule type: %w", err)
	}

	rt, err := engine.RuleTypePBFromDB(&newDBRecord)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", newDBRecord.Name, err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = newDBRecord.Provider
	logger.BusinessRecord(ctx).Project = newDBRecord.ProjectID
	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: newDBRecord.Name, ID: newDBRecord.ID}

	return rt, nil
}

func (r *ruleTypeService) UpdateRuleType(
	ctx context.Context,
	provider db.Provider,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	existingRuleType, err := r.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  provider.Name,
		ProjectID: provider.ProjectID,
		Name:      ruleTypeName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("failed to get rule type: %w", err)
	}

	// extra validation applies when updating rules to make sure the update
	// does not break profiles which use the rule
	err = validateRuleUpdate(&existingRuleType, ruleType)
	if err != nil {
		return nil, err
	}

	serializedRule, err := util.GetBytesFromProto(ruleTypeDef)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule definition to db: %w", err)
	}

	severity, err := getRuleTypeSeverity(ruleType.GetSeverity())
	if err != nil {
		return nil, err
	}

	updatedRuleType, err := r.store.UpdateRuleType(ctx, db.UpdateRuleTypeParams{
		ID:            existingRuleType.ID,
		Description:   ruleType.GetDescription(),
		Definition:    serializedRule,
		SeverityValue: *severity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update rule type: %w", err)
	}

	result, err := engine.RuleTypePBFromDB(&updatedRuleType)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", existingRuleType.Name, err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = existingRuleType.Provider
	logger.BusinessRecord(ctx).Project = existingRuleType.ProjectID
	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: existingRuleType.Name, ID: existingRuleType.ID}

	return result, nil
}

func getRuleTypeSeverity(severity *pb.Severity) (*db.Severity, error) {
	sev := severity.InitializedStringValue()
	var seval db.Severity

	if err := seval.Scan(sev); err != nil {
		// errors from the `Scan` method appear to be caused entirely by bad
		// input
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	return &seval, nil
}

func validateRuleUpdate(existingRecord *db.RuleType, newRuleType *pb.RuleType) error {
	oldRuleType, err := engine.RuleTypePBFromDB(existingRecord)
	if err != nil {
		return fmt.Errorf("cannot convert rule type %s to pb: %w", newRuleType.GetName(), err)
	}

	oldDef := oldRuleType.GetDef()
	newDef := newRuleType.GetDef()

	// In case we have profiles that use this rule type, we need to validate
	// that the incoming rule is valid against the old rule. Unlike previous
	// iterations of this code, the checks are carried out irrespective of
	// whether any profiles currently use this rule type.
	if err := schemaupdate.ValidateSchemaUpdate(oldDef.GetRuleSchema(), newDef.GetRuleSchema()); err != nil {
		return fmt.Errorf("%w: rule schema update is invalid: %w", ErrRuleTypeInvalid, err)
	}
	if err := schemaupdate.ValidateSchemaUpdate(oldDef.GetParamSchema(), newDef.GetParamSchema()); err != nil {
		return fmt.Errorf("%w: parameter schema update is invalid: %w", ErrRuleTypeInvalid, err)
	}

	return nil
}
