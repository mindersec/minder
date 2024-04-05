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

package controlplane

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListEvaluationResults lists the evaluation results for entities filtered b
// entity type, labels, profiles, and rule types.
func (s *Server) ListEvaluationResults(
	ctx context.Context,
	in *minderv1.ListEvaluationResultsRequest,
) (*minderv1.ListEvaluationResultsResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if _, err := uuid.Parse(in.GetProfile()); err != nil && in.GetProfile() != "" {
		return nil, status.Error(codes.InvalidArgument, "Error invalid profile ID")
	}

	// Build indexes of the request parameters
	rtIndex, entIdIndex, entTypeIndex := indexRequestParams(in)

	// Build a list of all profiles
	profileList, err := buildProjectsProfileList(ctx, s.store, []uuid.UUID{projectID}, in.GetLabelFilter())
	if err != nil {
		return nil, err
	}

	// Build a list of the status of the profiles
	profileStatusList, err := buildProjectStatusList(ctx, s.store, []uuid.UUID{projectID})
	if err != nil {
		return nil, err
	}

	// Filter the profile and status lists to those in the reques params
	profileList, profileStatusList = filterProfileLists(in, profileList, profileStatusList)

	// Do the final sort of all the data
	entities, profileStatuses, statusByEntity, err := sortEntitiesEvaluationStatus(
		ctx, s.store, profileList, profileStatusList, rtIndex, entIdIndex, entTypeIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("sorting rule evaluations: %w", err)
	}

	return buildListEvaluationResponse(entities, profileStatuses, statusByEntity), nil
}

// sortEntitiesEvaluationStatus queries the database from the filtered lists
// and compiles the unified entities map, the map of profile statuses and the
// status by entity map that will be assembled into the evaluation status
// response.
func sortEntitiesEvaluationStatus(
	ctx context.Context, store db.Store,
	profileList []db.ListProfilesByProjectIDAndLabelRow,
	profileStatusList map[uuid.UUID]db.GetProfileStatusByProjectRow,
	rtIndex, entIdIndex, entTypeIndex map[string]struct{},
) (
	entities map[string]*minderv1.EntityTypedId,
	profileStatuses map[uuid.UUID]*minderv1.ProfileStatus,
	statusByEntity map[string]map[uuid.UUID][]*minderv1.RuleEvaluationStatus, err error,
) {
	entities = map[string]*minderv1.EntityTypedId{}
	profileStatuses = map[uuid.UUID]*minderv1.ProfileStatus{}
	statusByEntity = map[string]map[uuid.UUID][]*minderv1.RuleEvaluationStatus{}

	for _, p := range profileList {
		p := p
		evals, err := store.ListRuleEvaluationsByProfileId(
			ctx, db.ListRuleEvaluationsByProfileIdParams{ProfileID: p.Profile.ID},
		)
		if err != nil {
			return nil, nil, nil,
				status.Errorf(codes.Internal,
					"error reading evaluations from profile %q: %v", p.Profile.ID.String(), err)
		}

		for _, e := range evals {
			// Filter by rule type name
			if _, ok := rtIndex[e.RuleTypeName]; !ok && len(rtIndex) > 0 {
				continue
			}

			ent := buildEntityFromEvaluation(e)
			entString := fmt.Sprintf("%s/%s", ent.Type, ent.Id)

			/// If we're constrained to a single entity type, ignore others
			if _, ok := entTypeIndex[ent.Type.String()]; !ok && len(entTypeIndex) > 0 {
				continue
			}

			// Filter other entities if we have a list
			if _, ok := entIdIndex[entString]; !ok && len(entIdIndex) > 0 {
				continue
			}

			entities[entString] = ent

			if _, ok := profileStatuses[p.Profile.ID]; !ok {
				profileStatuses[p.Profile.ID] = buildProfileStatus(&p, profileStatusList)
			}

			stat, err := buildRuleEvaluationStatusFromDBEvaluation(ctx, &p, e)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("building rule evaluation status: %w", err)
			}
			if _, ok := statusByEntity[entString]; !ok {
				statusByEntity[entString] = make(map[uuid.UUID][]*minderv1.RuleEvaluationStatus)
			}
			statusByEntity[entString][p.Profile.ID] = append(statusByEntity[entString][p.Profile.ID], stat)
		}
	}
	return entities, profileStatuses, statusByEntity, err
}

// buildListEvaluationResponse builds the final response from the sorted and
// filetered list of evaluations, statuses and profiles
func buildListEvaluationResponse(
	entities map[string]*minderv1.EntityTypedId,
	profileStatuses map[uuid.UUID]*minderv1.ProfileStatus,
	statusByEntity map[string]map[uuid.UUID][]*minderv1.RuleEvaluationStatus,
) *minderv1.ListEvaluationResultsResponse {
	res := &minderv1.ListEvaluationResultsResponse{
		Entities: []*minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{},
	}

	for entString, ent := range entities {
		if _, ok := statusByEntity[entString]; !ok {
			continue
		}

		returnProfileEvals := &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
			Entity:   ent,
			Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{},
		}

		for profileId, results := range statusByEntity[entString] {
			res := &minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
				ProfileStatus: profileStatuses[profileId],
				Results:       results,
			}
			returnProfileEvals.Profiles = append(returnProfileEvals.Profiles, res)
		}

		res.Entities = append(res.Entities, returnProfileEvals)
	}

	return res
}

