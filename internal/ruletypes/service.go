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

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/marketplaces/namespaces"
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
		projectID uuid.UUID,
		provider *db.Provider,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)

	// CreateSubscriptionRuleType creates rule types in the database
	// the new rule type is validated
	// if the rule type already exists - this will return an error
	// returns the pb definition of the new rule type on success
	CreateSubscriptionRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		// TODO: should this be a whole instance of the subscription?
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)

	// UpdateRuleType updates rule types in the database
	// the new rule type is validated, and backwards compatibility verified
	// if the rule does not already exist - this will return an error
	// returns the pb definition of the updated rule type on success
	UpdateRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)

	// UpdateSubscriptionRuleType updates rule types in the database
	// the new rule type is validated, and backwards compatibility verified
	// if the rule does not already exist - this will return an error
	// returns the pb definition of the updated rule type on success
	UpdateSubscriptionRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
	) (*pb.RuleType, error)

	// UpsertSubscriptionRuleType creates the rule type if it does not exist
	// or updates it if it already exists. This is used in the subscription
	// logic.
	UpsertSubscriptionRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
	) error
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
	projectID uuid.UUID,
	provider *db.Provider,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	return r.CreateSubscriptionRuleType(ctx, projectID, provider, uuid.Nil, ruleType)
}

func (r *ruleTypeService) CreateSubscriptionRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	if err := namespaces.ValidateNamespacedNameRules(ruleType.GetName(), subscriptionID); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	_, err := r.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  provider.Name,
		ProjectID: projectID,
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
		Name:           ruleTypeName,
		Provider:       provider.Name,
		ProviderID:     provider.ID,
		ProjectID:      projectID,
		Description:    ruleType.GetDescription(),
		Definition:     serializedRule,
		Guidance:       ruleType.GetGuidance(),
		SeverityValue:  *severity,
		SubscriptionID: uuid.NullUUID{UUID: subscriptionID, Valid: subscriptionID != uuid.Nil},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rule type: %w", err)
	}

	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: newDBRecord.Name, ID: newDBRecord.ID}

	rt, err := engine.RuleTypePBFromDB(&newDBRecord)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", newDBRecord.Name, err)
	}

	return rt, nil
}

func (r *ruleTypeService) UpdateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	return r.UpdateSubscriptionRuleType(ctx, projectID, provider, uuid.Nil, ruleType)
}

func (r *ruleTypeService) UpdateSubscriptionRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) (*pb.RuleType, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	oldRuleType, err := r.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  provider.Name,
		ProjectID: projectID,
		Name:      ruleTypeName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("failed to get rule type: %w", err)
	}

	if err = namespaces.DoesSubscriptionIDMatch(subscriptionID, oldRuleType.SubscriptionID); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	// extra validation applies when updating rules to make sure the update
	// does not break profiles which use the rule
	err = validateRuleUpdate(&oldRuleType, ruleType)
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
		ID:            oldRuleType.ID,
		Description:   ruleType.GetDescription(),
		Definition:    serializedRule,
		SeverityValue: *severity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update rule type: %w", err)
	}

	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: oldRuleType.Name, ID: oldRuleType.ID}

	result, err := engine.RuleTypePBFromDB(&updatedRuleType)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", oldRuleType.Name, err)
	}

	return result, nil
}

func (s *ruleTypeService) UpsertSubscriptionRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
) error {
	// In future, we may want to refactor the code so that we use upserts
	// instead of separate create and update methods. For now, simulate upsert
	// semantics by trying to create, then trying to update.
	_, err := s.CreateSubscriptionRuleType(ctx, projectID, provider, subscriptionID, ruleType)
	if err == nil {
		// Rule successfully created, we can stop here.
		return nil
	} else if !errors.Is(err, ErrRuleAlreadyExists) {
		return fmt.Errorf("error while creating rule: %w", err)
	}

	// If we get here: rule already exists. Let's update it.
	_, err = s.UpdateSubscriptionRuleType(ctx, projectID, provider, subscriptionID, ruleType)
	if err != nil {
		return fmt.Errorf("error while updating rule: %w", err)
	}
	return nil
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
