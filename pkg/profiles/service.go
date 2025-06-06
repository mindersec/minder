// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/marketplaces/namespaces"
	"github.com/mindersec/minder/internal/reconcilers"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/selectors"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// ProfileService encapsulates methods for creating and updating profiles
// TODO: other methods such as deletion and patch should be moved here
type ProfileService interface {
	// CreateProfile creates the profile in the specified project
	// returns the updated profile structure on successful update
	// subscriptionID should be set to nil when not calling
	CreateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		profile *minderv1.Profile,
		qtx db.Querier,
	) (*minderv1.Profile, error)

	// UpdateProfile updates the profile in the specified project
	// returns the updated profile structure on successful update
	UpdateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		profile *minderv1.Profile,
		qtx db.Querier,
	) (*minderv1.Profile, error)

	// PatchProfile updates the profile (as either a UUID or name)
	// in the specified project by applying the changes in the
	// provided profile structure as specified by the updateMask
	PatchProfile(
		ctx context.Context,
		projectID uuid.UUID,
		profileID string,
		profile *minderv1.Profile,
		updateMask *fieldmaskpb.FieldMask,
		qtx db.Querier,
	) (*minderv1.Profile, error)

	DeleteProfile(
		ctx context.Context,
		projectID uuid.UUID,
		profile string,
		qtx db.Querier,
	) (*db.Profile, error)
}

type profileService struct {
	publisher interfaces.Publisher
	validator *Validator
}

// NewProfileService creates an instance of ProfileService
func NewProfileService(
	publisher interfaces.Publisher,
	selChecker selectors.SelectionChecker,
) ProfileService {
	return &profileService{
		publisher: publisher,
		validator: NewValidator(selChecker),
	}
}

// Note that there are no unit tests for these methods in this package. They are instead tested by the tests in
// `handlers_profiles`. In order to implement a full test suite for creation and update, further refactoring will be
// needed.

func (p *profileService) CreateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *minderv1.Profile,
	qtx db.Querier,
) (*minderv1.Profile, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	rulesInProf, err := p.validator.ValidateAndExtractRules(ctx, qtx, projectID, profile)
	if err != nil {
		return nil, err
	}

	if err = namespaces.ValidateNamespacedNameRules(profile.GetName(), subscriptionID); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "name failed namespace validation: %v", err)
	}

	if err = namespaces.ValidateLabelsPresence(profile.GetLabels(), subscriptionID); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "labels failed namespace validation: %v", err)
	}

	// Adds default rule names, if not present
	PopulateRuleNames(profile, rulesInProf)

	displayName := profile.GetDisplayName()

	listParams := db.ListProfilesByProjectIDAndLabelParams{
		ProjectID: projectID,
	}

	existingProfiles, err := qtx.ListProfilesByProjectIDAndLabel(ctx, listParams)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get profiles: %s", err)
	}

	profileMap := MergeDatabaseListIntoProfiles(existingProfiles)

	existingProfileNames := make([]string, 0, len(profileMap))

	// Derive the profile name from the profile display name
	name := DeriveProfileNameFromDisplayName(profile, existingProfileNames)

	// if empty use the name
	if displayName == "" {
		displayName = profile.GetName()
	}

	params := db.CreateProfileParams{
		ProjectID:      projectID,
		Name:           name,
		DisplayName:    displayName,
		Labels:         profile.GetLabels(),
		Remediate:      db.ValidateRemediateType(profile.GetRemediate()),
		Alert:          db.ValidateAlertType(profile.GetAlert()),
		SubscriptionID: uuid.NullUUID{UUID: subscriptionID, Valid: subscriptionID != uuid.Nil},
	}

	// Create profile
	newProfile, err := qtx.CreateProfile(ctx, params)
	if db.ErrIsUniqueViolation(err) {
		zerolog.Ctx(ctx).Warn().Str("profile", name).Msgf("profile already exists: %v", err)
		return nil, util.UserVisibleError(codes.AlreadyExists, "profile already exists")
	} else if err != nil {
		zerolog.Ctx(ctx).Warn().Str("profile", name).Msgf("error creating profile: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	// Create entity rules entries
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       profile.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          profile.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: profile.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      profile.GetPullRequest(),
		minderv1.Entity_ENTITY_RELEASE:            profile.GetRelease(),
		minderv1.Entity_ENTITY_PIPELINE_RUN:       profile.GetPipelineRun(),
		minderv1.Entity_ENTITY_TASK_RUN:           profile.GetTaskRun(),
		minderv1.Entity_ENTITY_BUILD:              profile.GetBuild(),
	} {
		if err := createProfileRulesForEntity(ctx, ent, &newProfile, qtx, entRules, rulesInProf); err != nil {
			return nil, err
		}
	}

	if err := p.createSelectors(ctx, newProfile.ID, qtx, profile.GetSelection()); err != nil {
		return nil, err
	}

	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: newProfile.ID}
	p.sendNewProfileEvent(ctx, projectID)

	profile.Id = ptr.Ptr(newProfile.ID.String())
	profile.Context = &minderv1.Context{
		Project: ptr.Ptr(newProfile.ProjectID.String()),
	}

	profile.Remediate = ptr.Ptr(string(newProfile.Remediate.ActionType))
	profile.Alert = ptr.Ptr(string(newProfile.Alert.ActionType))

	return profile, nil
}

