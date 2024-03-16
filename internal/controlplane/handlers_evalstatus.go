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
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListEvaluationResults lists the evaluation results for entities filtered b
// entity type, labels, profiles, and rule types.
func (s *Server) ListEvaluationResults(
	ctx context.Context,
	in *minderv1.ListEvaluationResultsRequest,
) (*minderv1.ListEvaluationResultsResponse, error) {
	if _, err := uuid.Parse(in.GetProfile()); err != nil && in.GetProfile() != "" {
		return nil, status.Error(codes.InvalidArgument, "Error invalid profile ID")
	}

	projectID, err := uuid.Parse(*in.GetContext().Project)
	if err != nil {
		return nil, fmt.Errorf("error parsing project identifier")
	}

	// Build indexes of the request parameters
	rtIndex, entIdIndex, entTypeIndex := indexRequestParams(in)

	// Build a list of all profiles
	profileList, err := buildProjectsProfileList(ctx, s.store, []uuid.UUID{projectID})
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
	profileList []db.ListProfilesByProjectIDRow,
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
			ctx, db.ListRuleEvaluationsByProfileIdParams{ProfileID: p.ID},
		)
		if err != nil {
			return nil, nil, nil, status.Errorf(codes.Internal, "error reading evaluations from profile %q: %v", p.ID.String(), err)
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

			if _, ok := profileStatuses[p.ID]; !ok {
				profileStatuses[p.ID] = buildProfileStatus(&p, profileStatusList)
			}

			stat := buildRuleEvaluationStatusFromDBEvaluation(&p, e)
			if _, ok := statusByEntity[entString]; !ok {
				statusByEntity[entString] = make(map[uuid.UUID][]*minderv1.RuleEvaluationStatus)
			}
			statusByEntity[entString][p.ID] = append(statusByEntity[entString][p.ID], stat)
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
	ctx context.Context, store db.Store, projects []uuid.UUID,
) ([]db.ListProfilesByProjectIDRow, error) {
	profileList := []db.ListProfilesByProjectIDRow{}

	for _, projectID := range projects {
		profiles, err := store.ListProfilesByProjectID(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error listing profiles")
		}

		profileList = append(profileList, profiles...)
	}
	return profileList, nil
}

