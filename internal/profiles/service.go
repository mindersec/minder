// Copyright 2024 Stacklok, Inc
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

package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	// ignore this linter warning - this is pre-existing code, and I do not
	// want to change the logging library it uses at this time.
	// nolint:depguard
	"log"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ProfileService encapsulates methods for creating and updating profiles
// TODO: other methods such as deletion and patch should be moved here
type ProfileService interface {
	// CreateProfile creates the profile in the specified project
	// returns the updated profile structure on successful update
	CreateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		profile *minderv1.Profile,
	) (*minderv1.Profile, error)

	// UpdateProfile updates the profile in the specified project
	// returns the updated profile structure on successful update
	UpdateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		provider *db.Provider,
		profile *minderv1.Profile,
	) (*minderv1.Profile, error)
}

type profileService struct {
	store     db.Store
	publisher events.Interface
	validator *Validator
}

// NewProfileService creates an instance of ProfileService
func NewProfileService(
	store db.Store,
	publisher events.Interface,
) ProfileService {
	return &profileService{
		store:     store,
		publisher: publisher,
		validator: NewValidator(store),
	}
}

// Note that there are no unit tests for these methods in this package. They are instead tested by the tests in
// `handlers_profiles`. In order to implement a full test suite for creation and update, further refactoring will be
// needed.

func (p *profileService) CreateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	profile *minderv1.Profile,
) (*minderv1.Profile, error) {

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	rulesInProf, err := p.validator.ValidateAndExtractRules(ctx, projectID, provider.Name, profile)
	if err != nil {
		return nil, err
	}

	// Adds default rule names, if not present
	PopulateRuleNames(profile)

	// Now that we know it's valid, let's persist it!
	tx, err := p.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}
	defer p.store.Rollback(tx)

	qtx := p.store.GetQuerierWithTransaction(tx)

	params := db.CreateProfileParams{
		Provider:   provider.Name,
		ProviderID: provider.ID,
		ProjectID:  projectID,
		Name:       profile.GetName(),
		Remediate:  validateActionType(profile.GetRemediate()),
		Alert:      validateActionType(profile.GetAlert()),
	}

	// Create profile
	newProfile, err := qtx.CreateProfile(ctx, params)
	if db.ErrIsUniqueViolation(err) {
		log.Printf("profile already exists: %v", err)
		return nil, util.UserVisibleError(codes.AlreadyExists, "profile already exists")
	} else if err != nil {
		log.Printf("error creating profile: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	// Create entity rules entries
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       profile.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          profile.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: profile.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      profile.GetPullRequest(),
	} {
		if err := createProfileRulesForEntity(ctx, ent, &newProfile, qtx, entRules, rulesInProf); err != nil {
			return nil, err
		}
	}

	if err := p.store.Commit(tx); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: newProfile.ID}
	p.sendNewProfileEvent(provider.Name, projectID)

	profile.Id = ptr.Ptr(newProfile.ID.String())
	profile.Context = &minderv1.Context{
		Provider: &newProfile.Provider,
		Project:  ptr.Ptr(newProfile.ProjectID.String()),
	}

	return profile, nil
}