func (p *profileService) UpdateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *minderv1.Profile,
	qtx db.Querier,
) (*minderv1.Profile, error) {
	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	rules, err := p.validator.ValidateAndExtractRules(ctx, qtx, projectID, profile)
	if err != nil {
		return nil, err
	}

	// Get object and ensure we lock it for update
	oldDBProfile, err := getProfileFromPBForUpdateWithQuerier(ctx, profile, projectID, qtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "error fetching profile to be updated: %v", err)
	}

	if err = namespaces.DoesSubscriptionIDMatch(subscriptionID, oldDBProfile.SubscriptionID); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "namespace validation failed: %v", err)
	}

	// validate update
	if err = validateProfileUpdate(oldDBProfile, profile, projectID); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile update: %v", err)
	}

	// Adds default rule names, if not present
	PopulateRuleNames(profile, rules)

	displayName := profile.GetDisplayName()
	// if empty use the name
	if displayName == "" {
		displayName = profile.GetName()
	}

	// Update top-level profile db object
	updatedProfile, err := qtx.UpdateProfile(ctx, db.UpdateProfileParams{
		ProjectID:   projectID,
		ID:          oldDBProfile.ID,
		DisplayName: displayName,
		Labels:      profile.GetLabels(),
		Remediate:   db.ValidateRemediateType(profile.GetRemediate()),
		Alert:       db.ValidateAlertType(profile.GetAlert()),
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
		minderv1.Entity_ENTITY_RELEASE:            profile.GetRelease(),
		minderv1.Entity_ENTITY_PIPELINE_RUN:       profile.GetPipelineRun(),
		minderv1.Entity_ENTITY_TASK_RUN:           profile.GetTaskRun(),
		minderv1.Entity_ENTITY_BUILD:              profile.GetBuild(),
	} {
		if err = updateProfileRulesForEntity(ctx, ent, &updatedProfile, qtx, entRules); err != nil {
			return nil, err
		}

		updatedIDs, err := upsertRuleInstances(
			ctx,
			qtx,
			updatedProfile.ID,
			updatedProfile.ProjectID,
			entRules,
			entities.EntityTypeToDB(ent),
			rules,
		)
		if err != nil {
			return nil, err
		}

		// Any rule which was not updated was deleted from the profile.
		// Remove from the database as well.
		err = qtx.DeleteNonUpdatedRules(ctx, db.DeleteNonUpdatedRulesParams{
			ProfileID:  updatedProfile.ID,
			EntityType: entities.EntityTypeToDB(ent),
			UpdatedIds: updatedIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("error while cleaning up rule instances: %w", err)
		}
	}

	if err := p.updateSelectors(ctx, updatedProfile.ID, qtx, profile.GetSelection()); err != nil {
		return nil, err
	}

	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: updatedProfile.Name, ID: updatedProfile.ID}

	profile.Id = ptr.Ptr(updatedProfile.ID.String())
	profile.Context = &minderv1.Context{
		Project: ptr.Ptr(updatedProfile.ProjectID.String()),
	}

	profile.Remediate = ptr.Ptr(string(updatedProfile.Remediate.ActionType))
	profile.Alert = ptr.Ptr(string(updatedProfile.Alert.ActionType))

	// re-trigger profile evaluation
	p.sendNewProfileEvent(ctx, projectID)

	return profile, nil
}

func (p *profileService) PatchProfile(
	ctx context.Context,
	projectID uuid.UUID,
	profile string,
	patch *minderv1.Profile,
	updateMask *fieldmaskpb.FieldMask,
	qtx db.Querier,
) (*minderv1.Profile, error) {
	var dbProfile db.Profile
	profileID, err := uuid.Parse(profile)
	if err != nil {
		// if the profile is not a valid UUID, try to look it up by name
		dbProfile, err = qtx.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
			ProjectID: projectID,
			Name:      profile,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "profile %q not found", profile)
			}
			return nil, status.Errorf(codes.Internal, "error fetching profile to be patched: %v", err)
		}
		profileID = dbProfile.ID
	}
	baseProfilePb, err := getProfilePBFromDB(ctx, profileID, projectID, qtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile %s: %w", profileID, err)
	}

	patchProfilePb(baseProfilePb, patch, updateMask)

	return p.UpdateProfile(ctx, projectID, uuid.Nil, baseProfilePb, qtx)
}

