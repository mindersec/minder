// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletypes contains logic relating to the management of rule types in
// minder
package ruletypes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/marketplaces/namespaces"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/schemaupdate"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

var (
	// ErrDataSourceNotFound is returned when a data source is not found
	ErrDataSourceNotFound = errors.New("data source not found")
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
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
		qtx db.Querier,
	) (*pb.RuleType, error)

	// UpdateRuleType updates rule types in the database
	// the new rule type is validated, and backwards compatibility verified
	// if the rule does not already exist - this will return an error
	// returns the pb definition of the updated rule type on success
	UpdateRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
		qtx db.Querier,
	) (*pb.RuleType, error)

	// UpsertRuleType creates the rule type if it does not exist
	// or updates it if it already exists. This is used in the subscription
	// logic.
	UpsertRuleType(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		ruleType *pb.RuleType,
		qtx db.Querier,
	) error
}

type ruleTypeService struct{}

// NewRuleTypeService creates a new instance of RuleTypeService
func NewRuleTypeService() RuleTypeService {
	return &ruleTypeService{}
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

func (_ *ruleTypeService) CreateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
	qtx db.Querier,
) (*pb.RuleType, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	if err := namespaces.ValidateNamespacedNameRules(ruleType.GetName(), subscriptionID); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	projects, err := qtx.GetParentProjects(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent projects: %w", err)
	}

	_, err = qtx.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Projects: projects,
		Name:     ruleTypeName,
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

	releasePhase, err := GetDBReleaseStatusFromPBReleasePhase(ruleType.GetReleasePhase())
	if err != nil {
		return nil, err
	}

	ruleType = ruleType.WithDefaultDisplayName().WithDefaultShortFailureMessage()
	newDBRecord, err := qtx.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:                ruleTypeName,
		DisplayName:         ruleType.GetDisplayName(),
		ShortFailureMessage: ruleType.GetShortFailureMessage(),
		ProjectID:           projectID,
		Description:         ruleType.GetDescription(),
		Definition:          serializedRule,
		Guidance:            ruleType.GetGuidance(),
		SeverityValue:       *severity,
		SubscriptionID:      uuid.NullUUID{UUID: subscriptionID, Valid: subscriptionID != uuid.Nil},
		ReleasePhase:        *releasePhase,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create rule type: %w", err)
	}

	// Data Sources reference update. Note that this step can be
	// safely performed after updating the rule, as the only thing
	// we need from the previous code is project id and rule id.
	ds := ruleTypeDef.GetEval().GetDataSources()
	if err := processDataSources(ctx, newDBRecord.ID, ds, projectID, projects, qtx); err != nil {
		return nil, fmt.Errorf("failed adding references to data sources: %w", err)
	}

	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: newDBRecord.Name, ID: newDBRecord.ID}

	rt, err := RuleTypePBFromDB(&newDBRecord)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", newDBRecord.Name, err)
	}

	return rt, nil
}

func (_ *ruleTypeService) UpdateRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
	qtx db.Querier,
) (*pb.RuleType, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	if err := ruleType.Validate(); err != nil {
		return nil, errors.Join(ErrRuleTypeInvalid, err)
	}

	ruleTypeName := ruleType.GetName()
	ruleTypeDef := ruleType.GetDef()

	oldRuleType, err := qtx.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		// we only need to check the project that the rule type is in
		Projects: []uuid.UUID{projectID},
		Name:     ruleTypeName,
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

	releasePhase, err := GetDBReleaseStatusFromPBReleasePhase(ruleType.GetReleasePhase())
	if err != nil {
		return nil, err
	}

	ruleType = ruleType.WithDefaultDisplayName().WithDefaultShortFailureMessage()
	updatedRuleType, err := qtx.UpdateRuleType(ctx, db.UpdateRuleTypeParams{
		ID:                  oldRuleType.ID,
		Description:         ruleType.GetDescription(),
		Definition:          serializedRule,
		SeverityValue:       *severity,
		DisplayName:         ruleType.GetDisplayName(),
		ShortFailureMessage: ruleType.GetShortFailureMessage(),
		ReleasePhase:        *releasePhase,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update rule type: %w", err)
	}

	projects, err := qtx.GetParentProjects(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent projects: %w", err)
	}

	// Data Sources reference update. Note that this step can be
	// safely performed after updating the rule, as the only thing
	// we need from the previous code is project id and rule id.
	ds := ruleTypeDef.GetEval().GetDataSources()
	if err := processDataSources(ctx, oldRuleType.ID, ds, projectID, projects, qtx); err != nil {
		return nil, fmt.Errorf("failed updating references to data sources: %w", err)
	}

	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: oldRuleType.Name, ID: oldRuleType.ID}

	result, err := RuleTypePBFromDB(&updatedRuleType)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %w", oldRuleType.Name, err)
	}

	return result, nil
}