// indexRequestParams returns indexes of the request params to look them up.
// The functions returns indexes of the ruletype, entity ids and full entity.
func indexRequestParams(
	in *minderv1.ListEvaluationResultsRequest,
) (rtIndex, entIdIndex, entTypeIndex map[string]struct{}) {
	// These are the indexes
	rtIndex = map[string]struct{}{}
	entIdIndex = map[string]struct{}{}
	entTypeIndex = map[string]struct{}{}

	for _, rt := range in.GetRuleName() {
		if rt == "" {
			continue
		}
		rtIndex[rt] = struct{}{}
	}

	for _, ent := range in.GetEntity() {
		if ent.Id == "" {
			entTypeIndex[ent.Type.String()] = struct{}{}
		} else {
			entIdIndex[fmt.Sprintf("%s/%s", ent.Type, ent.Id)] = struct{}{}
		}
	}
	return rtIndex, entIdIndex, entTypeIndex
}

// buildProjectStatusList returns a list if project statuses keyed by project ID
func buildProjectStatusList(
	ctx context.Context, store db.Store, projects []uuid.UUID,
) (map[uuid.UUID]db.GetProfileStatusByProjectRow, error) {
	profileStatusList := map[uuid.UUID]db.GetProfileStatusByProjectRow{}
	for _, projectID := range projects {
		sl, err := store.GetProfileStatusByProject(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error reading profile status: %v", err)
		}

		for _, srow := range sl {
			profileStatusList[srow.ID] = srow
		}
	}
	return profileStatusList, nil
}

// buildProjectsProfileList takes a list of projects and returns a list if profiles
func buildProjectsProfileList(
	ctx context.Context, store db.Store, projects []uuid.UUID, filter string,
) ([]db.ListProfilesByProjectIDAndLabelRow, error) {
	profileList := []db.ListProfilesByProjectIDAndLabelRow{}

	listParams := db.ListProfilesByProjectIDAndLabelParams{}
	listParams.LabelsFromFilter(filter)

	for _, projectID := range projects {
		listParams.ProjectID = projectID

		zerolog.Ctx(ctx).Debug().Interface("listParams", listParams).Msg("profile list parameters")

		profiles, err := store.ListProfilesByProjectIDAndLabel(ctx, listParams)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error listing profiles")
		}

		profileList = append(profileList, profiles...)
	}
	return profileList, nil
}

func filterProfileLists(
	in *minderv1.ListEvaluationResultsRequest,
	inProfileList []db.ListProfilesByProjectIDAndLabelRow,
	inProfileStatus map[uuid.UUID]db.GetProfileStatusByProjectRow,
) ([]db.ListProfilesByProjectIDAndLabelRow, map[uuid.UUID]db.GetProfileStatusByProjectRow) {
	if in.GetProfile() == "" {
		return inProfileList, inProfileStatus
	}

	outProfileList := []db.ListProfilesByProjectIDAndLabelRow{}
	outProfileStatus := map[uuid.UUID]db.GetProfileStatusByProjectRow{}

	for _, p := range inProfileList {
		if p.Profile.ID.String() == in.GetProfile() {
			outProfileList = append(outProfileList, p)
		}
	}

	if v, ok := inProfileStatus[uuid.MustParse(in.GetProfile())]; ok {
		outProfileStatus[uuid.MustParse(in.GetProfile())] = v
	}

	return outProfileList, outProfileStatus
}

