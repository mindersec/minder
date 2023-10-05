package db

import (
	"context"
	"encoding/json"
	"github.com/stacklok/mediator/internal/util/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func createRandomProfile(t *testing.T, provName string, projectID uuid.UUID) Profile {
	t.Helper()

	seed := time.Now().UnixNano()

	arg := CreateProfileParams{
		Name:      rand.RandomName(seed),
		Provider:  provName,
		ProjectID: projectID,
		Remediate: NullActionType{
			ActionType: "on",
			Valid:      true,
		},
	}

	prof, err := testQueries.CreateProfile(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, prof)

	return prof
}

func createRandomRuleType(t *testing.T, provName string, projectID uuid.UUID) RuleType {
	t.Helper()

	seed := time.Now().UnixNano()

	arg := CreateRuleTypeParams{
		Name:        rand.RandomName(seed),
		Provider:    provName,
		ProjectID:   projectID,
		Description: rand.RandomString(64, seed),
		Guidance:    rand.RandomString(64, seed),
		Definition:  json.RawMessage(`{"key": "value"}`),
	}

	ruleType, err := testQueries.CreateRuleType(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, ruleType)

	return ruleType
}

func profileIDStatusByIdAndProject(t *testing.T, profileID uuid.UUID, projectID uuid.UUID) GetProfileStatusByIdAndProjectRow {
	t.Helper()

	profileStatus, err := testQueries.GetProfileStatusByIdAndProject(context.Background(), GetProfileStatusByIdAndProjectParams{
		ID:        profileID,
		ProjectID: projectID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, profileStatus)

	return profileStatus
}

func upsertEvalStatus(t *testing.T, profileID uuid.UUID, repoID uuid.UUID, ruleTypeID uuid.UUID, entity Entities, evalStatus EvalStatusTypes, remediationStatus RemediationStatusTypes) {
	t.Helper()

	var id uuid.UUID
	var err error
	id, err = testQueries.InsertRuleEvaluations(context.Background(), InsertRuleEvaluationsParams{
		ProfileID: profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  repoID,
			Valid: true,
		},
		RuleTypeID: ruleTypeID,
		Entity:     entity,
	})
	if err != nil {
		id, err = testQueries.GetRuleEvaluationID(context.Background(), profileID)
	}
	require.NoError(t, err)
	require.NotNil(t, id)

	_, err = testQueries.UpsertRuleDetailsEval(context.Background(), UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     evalStatus,
		Details:    "",
	})
	require.NoError(t, err)
}

func TestCreateProfileStatusStoredProcedure(t *testing.T) {
	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	repo := createRandomRepository(t, proj.ID, prov.Name)
	ruleType1 := createRandomRuleType(t, prov.Name, proj.ID)
	ruleType2 := createRandomRuleType(t, prov.Name, proj.ID)

	profile := createRandomProfile(t, prov.Name, proj.ID)
	require.NotEmpty(t, profile)

	prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesPending)

	upsertEvalStatus(t, profile.ID, repo.ID, ruleType1.ID, EntitiesRepository, EvalStatusTypesSuccess, RemediationStatusTypesSkipped)
	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesSuccess)

	evalStatusRows, err := testQueries.ListRuleEvaluationsByProfileId(context.Background(),
		ListRuleEvaluationsByProfileIdParams{
			ProfileID: profile.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, evalStatusRows, 1)
	require.True(t, evalStatusRows[0].EvalStatus.Valid)
	require.Equal(t, evalStatusRows[0].EvalStatus.EvalStatusTypes, EvalStatusTypesSuccess)
	require.Equal(t, evalStatusRows[0].RepoName, repo.RepoName)

	upsertEvalStatus(t, profile.ID, repo.ID, ruleType2.ID, EntitiesRepository, EvalStatusTypesSuccess, RemediationStatusTypesSkipped)
	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesSuccess)

	evalStatusRows, err = testQueries.ListRuleEvaluationsByProfileId(context.Background(),
		ListRuleEvaluationsByProfileIdParams{
			ProfileID: profile.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, evalStatusRows, 2)
	require.True(t, evalStatusRows[0].EvalStatus.Valid)
	require.Equal(t, evalStatusRows[0].EvalStatus.EvalStatusTypes, EvalStatusTypesSuccess)
	require.Equal(t, evalStatusRows[0].RepoName, repo.RepoName)
	require.True(t, evalStatusRows[1].EvalStatus.Valid)
	require.Equal(t, evalStatusRows[1].EvalStatus.EvalStatusTypes, EvalStatusTypesSuccess)
	require.Equal(t, evalStatusRows[1].RepoName, repo.RepoName)

	upsertEvalStatus(t, profile.ID, repo.ID, ruleType1.ID, EntitiesRepository, EvalStatusTypesFailure, RemediationStatusTypesSkipped)
	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesFailure)

	upsertEvalStatus(t, profile.ID, repo.ID, ruleType1.ID, EntitiesRepository, EvalStatusTypesSuccess, RemediationStatusTypesSkipped)
	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesSuccess)

	upsertEvalStatus(t, profile.ID, repo.ID, ruleType2.ID, EntitiesRepository, EvalStatusTypesError, RemediationStatusTypesSkipped)
	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, proj.ID)
	require.Equal(t, prfStatusRow.ProfileStatus, EvalStatusTypesError)
}