func patchProfilePb(oldProfilePb, patchPb *minderv1.Profile, updateMask *fieldmaskpb.FieldMask) {
	// if there is no update mask, there's no changes. The grpc-rest gateway always sets the update mask
	if updateMask == nil {
		return
	}

	oldReflect := oldProfilePb.ProtoReflect()
	patchReflect := patchPb.ProtoReflect()

	for _, attr := range updateMask.Paths {
		fieldDesc := patchReflect.Descriptor().Fields().ByName(protoreflect.Name(attr))
		if fieldDesc == nil {
			continue
		}

		copyFieldValue(oldReflect, patchReflect, fieldDesc)
	}
}

// DeleteProfile deletes the profile in the specified project.  profile may be either
// the ID of the profile or the name of the profile, which will be looked up if needed.
// This function assumes that any transactions are externally managed by the supplied qtx.
func (*profileService) DeleteProfile(
	ctx context.Context,
	projectID uuid.UUID,
	profile string,
	qtx db.Querier,
) (*db.Profile, error) {
	var dbProfile db.Profile
	profileID, err := uuid.Parse(profile)
	if err == nil {
		// if the profile is a valid UUID, look it up by ID
		dbProfile, err = qtx.GetProfileByIDAndLock(ctx, db.GetProfileByIDAndLockParams{
			ProjectID: projectID,
			ID:        profileID,
		})
	} else {
		// if the profile is not a valid UUID, try to look it up by name
		dbProfile, err = qtx.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
			ProjectID: projectID,
			Name:      profile,
		})
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile %q not found", profile)
		}
		return nil, status.Errorf(codes.Internal, "error fetching profile to be deleted: %v", err)
	}

	// TEMPORARY HACK: Since we do not need to support the deletion of bundle
	// profile yet, reject deletion requests in the API
	// TODO: Move this deletion logic to ProfileService
	if dbProfile.SubscriptionID.Valid {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete profile from bundle")
	}

	err = qtx.DeleteProfile(ctx, db.DeleteProfileParams{
		ProjectID: projectID,
		ID:        dbProfile.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete profile: %s", err)
	}

	return &dbProfile, nil
}

// this is NOT a generic function, it only works because our Profiles only contain repeated or scalars.
func copyFieldValue(dstReflect, srcReflect protoreflect.Message, fieldDesc protoreflect.FieldDescriptor) {
	if fieldDesc.Cardinality() == protoreflect.Repeated {
		srcList := srcReflect.Get(fieldDesc).List()

		// truncate the destination list to zero
		dstList := dstReflect.Mutable(fieldDesc).List()
		dstList.Truncate(0)

		// append all elements from the source list to the destination list
		// effectively replacing the destination list with the source list
		for i := 0; i < srcList.Len(); i++ {
			dstList.Append(srcList.Get(i))
		}
	} else {
		dstReflect.Set(fieldDesc, srcReflect.Get(fieldDesc))
	}
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

	_, err := upsertRuleInstances(
		ctx,
		qtx,
		profile.ID,
		profile.ProjectID,
		rules,
		entities.EntityTypeToDB(entity),
		rulesInProf,
	)
	if err != nil {
		return fmt.Errorf("error while creating rule instances: %w", err)
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error marshalling %s rules", entity)
		return status.Errorf(codes.Internal, "error creating profile: %s", err)
	}
	_, err = qtx.CreateProfileForEntity(ctx, db.CreateProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error creating profile for entity %s", entity)
		return status.Errorf(codes.Internal, "error creating profile: %s", err)
	}

	return err
}

func (p *profileService) sendNewProfileEvent(
	ctx context.Context,
	projectID uuid.UUID,
) {
	// both errors in this case are considered non-fatal
	msg, err := reconcilers.NewProfileInitMessage(projectID)
	if err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error creating reconciler event")
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := p.publisher.Publish(constants.TopicQueueReconcileProfileInit, msg); err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error publishing reconciler event")
	}
}

func getProfileFromPBForUpdateWithQuerier(
	ctx context.Context,
	profile *minderv1.Profile,
	projectID uuid.UUID,
	querier db.Querier,
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
	querier db.Querier,
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
	querier db.Querier,
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
	newProfile *minderv1.Profile,
	projectID uuid.UUID,
) error {
	if old.Name != newProfile.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile name")
	}

	if old.ProjectID != projectID {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile project")
	}

	if err := namespaces.ValidateLabelsUpdate(newProfile.GetLabels(), old.Labels); err != nil {
		return util.UserVisibleError(codes.InvalidArgument, "labels update failed validation: %v", err)
	}

	return nil
}