// buildRuleEvaluationStatusFromDBEvaluation converts from an evaluation and a
// profile from the database to a minder RuleEvaluationStatus
func buildRuleEvaluationStatusFromDBEvaluation(
	ctx context.Context,
	profile *db.ListProfilesByProjectIDAndLabelRow, eval db.ListRuleEvaluationsByProfileIdRow,
) (*minderv1.RuleEvaluationStatus, error) {
	guidance := ""
	// Only return the rule type guidance text when there is a problem
	if eval.EvalStatus.EvalStatusTypes == db.EvalStatusTypesFailure ||
		eval.EvalStatus.EvalStatusTypes == db.EvalStatusTypesError {
		guidance = eval.RuleTypeGuidance
	}

	sev := &minderv1.Severity{}
	sev.EnsureDefault()
	if err := sev.Value.FromString(string(eval.RuleTypeSeverityValue)); err != nil {
		zerolog.Ctx(ctx).
			Err(err).
			Str("value", string(eval.RuleTypeSeverityValue)).
			Msg("error converting severity will use defaults")
	}

	entityInfo := map[string]string{}
	remediationURL := ""
	var err error
	if eval.Entity == db.EntitiesRepository {
		entityInfo["provider"] = eval.Provider
		entityInfo["repo_owner"] = eval.RepoOwner
		entityInfo["repo_name"] = eval.RepoName
		entityInfo["repository_id"] = eval.RepositoryID.UUID.String()

		remediationURL, err = getRemediationURLFromMetadata(
			eval.RemMetadata.RawMessage, fmt.Sprintf("%s/%s", eval.RepoOwner, eval.RepoName),
		)
		if err != nil {
			return nil, fmt.Errorf("parsing remediation pull request data: %w", err)
		}
	}

	// Default to the rule type name if no display_name is set
	nString := eval.RuleTypeDisplayName
	if nString == "" {
		nString = eval.RuleTypeName
	}

	return &minderv1.RuleEvaluationStatus{
		RuleEvaluationId:       eval.RuleEvaluationID.String(),
		RuleId:                 eval.RuleTypeID.String(),
		ProfileId:              profile.Profile.ID.String(),
		RuleName:               eval.RuleName,
		Entity:                 string(eval.Entity),
		Status:                 string(eval.EvalStatus.EvalStatusTypes),
		LastUpdated:            timestamppb.New(eval.EvalLastUpdated.Time),
		EntityInfo:             entityInfo,
		Details:                eval.EvalDetails.String,
		Guidance:               guidance,
		RemediationStatus:      string(eval.RemStatus.RemediationStatusTypes),
		RemediationLastUpdated: timestamppb.New(eval.RemLastUpdated.Time),
		RemediationDetails:     eval.RemDetails.String,
		RemediationUrl:         remediationURL,
		RuleDisplayName:        nString,
		RuleTypeName:           eval.RuleTypeName,
		Alert:                  buildEvalResultAlertFromLRERow(&eval),
		Severity:               sev,
	}, nil
}

func buildEntityFromEvaluation(eval db.ListRuleEvaluationsByProfileIdRow) *minderv1.EntityTypedId {
	ent := &minderv1.EntityTypedId{
		Type: dbEntityToEntity(eval.Entity),
	}

	if ent.Type == minderv1.Entity_ENTITY_REPOSITORIES && eval.RepoOwner != "" && eval.RepoName != "" {
		ent.Id = eval.RepositoryID.UUID.String()
	}
	return ent
}

// buildProfileStatus build a minderv1.ProfileStatus struct from a lookup row
func buildProfileStatus(
	row *db.ListProfilesByProjectIDAndLabelRow,
	profileStatusList map[uuid.UUID]db.GetProfileStatusByProjectRow,
) *minderv1.ProfileStatus {
	pfStatus := ""
	if _, ok := profileStatusList[row.Profile.ID]; ok {
		pfStatus = string(profileStatusList[row.Profile.ID].ProfileStatus)
	}

	displayName := row.Profile.DisplayName
	if displayName == "" {
		displayName = row.Profile.Name
	}

	return &minderv1.ProfileStatus{
		ProfileId:          row.Profile.ID.String(),
		ProfileName:        row.Profile.Name,
		ProfileDisplayName: displayName,
		ProfileStatus:      pfStatus,
		LastUpdated:        timestamppb.New(row.Profile.UpdatedAt),
	}
}

// buildEvalResultAlertFromLRERow build the evaluation result alert from a
// database row.
func buildEvalResultAlertFromLRERow(eval *db.ListRuleEvaluationsByProfileIdRow) *minderv1.EvalResultAlert {
	era := &minderv1.EvalResultAlert{
		Status:      string(eval.AlertStatus.AlertStatusTypes),
		LastUpdated: timestamppb.New(eval.AlertLastUpdated.Time),
		Details:     eval.AlertDetails.String,
	}

	if eval.AlertMetadata.Valid && eval.AlertStatus.AlertStatusTypes == db.AlertStatusTypesOn {
		repoSlug := ""
		if eval.RepoOwner != "" && eval.RepoName != "" {
			repoSlug = fmt.Sprintf("%s/%s", eval.RepoOwner, eval.RepoName)
		}
		urlString, err := getAlertURLFromMetadata(
			eval.AlertMetadata.RawMessage, repoSlug,
		)
		if err == nil {
			era.Url = urlString
		}
	}

	return era
}

func dbEntityToEntity(dbEnt db.Entities) minderv1.Entity {
	switch dbEnt {
	case db.EntitiesPullRequest:
		return minderv1.Entity_ENTITY_PULL_REQUESTS
	case db.EntitiesArtifact:
		return minderv1.Entity_ENTITY_ARTIFACTS
	case db.EntitiesRepository:
		return minderv1.Entity_ENTITY_REPOSITORIES
	case db.EntitiesBuildEnvironment:
		return minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS
	default:
		return minderv1.Entity_ENTITY_UNSPECIFIED
	}
}
