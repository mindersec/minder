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
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/engcontext"
	"github.com/stacklok/minder/internal/flags"
	"github.com/stacklok/minder/internal/history"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	defaultPageSize uint64 = 25
)

// ListEvaluationHistory lists current and past evaluation results for
// entities.
func (s *Server) ListEvaluationHistory(
	ctx context.Context,
	in *minderv1.ListEvaluationHistoryRequest,
) (*minderv1.ListEvaluationHistoryResponse, error) {
	if !flags.Bool(ctx, s.featureFlags, flags.EvalHistory) {
		return nil, status.Error(codes.Unimplemented, "Not implemented")
	}

	// process cursor
	cursor := &history.DefaultCursor
	size := defaultPageSize
	if in.GetCursor() != nil {
		parsedCursor, err := history.ParseListEvaluationCursor(
			in.GetCursor().GetCursor(),
		)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid cursor")
		}
		cursor = parsedCursor
		size = in.GetCursor().GetSize()
	}

	// process filter
	opts := []history.FilterOpt{}
	opts = append(opts, FilterOptsFromStrings(in.GetEntityType(), history.WithEntityType)...)
	opts = append(opts, FilterOptsFromStrings(in.GetEntityName(), history.WithEntityName)...)
	opts = append(opts, FilterOptsFromStrings(in.GetProfileName(), history.WithProfileName)...)
	opts = append(opts, FilterOptsFromStrings(in.GetStatus(), history.WithStatus)...)
	opts = append(opts, FilterOptsFromStrings(in.GetRemediation(), history.WithRemediation)...)
	opts = append(opts, FilterOptsFromStrings(in.GetAlert(), history.WithAlert)...)

	if in.GetFrom() != nil {
		opts = append(opts, history.WithFrom(in.GetFrom().AsTime()))
	}
	if in.GetTo() != nil {
		opts = append(opts, history.WithTo(in.GetTo().AsTime()))
	}

	// we always filter by project id
	opts = append(opts, history.WithProjectID(GetProjectID(ctx)))

	filter, err := history.NewListEvaluationFilter(opts...)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid filter")
	}

	// retrieve data set
	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer s.store.Rollback(tx)

	result, err := s.history.ListEvaluationHistory(
		ctx,
		s.store.GetQuerierWithTransaction(tx),
		cursor,
		size,
		filter,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "error retrieving evaluations")
	}

	// convert data set to proto
	data, err := fromEvaluationHistoryRow(result.Data)
	if err != nil {
		return nil, err
	}

	// return data set to client
	resp := &minderv1.ListEvaluationHistoryResponse{}
	if len(data) == 0 {
		return resp, nil
	}

	resp.Data = data
	resp.Page = &minderv1.CursorPage{}

	if result.Next != nil {
		resp.Page.Next = makeCursor(result.Next, size)
	}
	if result.Prev != nil {
		resp.Page.Prev = makeCursor(result.Prev, size)
	}

	return resp, nil
}

func fromEvaluationHistoryRow(
	rows []db.ListEvaluationHistoryRow,
) ([]*minderv1.EvaluationHistory, error) {
	res := []*minderv1.EvaluationHistory{}

	for _, row := range rows {
		var dbEntityType db.Entities
		if err := dbEntityType.Scan(row.EntityType); err != nil {
			return nil, errors.New("internal error")
		}
		entityType := dbEntityToEntity(dbEntityType)
		entityName, err := getEntityName(dbEntityType, row)
		if err != nil {
			return nil, err
		}

		var alert *minderv1.EvaluationHistoryAlert
		if row.AlertStatus.Valid {
			alert = &minderv1.EvaluationHistoryAlert{
				Status:  string(row.AlertStatus.AlertStatusTypes),
				Details: row.AlertDetails.String,
			}
		}
		var remediation *minderv1.EvaluationHistoryRemediation
		if row.RemediationStatus.Valid {
			remediation = &minderv1.EvaluationHistoryRemediation{
				Status:  string(row.RemediationStatus.RemediationStatusTypes),
				Details: row.RemediationDetails.String,
			}
		}

		res = append(res, &minderv1.EvaluationHistory{
			EvaluatedAt: timestamppb.New(row.EvaluatedAt),
			Entity: &minderv1.EvaluationHistoryEntity{
				Id:   row.EvaluationID.String(),
				Type: entityType,
				Name: entityName,
			},
			Rule: &minderv1.EvaluationHistoryRule{
				Name:     row.RuleName,
				RuleType: row.RuleType,
				Profile:  row.ProfileName,
			},
			Status: &minderv1.EvaluationHistoryStatus{
				Status:  string(row.EvaluationStatus),
				Details: row.EvaluationDetails,
			},
			Alert:       alert,
			Remediation: remediation,
		})
	}

	return res, nil
}

func makeCursor(cursor []byte, size uint64) *minderv1.Cursor {
	return &minderv1.Cursor{
		Cursor: base64.StdEncoding.EncodeToString(cursor),
		Size:   size,
	}
}

// FilterOptsFromStrings calls the given function `f` on each element
// of values. Such elements are either "complex", i.e. they represent
// a comma-separated list of sub-elements, or "simple", they do not
// contain comma characters. If element contains one or more comma
// characters, it is further split into sub-elements before calling
// `f` in them.
func FilterOptsFromStrings(
	values []string,
	f func(string) history.FilterOpt,
) []history.FilterOpt {
	opts := []history.FilterOpt{}
	for _, val := range values {
		for _, part := range strings.Split(val, ",") {
			opts = append(opts, f(part))
		}
	}
	return opts
}

// ListEvaluationResults lists the latest evaluation results for
// entities filtered by entity type, labels, profiles, and rule types.
func (s *Server) ListEvaluationResults(
	ctx context.Context,
	in *minderv1.ListEvaluationResultsRequest,
) (*minderv1.ListEvaluationResultsResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
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
				// A failure parsing the PR metadata points to a corrupt record. Log but don't err.
				zerolog.Ctx(ctx).Error().Err(err).Msg("error building rule evaluation status")
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

func getEntityName(
	dbEnt db.Entities,
	row db.ListEvaluationHistoryRow,
) (string, error) {
	switch dbEnt {
	case db.EntitiesPullRequest:
		if !row.RepoOwner.Valid {
			return "", errors.New("repo_owner is missing")
		}
		if !row.RepoName.Valid {
			return "", errors.New("repo_name is missing")
		}
		if !row.PrNumber.Valid {
			return "", errors.New("pr_number is missing")
		}
		return fmt.Sprintf("%s/%s#%d",
			row.RepoOwner.String,
			row.RepoName.String,
			row.PrNumber.Int64,
		), nil
	case db.EntitiesArtifact:
		if !row.ArtifactName.Valid {
			return "", errors.New("artifact_name is missing")
		}
		return row.ArtifactName.String, nil
	case db.EntitiesRepository:
		if !row.RepoOwner.Valid {
			return "", errors.New("repo_owner is missing")
		}
		if !row.RepoName.Valid {
			return "", errors.New("repo_name is missing")
		}
		return fmt.Sprintf("%s/%s",
			row.RepoOwner.String,
			row.RepoName.String,
		), nil
	case db.EntitiesBuildEnvironment:
		return "", errors.New("invalid entity type")
	default:
		return "", errors.New("invalid entity type")
	}
}