// TODO: refactor this to reduce cyclomatic complexity
//
//nolint:gocyclo
func (p *profileService) UpdateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	provider *db.Provider,
	profile *minderv1.Profile,
) (*minderv1.Profile, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	rules, err := p.validator.ValidateAndExtractRules(ctx, projectID, provider.Name, profile)
	if err != nil {
		return nil, err
	}

	tx, err := p.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}
	defer p.store.Rollback(tx)

	qtx := p.store.GetQuerierWithTransaction(tx)

	// Get object and ensure we lock it for update
	oldDBProfile, err := getProfileFromPBForUpdateWithQuerier(ctx, profile, projectID, qtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "error fetching profile to be updated: %v", err)
	}

	// validate update
	if err := validateProfileUpdate(oldDBProfile, profile, projectID, provider); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile update: %v", err)
	}

	// Adds default rule names, if not present
	PopulateRuleNames(profile)

	oldProfile, err := getProfilePBFromDB(ctx, oldDBProfile.ID, projectID, qtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	oldRules, err := p.getRulesFromProfile(ctx, oldProfile, projectID, provider)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "error fetching profile to be updated: %v", err)
	}

	// Update top-level profile db object
	updatedProfile, err := qtx.UpdateProfile(ctx, db.UpdateProfileParams{
		ProjectID: projectID,
		ID:        oldDBProfile.ID,
		Remediate: validateActionType(profile.GetRemediate()),
		Alert:     validateActionType(profile.GetAlert()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	// Create entity rules entries
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       profile.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          profile.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: profile.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      profile.GetPullRequest(),
	} {
		if err := updateProfileRulesForEntity(ctx, ent, &updatedProfile, qtx, entRules, rules); err != nil {
			return nil, err
		}
	}

	unusedRuleStatuses := getUnusedOldRuleStatuses(rules, oldRules)
	unusedRuleTypes := getUnusedOldRuleTypes(rules, oldRules)

	if err := deleteUnusedRulesFromProfile(ctx, &updatedProfile, unusedRuleTypes, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := deleteRuleStatusesForProfile(ctx, &updatedProfile, unusedRuleStatuses, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := p.store.Commit(tx); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}

	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: updatedProfile.Name, ID: updatedProfile.ID}

	profile.Id = ptr.Ptr(updatedProfile.ID.String())
	profile.Context = &minderv1.Context{
		Provider: &updatedProfile.Provider,
		Project:  ptr.Ptr(updatedProfile.ProjectID.String()),
	}

	// re-trigger profile evaluation
	p.sendNewProfileEvent(provider.Name, projectID)

	return profile, nil
}

func createProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
	rulesInProf RuleMapping,
) error {
	if rules == nil {
		return nil
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		log.Printf("error marshalling %s rules: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}
	entProf, err := qtx.CreateProfileForEntity(ctx, db.CreateProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		log.Printf("error creating profile for entity %s: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}

	for idx := range rulesInProf {
		ruleRef := rulesInProf[idx]

		if ruleRef.Entity != entity {
			continue
		}

		ruleID := ruleRef.RuleID

		_, err := qtx.UpsertRuleInstantiation(ctx, db.UpsertRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      ruleID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("the rule instantiation for rule already existed.")
		} else if err != nil {
			log.Printf("error creating rule instantiation: %v", err)
			return status.Errorf(codes.Internal, "error creating profile")
		}
	}

	return err
}

// TODO: DEDUPE
// validateActionType returns the appropriate remediate type or the
// NULL DB type if the input is invalid, thus letting the server run
// the profile with the default remediate type.
func validateActionType(r string) db.NullActionType {
	switch r {
	case "on":
		return db.NullActionType{ActionType: db.ActionTypeOn, Valid: true}
	case "off":
		return db.NullActionType{ActionType: db.ActionTypeOff, Valid: true}
	case "dry_run":
		return db.NullActionType{ActionType: db.ActionTypeDryRun, Valid: true}
	}

	return db.NullActionType{Valid: false}
}

func (p *profileService) sendNewProfileEvent(
	providerName string,
	projectID uuid.UUID,
) {
	// both errors in this case are considered non-fatal
	msg, err := reconcilers.NewProfileInitMessage(providerName, projectID)
	if err != nil {
		log.Printf("error creating reconciler event: %v", err)
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := p.publisher.Publish(reconcilers.InternalProfileInitEventTopic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}
}

func getProfileFromPBForUpdateWithQuerier(
	ctx context.Context,
	profile *minderv1.Profile,
	projectID uuid.UUID,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	if profile.GetId() != "" {
		return getProfileFromPBForUpdateByID(ctx, profile, projectID, querier)
	}

	return getProfileFromPBForUpdateByName(ctx, profile, projectID, querier)
}

func getProfileFromPBForUpdateByID(
	ctx context.Context,
	profile *minderv1.Profile,
	projectID uuid.UUID,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	id, err := uuid.Parse(profile.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	pdb, err := querier.GetProfileByIDAndLock(ctx, db.GetProfileByIDAndLockParams{
		ID:        id,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func getProfileFromPBForUpdateByName(
	ctx context.Context,
	profile *minderv1.Profile,
	projectID uuid.UUID,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	pdb, err := querier.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
		Name:      profile.GetName(),
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func validateProfileUpdate(
	old *db.Profile,
	new *minderv1.Profile,
	projectID uuid.UUID,
	provider *db.Provider,
) error {
	if old.Name != new.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile name")
	}

	if old.ProjectID != projectID {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile project")
	}

	if old.Provider != provider.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile provider")
	}

	return nil
}

// TODO: de-dupe
func getProfilePBFromDB(
	ctx context.Context,
	id uuid.UUID,
	projectID uuid.UUID,
	querier db.ExtendQuerier,
) (*minderv1.Profile, error) {
	profiles, err := querier.GetProfileByProjectAndID(ctx, db.GetProfileByProjectAndIDParams{
		ProjectID: projectID,
		ID:        id,
	})
	if err != nil {
		return nil, err
	}

	pols := engine.MergeDatabaseGetIntoProfiles(profiles)
	if len(pols) == 0 {
		return nil, fmt.Errorf("profile not found")
	} else if len(pols) > 1 {
		return nil, fmt.Errorf("expected only one profile, got %d", len(pols))
	}

	// This should be only one profile
	for _, profile := range pols {
		return profile, nil
	}

	return nil, fmt.Errorf("profile not found")
}

func (p *profileService) getRulesFromProfile(
	ctx context.Context,
	profile *minderv1.Profile,
	projectID uuid.UUID,
	provider *db.Provider,
) (RuleMapping, error) {
	// We capture the rule instantiations here so we can
	// track them in the db later.
	rulesInProf := make(RuleMapping)

	err := engine.TraverseAllRulesForPipeline(profile, func(r *minderv1.Profile_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := p.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider:  provider.Name,
			ProjectID: projectID,
			Name:      r.GetType(),
		})
		if err != nil {
			return fmt.Errorf("error getting rule type %s: %w", r.GetType(), err)
		}

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to pb: %w", rtdb.Name, err)
		}

		key := RuleTypeAndNamePair{
			RuleType: r.GetType(),
			RuleName: ComputeRuleName(r),
		}

		rulesInProf[key] = EntityAndRuleTuple{
			Entity: minderv1.EntityFromString(rtyppb.Def.InEntity),
			RuleID: rtdb.ID,
		}

		return nil
	},
	)

	if err != nil {
		return nil, err
	}

	return rulesInProf, nil
}

func deleteUnusedRulesFromProfile(
	ctx context.Context,
	profile *db.Profile,
	unusedRules []EntityAndRuleTuple,
	querier db.ExtendQuerier,
) error {
	for _, rule := range unusedRules {
		// get entity profile
		log.Printf("getting profile for entity %s", rule.Entity)
		entProf, err := querier.GetProfileForEntity(ctx, db.GetProfileForEntityParams{
			ProfileID: profile.ID,
			Entity:    entities.EntityTypeToDB(rule.Entity),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("skipping rule deletion for entity %s, profile not found", rule.Entity)
				continue
			}
			log.Printf("error getting profile for entity %s: %v", rule.Entity, err)
			return fmt.Errorf("error getting profile for entity %s: %w", rule.Entity, err)
		}

		log.Printf("deleting rule instantiation for rule %s for entity profile %s", rule.RuleID, entProf.ID)
		if err := querier.DeleteRuleInstantiation(ctx, db.DeleteRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      rule.RuleID,
		}); err != nil {
			log.Printf("error deleting rule instantiation: %v", err)
			return fmt.Errorf("error deleting rule instantiation: %w", err)
		}
	}

	return nil
}

func updateProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
	rulesInProf RuleMapping,
) error {
	if len(rules) == 0 {
		return qtx.DeleteProfileForEntity(ctx, db.DeleteProfileForEntityParams{
			ProfileID: profile.ID,
			Entity:    entities.EntityTypeToDB(entity),
		})
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		log.Printf("error marshalling %s rules: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}
	entProf, err := qtx.UpsertProfileForEntity(ctx, db.UpsertProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		log.Printf("error updating profile for entity %s: %v", entity, err)
		return err
	}

	for idx := range rulesInProf {
		ruleRef := rulesInProf[idx]

		if ruleRef.Entity != entity {
			continue
		}

		_, err := qtx.UpsertRuleInstantiation(ctx, db.UpsertRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      ruleRef.RuleID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("the rule instantiation for rule already existed.")
		} else if err != nil {
			log.Printf("error creating rule instantiation: %v", err)
			return status.Errorf(codes.Internal, "error updating profile")
		}
	}

	return err
}

func getUnusedOldRuleStatuses(
	newRules, oldRules RuleMapping,
) RuleMapping {
	unusedRuleStatuses := make(RuleMapping)

	for ruleTypeAndName, rule := range oldRules {
		if _, ok := newRules[ruleTypeAndName]; !ok {
			unusedRuleStatuses[ruleTypeAndName] = rule
		}
	}

	return unusedRuleStatuses
}

func getUnusedOldRuleTypes(newRules, oldRules RuleMapping) []EntityAndRuleTuple {
	var unusedRuleTypes []EntityAndRuleTuple

	oldRulesTypeMap := make(map[string]EntityAndRuleTuple)
	for ruleTypeAndName, rule := range oldRules {
		oldRulesTypeMap[ruleTypeAndName.RuleType] = rule
	}

	newRulesTypeMap := make(map[string]EntityAndRuleTuple)
	for ruleTypeAndName, rule := range newRules {
		newRulesTypeMap[ruleTypeAndName.RuleType] = rule
	}

	for ruleType, rule := range oldRulesTypeMap {
		if _, ok := newRulesTypeMap[ruleType]; !ok {
			unusedRuleTypes = append(unusedRuleTypes, rule)
		}
	}

	return unusedRuleTypes
}

func deleteRuleStatusesForProfile(
	ctx context.Context,
	profile *db.Profile,
	unusedRuleStatuses RuleMapping,
	querier db.ExtendQuerier,
) error {
	for ruleTypeAndName, rule := range unusedRuleStatuses {
		log.Printf("deleting rule evaluations for rule %s in profile %s", rule.RuleID, profile.ID)

		if err := querier.DeleteRuleStatusesForProfileAndRuleType(ctx, db.DeleteRuleStatusesForProfileAndRuleTypeParams{
			ProfileID:  profile.ID,
			RuleTypeID: rule.RuleID,
			RuleName:   ruleTypeAndName.RuleName,
		}); err != nil {
			log.Printf("error deleting rule evaluations: %v", err)
			return fmt.Errorf("error deleting rule evaluations: %w", err)
		}
	}

	return nil
}