func filterProfileLists(
	in *minderv1.ListEvaluationResultsRequest,
	inProfileList []db.ListProfilesByProjectIDRow,
	inProfileStatus map[uuid.UUID]db.GetProfileStatusByProjectRow,
) ([]db.ListProfilesByProjectIDRow, map[uuid.UUID]db.GetProfileStatusByProjectRow) {
	if in.GetProfile() == "" {
		return inProfileList, inProfileStatus
	}

	outProfileList := []db.ListProfilesByProjectIDRow{}
	outProfileStatus := map[uuid.UUID]db.GetProfileStatusByProjectRow{}

	for _, p := range inProfileList {
		if p.ID.String() == in.GetProfile() {
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
	profile *db.ListProfilesByProjectIDRow, eval db.ListRuleEvaluationsByProfileIdRow,
) *minderv1.RuleEvaluationStatus {
	guidance := ""
	// Only return the rule type guidance text when there is a problem
	if eval.EvalStatus.EvalStatusTypes == db.EvalStatusTypesFailure ||
		eval.EvalStatus.EvalStatusTypes == db.EvalStatusTypesError {
		guidance = eval.RuleTypeGuidance
	}

	return &minderv1.RuleEvaluationStatus{
		ProfileId:              profile.ID.String(),
		RuleName:               eval.RuleName,
		Entity:                 string(eval.Entity),
		Status:                 string(eval.EvalStatus.EvalStatusTypes),
		LastUpdated:            timestamppb.New(eval.EvalLastUpdated.Time),
		EntityInfo:             map[string]string{},
		Details:                eval.EvalDetails.String,
		Guidance:               guidance,
		RemediationStatus:      string(eval.RemStatus.RemediationStatusTypes),
		RemediationLastUpdated: timestamppb.New(eval.RemLastUpdated.Time),
		RemediationDetails:     eval.RemDetails.String,
		RuleTypeName:           eval.RuleTypeName,
		Alert:                  buildEvalResultAlertFrom(&eval),
		Severity:               &minderv1.Severity{},
	}
}

func buildEntityFromEvaluation(eval db.ListRuleEvaluationsByProfileIdRow) *minderv1.EntityTypedId {
	ent := &minderv1.EntityTypedId{
		Type: dbEntityToEntity(eval.Entity),
	}

	if ent.Type == minderv1.Entity_ENTITY_REPOSITORIES && eval.RepoOwner != "" && eval.RepoName != "" {
		ent.Id = fmt.Sprintf("%s/%s", eval.RepoOwner, eval.RepoName)
	}
	return ent
}

// buildProfileStatus build a minderv1.ProfileStatus struct from a lookup row
func buildProfileStatus(
	row *db.ListProfilesByProjectIDRow,
	profileStatusList map[uuid.UUID]db.GetProfileStatusByProjectRow,
) *minderv1.ProfileStatus {
	pfStatus := ""
	if _, ok := profileStatusList[row.ID]; ok {
		pfStatus = string(profileStatusList[row.ID].ProfileStatus)
	}

	return &minderv1.ProfileStatus{
		ProfileId:     row.ID.String(),
		ProfileName:   row.Name,
		ProfileStatus: pfStatus,
		LastUpdated:   timestamppb.New(row.UpdatedAt),
	}
}

// buildEvalResultAlertFrom
func buildEvalResultAlertFrom(eval *db.ListRuleEvaluationsByProfileIdRow) *minderv1.EvalResultAlert {
	return &minderv1.EvalResultAlert{
		Status:      string(eval.AlertStatus.AlertStatusTypes),
		LastUpdated: timestamppb.New(eval.AlertLastUpdated.Time),
		Details:     eval.AlertDetails.String,
	}
}

func mockListEvaluationResults() *minderv1.ListEvaluationResultsResponse {
	res := []*minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{}
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/depot",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				ProfileStatus: &minderv1.ProfileStatus{
					ProfileId:     "a4d5589b-cf5a-42ab-a940-15d2a8c5a3e1",
					ProfileName:   "my-profile",
					LastUpdated:   &timestamppb.Timestamp{Seconds: time.Now().Unix()},
					ProfileStatus: "failure",
				},
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "a4d5589b-cf5a-42ab-a940-15d2a8c5a3e1",
						RuleId:      "12d4c5e8-7a6e-4a50-afba-8883d9a45673",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "depot",
							"repo_owner":    "stacklok",
							"repository_id": "a2b6392d-3b4f-48a9-b641-dc9c595c3daf",
						},
						Details:                "Security alerts are enabled for the repository.",
						Guidance:               "Keep the security alerts feature enabled to stay informed about potential vulnerabilities.",
						RemediationStatus:      "not available",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						RuleTypeName:           "branch_protection_lock_branch",
						RuleDescriptionName:    "branch_protection_lock_branch",
						Alert: &minderv1.EvalResultAlert{
							Status:      "on",
							LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
							Details:     "",
							Url:         "https://github.com/stacklok/depot/security/advisories/GHSA-cxf2-4pf8-4rgc",
						},
						Severity: &minderv1.Severity{Value: minderv1.Severity_VALUE_CRITICAL},
					},
					{
						ProfileId:   "a4d5589b-cf5a-42ab-a940-15d2a8c5a3e1",
						RuleId:      "12d4c5e8-7a6e-4a50-afba-8883d9a45673",
						Entity:      "repository",
						Status:      "success",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "depot",
							"repo_owner":    "stacklok",
							"repository_id": "a2b6392d-3b4f-48a9-b641-dc9c595c3daf",
						},
						Details:                "Security alerts are enabled for the repository.",
						Guidance:               "Keep the security alerts feature enabled to stay informed about potential vulnerabilities.",
						RemediationStatus:      "not available",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						RuleTypeName:           "branch_protection_lock_branch",
						RuleDescriptionName:    "branch_protection_lock_branch",
						Alert: &minderv1.EvalResultAlert{
							Status:      "off",
							LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
							Details:     "",
						},
						Severity: &minderv1.Severity{Value: minderv1.Severity_VALUE_LOW},
					},
				},
			},
		},
	},
	)
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/insight",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				ProfileStatus: &minderv1.ProfileStatus{
					ProfileId:     "ebd3f978-f2b0-4cd9-a9de-2536a2414a37",
					ProfileName:   "my-profile-2",
					LastUpdated:   &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
					ProfileStatus: "failure",
				},
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "ebd3f978-f2b0-4cd9-a9de-2536a2414a37",
						RuleId:      "9984ecef-b8b9-41d4-8d02-748889d3aef2",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "insight",
							"repo_owner":    "stacklok",
							"repository_id": "9d3ef47a-3d6f-43f8-9a29-d9112fbc92f3",
						},
						Details:                "denied",
						Guidance:               "Configure branch protection rules to require code reviews before merging.",
						RemediationStatus:      "pending",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 200},
						RuleTypeName:           "artifact_signature",
						RuleDescriptionName:    "artifact_signature",
						Alert: &minderv1.EvalResultAlert{
							Status:      "off",
							LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 180},
							Details:     "",
						},
						Severity: &minderv1.Severity{Value: minderv1.Severity_VALUE_CRITICAL},
					},
				},
			},
		},
	},
	)
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/streamline",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				ProfileStatus: &minderv1.ProfileStatus{
					ProfileId:     "c5bd9d1b-3d96-4a57-a5a0-3d6248c3b677",
					ProfileName:   "my-profile-3",
					LastUpdated:   &timestamppb.Timestamp{Seconds: time.Now().Unix() - 30},
					ProfileStatus: "error",
				},
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "c5bd9d1b-3d96-4a57-a5a0-3d6248c3b677",
						RuleId:      "2aaf7f8c-76c2-499e-b667-9a4d28b3c295",
						Entity:      "repository",
						Status:      "error",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 30},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "streamline",
							"repo_owner":    "stacklok",
							"repository_id": "b3229c1e-6f2c-4b4b-9f58-5b690f2af6f9",
						},
						Details:                "evaluation skipped: rule not applicable",
						Guidance:               "Update dependencies to their latest versions to improve security and compatibility.",
						RemediationStatus:      "skipped",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 150},
						RuleTypeName:           "dependabot_configured",
						RuleDescriptionName:    "dependabot_configured",
						Alert: &minderv1.EvalResultAlert{
							Status:      "error",
							LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
							Details:     "",
						},
						Severity: &minderv1.Severity{Value: minderv1.Severity_VALUE_HIGH},
					},
					{
						ProfileId:   "c5bd9d1b-3d96-4a57-a5a0-3d6248c3b677",
						RuleId:      "2aaf7f8c-76c2-499e-b667-9a4d28b3c295",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 30},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "streamline",
							"repo_owner":    "stacklok",
							"repository_id": "b3229c1e-6f2c-4b4b-9f58-5b690f2af6f9",
						},
						Details:                "denied",
						Guidance:               "Update dependencies to their latest versions to improve security and compatibility.",
						RemediationStatus:      "skipped",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 150},
						RuleTypeName:           "automatic_branch_deletion",
						RuleDescriptionName:    "automatic_branch_deletion",
						Alert: &minderv1.EvalResultAlert{
							Status:      "on",
							LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
							Details:     "",
							Url:         "https://github.com/stacklok/streamline/security/advisories/GHSA-cxf2-4pf8-4rgc",
						},
						Severity: &minderv1.Severity{Value: minderv1.Severity_VALUE_MEDIUM},
					},
				},
			},
		},
	},
	)
	return &minderv1.ListEvaluationResultsResponse{
		Entities: res,
	}
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