// TODO: de-dupe
func getProfilePBFromDB(
	ctx context.Context,
	id uuid.UUID,
	projectID uuid.UUID,
	querier db.Querier,
) (*minderv1.Profile, error) {
	profiles, err := querier.GetProfileByProjectAndID(ctx, db.GetProfileByProjectAndIDParams{
		ProjectID: projectID,
		ID:        id,
	})
	if err != nil {
		return nil, err
	}

	pols := MergeDatabaseGetIntoProfiles(profiles)
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

func updateProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
) error {
	if len(rules) == 0 {
		return qtx.DeleteProfileForEntity(ctx, db.DeleteProfileForEntityParams{
			ProfileID: profile.ID,
			Entity:    entities.EntityTypeToDB(entity),
		})
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error marshalling %s rules", entity)
		return status.Errorf(codes.Internal, "error creating profile: %s", err)
	}
	_, err = qtx.UpsertProfileForEntity(ctx, db.UpsertProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		zerolog.Ctx(ctx).Info().Err(err).Msgf("error updating profile for entity %s", entity)
		return err
	}

	return err
}

func upsertRuleInstances(
	ctx context.Context,
	qtx db.Querier,
	profileID uuid.UUID,
	projectID uuid.UUID,
	newRules []*minderv1.Profile_Rule,
	entityType db.Entities,
	rulesInProf RuleMapping,
) ([]uuid.UUID, error) {
	updatedIDs := make([]uuid.UUID, len(newRules))
	for i, rule := range newRules {
		// TODO: Clean up this logic once we no longer have to support the old tables.
		ruleIDAndName, ok := rulesInProf[RuleTypeAndNamePair{
			RuleType: rule.Type,
			RuleName: rule.Name,
		}]
		if !ok {
			return nil, fmt.Errorf("unable to find rule type ID for %s/%s", rule.Name, rule.Type)
		}

		def, err := json.Marshal(rule.Def)
		if err != nil {
			return nil, fmt.Errorf("unable to serialize rule def: %w", err)
		}

		params, err := json.Marshal(rule.Params)
		if err != nil {
			return nil, fmt.Errorf("unable to serialize rule params: %w", err)
		}

		id, err := qtx.UpsertRuleInstance(ctx, db.UpsertRuleInstanceParams{
			ProfileID: profileID,
			// TODO: Make non nullable in future PR
			ProjectID:  projectID,
			RuleTypeID: ruleIDAndName.RuleID,
			Name:       rule.Name,
			EntityType: entityType,
			Def:        def,
			Params:     params,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to insert new rule instance: %w", err)
		}

		updatedIDs[i] = id
	}

	return updatedIDs, nil
}

func (p *profileService) createSelectors(
	ctx context.Context,
	profID uuid.UUID,
	qtx db.Querier,
	selection []*minderv1.Profile_Selector,
) error {
	if err := p.validator.ValidateSelection(selection); err != nil {
		return err
	}

	return createSelectorDbRecords(ctx, profID, qtx, selection)
}

func (p *profileService) updateSelectors(
	ctx context.Context,
	profID uuid.UUID,
	qtx db.Querier,
	selection []*minderv1.Profile_Selector,
) error {
	if err := p.validator.ValidateSelection(selection); err != nil {
		return err
	}

	err := qtx.DeleteSelectorsByProfileID(ctx, profID)
	if err != nil {
		return fmt.Errorf("error deleting selectors: %w", err)
	}

	return createSelectorDbRecords(ctx, profID, qtx, selection)
}

func createSelectorDbRecords(
	ctx context.Context,
	profID uuid.UUID,
	qtx db.Querier,
	selection []*minderv1.Profile_Selector,
) error {
	for _, sel := range selection {
		dbEnt := db.NullEntities{}
		if minderv1.EntityFromString(sel.GetEntity()) != minderv1.Entity_ENTITY_UNSPECIFIED {
			dbEnt.Entities = entities.EntityTypeToDB(minderv1.EntityFromString(sel.GetEntity()))
			dbEnt.Valid = true
		}
		_, err := qtx.CreateSelector(ctx, db.CreateSelectorParams{
			ProfileID: profID,
			Entity:    dbEnt,
			Selector:  sel.GetSelector(),
			Comment:   sel.GetDescription(),
		})
		if err != nil {
			return fmt.Errorf("error creating selector: %w", err)
		}
	}

	return nil
}