func (s *ruleTypeService) UpsertRuleType(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	ruleType *pb.RuleType,
	qtx db.Querier,
) error {
	// In future, we may want to refactor the code so that we use upserts
	// instead of separate create and update methods. For now, simulate upsert
	// semantics by trying to create, then trying to update.
	_, err := s.CreateRuleType(ctx, projectID, subscriptionID, ruleType, qtx)
	if err == nil {
		// Rule successfully created, we can stop here.
		return nil
	} else if !errors.Is(err, ErrRuleAlreadyExists) {
		return fmt.Errorf("error while creating rule: %w", err)
	}

	// If we get here: rule already exists. Let's update it.
	_, err = s.UpdateRuleType(ctx, projectID, subscriptionID, ruleType, qtx)
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
	oldRuleType, err := RuleTypePBFromDB(existingRecord)
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

func processDataSources(
	ctx context.Context,
	ruleID uuid.UUID,
	ds []*pb.DataSourceReference,
	projectID uuid.UUID,
	projectHierarchy []uuid.UUID,
	qtx db.Querier,
) error {
	// We first verify that the data sources required are
	// available within the project hierarchy.
	datasources, err := getAvailableDataSources(ctx, ds, projectHierarchy, qtx)
	if err != nil {
		// We already have enough context. Let's not over-wrap the error.
		return err
	}

	// Then, we proceed to delete any data source reference we
	// have for the old definition of the rule type.
	deleteArgs := db.DeleteRuleTypeDataSourceParams{
		Ruleid:    ruleID,
		Projectid: projectID,
	}
	if err := qtx.DeleteRuleTypeDataSource(ctx, deleteArgs); err != nil {
		return fmt.Errorf("error deleting references to data source: %w", err)
	}

	// Finally, we add references to the required data source.
	for _, datasource := range datasources {
		insertArgs := db.AddRuleTypeDataSourceReferenceParams{
			Ruletypeid:   ruleID,
			Datasourceid: datasource.ID,
			Projectid:    projectID,
		}
		if _, err := qtx.AddRuleTypeDataSourceReference(ctx, insertArgs); err != nil {
			return fmt.Errorf("error adding references to data source: %w", err)
		}
	}

	return nil
}

func getAvailableDataSources(
	ctx context.Context,
	requiredDataSources []*pb.DataSourceReference,
	projects []uuid.UUID,
	qtx db.Querier,
) ([]db.DataSource, error) {
	datasources := make([]db.DataSource, 0)

	for _, datasource := range requiredDataSources {
		qarg := db.GetDataSourceByNameParams{
			Name:     datasource.Name,
			Projects: projects,
		}
		dbDataSource, err := qtx.GetDataSourceByName(ctx, qarg)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrDataSourceNotFound, datasource.Name)
		}
		if err != nil {
			return nil, fmt.Errorf("failed getting data sources: %w", err)
		}

		datasources = append(datasources, dbDataSource)
	}

	return datasources, nil
}
