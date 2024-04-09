package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

func createRandomProfile(t *testing.T, projectID uuid.UUID, labels []string) Profile {
	t.Helper()

	seed := time.Now().UnixNano()

	arg := CreateProfileParams{
		Name:      rand.RandomName(seed),
		ProjectID: projectID,
		Remediate: NullActionType{
			ActionType: "on",
			Valid:      true,
		},
		Alert: NullActionType{
			ActionType: "on",
			Valid:      true,
		},
		Labels: labels,
	}

	prof, err := testQueries.CreateProfile(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, prof)

	// listProfilesLabels doesn't join properly without the entityProfile entries
	ent, err := testQueries.CreateProfileForEntity(context.Background(), CreateProfileForEntityParams{
		ProfileID:       prof.ID,
		Entity:          EntitiesRepository,
		ContextualRules: json.RawMessage(`{"key": "value"}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, ent)

	return prof
}

func createRandomRuleType(t *testing.T, projectID uuid.UUID) RuleType {
	t.Helper()

	seed := time.Now().UnixNano()

	arg := CreateRuleTypeParams{
		Name:          rand.RandomName(seed),
		ProjectID:     projectID,
		Description:   rand.RandomString(64, seed),
		Guidance:      rand.RandomString(64, seed),
		Definition:    json.RawMessage(`{"key": "value"}`),
		SeverityValue: SeverityHigh,
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

func upsertEvalStatus(
	t *testing.T, profileID uuid.UUID, repoID uuid.UUID, ruleTypeID uuid.UUID,
	evalStatus EvalStatusTypes, details string,
) {
	t.Helper()

	id, err := testQueries.UpsertRuleEvaluations(context.Background(), UpsertRuleEvaluationsParams{
		ProfileID: profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  repoID,
			Valid: true,
		},
		RuleTypeID: ruleTypeID,
		Entity:     EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotNil(t, id)

	_, err = testQueries.UpsertRuleDetailsEval(context.Background(), UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     evalStatus,
		Details:    details,
	})
	require.NoError(t, err)
}

func upsertRemediationStatus(
	t *testing.T, profileID uuid.UUID, repoID uuid.UUID, ruleTypeID uuid.UUID,
	remStatus RemediationStatusTypes, details string, metadata json.RawMessage,
) {
	t.Helper()

	id, err := testQueries.UpsertRuleEvaluations(context.Background(), UpsertRuleEvaluationsParams{
		ProfileID: profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  repoID,
			Valid: true,
		},
		RuleTypeID: ruleTypeID,
		Entity:     EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotNil(t, id)

	_, err = testQueries.UpsertRuleDetailsRemediate(context.Background(), UpsertRuleDetailsRemediateParams{
		RuleEvalID: id,
		Status:     remStatus,
		Details:    details,
		Metadata:   metadata,
	})
	require.NoError(t, err)
}

func upsertAlertStatus(
	t *testing.T, profileID uuid.UUID, repoID uuid.UUID, ruleTypeID uuid.UUID,
	alertStatus AlertStatusTypes, details string, metadata json.RawMessage,
) {
	t.Helper()

	id, err := testQueries.UpsertRuleEvaluations(context.Background(), UpsertRuleEvaluationsParams{
		ProfileID: profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  repoID,
			Valid: true,
		},
		RuleTypeID: ruleTypeID,
		Entity:     EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotNil(t, id)

	_, err = testQueries.UpsertRuleDetailsAlert(context.Background(), UpsertRuleDetailsAlertParams{
		RuleEvalID: id,
		Status:     alertStatus,
		Metadata:   metadata,
		Details:    details,
	})
	require.NoError(t, err)
}

type testRandomEntities struct {
	prov Provider
	proj Project
	repo Repository

	ruleType1 RuleType
	ruleType2 RuleType
}

func createTestRandomEntities(t *testing.T) *testRandomEntities {
	t.Helper()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	repo := createRandomRepository(t, proj.ID, prov)
	ruleType1 := createRandomRuleType(t, proj.ID)
	ruleType2 := createRandomRuleType(t, proj.ID)

	return &testRandomEntities{
		prov:      prov,
		proj:      proj,
		repo:      repo,
		ruleType1: ruleType1,
		ruleType2: ruleType2,
	}
}

func TestProfileLabels(t *testing.T) {
	t.Parallel()

	randomEntities := createTestRandomEntities(t)

	health1 := createRandomProfile(t, randomEntities.proj.ID, []string{"stacklok:health"})
	require.NotEmpty(t, health1)
	health2 := createRandomProfile(t, randomEntities.proj.ID, []string{"stacklok:health", "obsolete"})
	require.NotEmpty(t, health2)
	obsolete := createRandomProfile(t, randomEntities.proj.ID, []string{"obsolete"})
	require.NotEmpty(t, obsolete)
	p1 := createRandomProfile(t, randomEntities.proj.ID, []string{})
	require.NotEmpty(t, p1)
	p2 := createRandomProfile(t, randomEntities.proj.ID, []string{})
	require.NotEmpty(t, p2)

	tests := []struct {
		name          string
		includeLabels []string
		excludeLabels []string
		expectedNames []string
	}{
		{
			name:          "list all profiles",
			includeLabels: []string{"*"},
			expectedNames: []string{health1.Name, health2.Name, obsolete.Name, p1.Name, p2.Name},
		},
		{
			name:          "list profiles with no labels",
			includeLabels: []string{},
			expectedNames: []string{p1.Name, p2.Name},
		},
		{
			name:          "list profiles with no labels using nil - default ",
			includeLabels: nil,
			expectedNames: []string{p1.Name, p2.Name},
		},
		{
			name:          "list profiles that have the obsolete label",
			includeLabels: []string{"obsolete"},
			expectedNames: []string{health2.Name, obsolete.Name},
		},
		{
			name:          "list profiles with the stacklok:health label",
			includeLabels: []string{"stacklok:health"},
			expectedNames: []string{health1.Name, health2.Name},
		},
		{
			name:          "list profile having both obsolete and stacklok:health labels",
			includeLabels: []string{"stacklok:health", "obsolete"},
			expectedNames: []string{health2.Name},
		},
		{
			name:          "list profiles with nonexistent labels",
			includeLabels: []string{"nonexistent"},
			expectedNames: []string{},
		},
		{
			name:          "both include and exclude",
			includeLabels: []string{"stacklok:health"},
			excludeLabels: []string{"obsolete"},
			expectedNames: []string{health1.Name},
		},
		{
			name:          "include all labels but exclude one",
			includeLabels: []string{"*"},
			excludeLabels: []string{"obsolete"},
			expectedNames: []string{health1.Name, p1.Name, p2.Name},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rows, err := testQueries.ListProfilesByProjectIDAndLabel(
				context.Background(), ListProfilesByProjectIDAndLabelParams{
					ProjectID:     randomEntities.proj.ID,
					IncludeLabels: tt.includeLabels,
					ExcludeLabels: tt.excludeLabels,
				})
			require.NoError(t, err)

			names := make([]string, 0, len(rows))
			for _, row := range rows {
				names = append(names, row.Profile.Name)
			}
			require.True(t, slices.Equal(names, tt.expectedNames), "expected %v, got %v", tt.expectedNames, names)
		})
	}
}

func TestCreateProfileStatusStoredProcedure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		ruleStatusSetupFn         func(profile Profile, randomEntities *testRandomEntities)
		expectedStatusAfterSetup  EvalStatusTypes
		ruleStatusModifyFn        func(profile Profile, randomEntities *testRandomEntities)
		expectedStatusAfterModify EvalStatusTypes
	}{
		{
			name: "Profile with no rule evaluations, should be pending",
			ruleStatusSetupFn: func(_ Profile, _ *testRandomEntities) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(_ Profile, _ *testRandomEntities) {
				// noop
			},
			expectedStatusAfterModify: EvalStatusTypesPending,
		},
		{
			name: "Profile with only success rule evaluation, should be success",
			ruleStatusSetupFn: func(_ Profile, _ *testRandomEntities) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name: "Profile with all skipped evaluations should be skipped",
			ruleStatusSetupFn: func(_ Profile, _ *testRandomEntities) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSkipped, "")

				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSkipped, "")
			},
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name: "Profile with one success and failure rule evaluation, should be failure",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesFailure, "")
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Profile with one success and one error results in error",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesError, "")
			},
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name: "Profile with one failure and one error results in error",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesError, "")
			},
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name: "Inserting success in addition to failure should result in failure",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Overwriting all to success results in success",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name: "Overwriting one to failure results in failure",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Skipped then failure results in failure",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSkipped, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesSkipped,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesFailure, "")
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Skipped then success results in success",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSkipped, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesSkipped,
			ruleStatusModifyFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
	}

	randomEntities := createTestRandomEntities(t)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := createRandomProfile(t, randomEntities.proj.ID, []string{})
			require.NotEmpty(t, profile)

			tt.ruleStatusSetupFn(profile, randomEntities)
			prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterSetup, prfStatusRow.ProfileStatus)

			tt.ruleStatusModifyFn(profile, randomEntities)
			prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterModify, prfStatusRow.ProfileStatus)

			err := testQueries.DeleteProfile(context.Background(), DeleteProfileParams{
				ID:        profile.ID,
				ProjectID: randomEntities.proj.ID,
			})
			require.NoError(t, err)
		})
	}
}

func TestCreateProfileStatusStoredDeleteProcedure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		ruleStatusSetupFn         func(profile Profile, randomEntities *testRandomEntities, delRepo *Repository)
		expectedStatusAfterSetup  EvalStatusTypes
		ruleStatusDeleteFn        func(delRepo *Repository)
		expectedStatusAfterModify EvalStatusTypes
	}{
		{
			name: "Removing last failure results in success",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities, delRepo *Repository) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")

				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusDeleteFn: func(delRepo *Repository) {
				err := testQueries.DeleteRepository(context.Background(), delRepo.ID)
				require.NoError(t, err)
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name: "Removing last error results in failure",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities, delRepo *Repository) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesSuccess, "")

				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesError, "")
			},

			expectedStatusAfterSetup: EvalStatusTypesError,
			ruleStatusDeleteFn: func(delRepo *Repository) {
				err := testQueries.DeleteRepository(context.Background(), delRepo.ID)
				require.NoError(t, err)
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Removing one error retains the other one",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities, delRepo *Repository) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesError, "")

				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesError, "")
			},

			expectedStatusAfterSetup: EvalStatusTypesError,
			ruleStatusDeleteFn: func(delRepo *Repository) {
				err := testQueries.DeleteRepository(context.Background(), delRepo.ID)
				require.NoError(t, err)
			},
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name: "Removing all but skipped returns skipped",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities, delRepo *Repository) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSkipped, "")

				upsertEvalStatus(
					t, profile.ID, delRepo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "")
			},

			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusDeleteFn: func(delRepo *Repository) {
				err := testQueries.DeleteRepository(context.Background(), delRepo.ID)
				require.NoError(t, err)
			},
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
	}

	randomEntities := createTestRandomEntities(t)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := createRandomProfile(t, randomEntities.proj.ID, []string{})
			require.NotEmpty(t, profile)

			delRepo := createRandomRepository(t, randomEntities.proj.ID, randomEntities.prov)

			tt.ruleStatusSetupFn(profile, randomEntities, &delRepo)
			prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterSetup, prfStatusRow.ProfileStatus)

			tt.ruleStatusDeleteFn(&delRepo)
			prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterModify, prfStatusRow.ProfileStatus)

			err := testQueries.DeleteProfile(context.Background(), DeleteProfileParams{
				ID:        profile.ID,
				ProjectID: randomEntities.proj.ID,
			})

			require.NoError(t, err)
		})
	}
}

type statusCount map[EvalStatusTypes]int

func getStatusCount(t *testing.T, rows []ListRuleEvaluationsByProfileIdRow) statusCount {
	t.Helper()

	sc := make(statusCount)
	for _, row := range rows {
		require.Equal(t, row.EvalStatus.Valid, true)
		sc[row.EvalStatus.EvalStatusTypes] += 1
	}
	return sc
}

func compareRows(t *testing.T, a, b *ListRuleEvaluationsByProfileIdRow) {
	t.Helper()

	require.Equal(t, a.EvalStatus, b.EvalStatus)
	require.Equal(t, a.EvalDetails, b.EvalDetails)
	require.Equal(t, a.RemStatus, b.RemStatus)
	require.Equal(t, a.RemDetails, b.RemDetails)
	require.Equal(t, a.RemMetadata, b.RemMetadata)
	require.Equal(t, a.AlertStatus, b.AlertStatus)
	require.Equal(t, a.AlertDetails, b.AlertDetails)
	require.Equal(t, a.AlertMetadata, b.AlertMetadata)
	require.Equal(t, a.Entity, b.Entity)
}

func rowForId(
	ruleTypeId uuid.UUID,
	rows []ListRuleEvaluationsByProfileIdRow,
) *ListRuleEvaluationsByProfileIdRow {
	for _, row := range rows {
		if row.RuleTypeID == ruleTypeId {
			return &row
		}
	}

	return nil
}

func verifyRow(
	t *testing.T,
	expectedRow *ListRuleEvaluationsByProfileIdRow,
	fetchedRows []ListRuleEvaluationsByProfileIdRow,
	rt RuleType,
	randomEntities *testRandomEntities,
) {
	t.Helper()

	if expectedRow == nil {
		return
	}
	row := rowForId(rt.ID, fetchedRows)
	compareRows(t, expectedRow, row)

	require.Equal(t, rt.ID, row.RuleTypeID)
	require.Equal(t, rt.Name, row.RuleTypeName)

	require.Equal(t, randomEntities.repo.RepoName, row.RepoName)
	require.Equal(t, randomEntities.repo.RepoOwner, row.RepoOwner)

	require.Equal(t, randomEntities.prov.ID, row.ProviderID)
}

func TestListRuleEvaluations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		profile           Profile
		ruleStatusSetupFn func(profile Profile, randomEntities *testRandomEntities)
		sc                statusCount
		totalRows         int
		rule1Expected     *ListRuleEvaluationsByProfileIdRow
		rule2Expected     *ListRuleEvaluationsByProfileIdRow
	}{
		{
			name: "Profile with one success rule evaluation",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
			},
			sc: statusCount{
				EvalStatusTypesSuccess: 1,
			},
			totalRows: 1,
			rule1Expected: &ListRuleEvaluationsByProfileIdRow{
				EvalStatus: NullEvalStatusTypes{
					EvalStatusTypes: EvalStatusTypesSuccess,
					Valid:           true,
				},
				EvalDetails: sql.NullString{
					String: "",
					Valid:  true,
				},
				RemStatus:    NullRemediationStatusTypes{},
				RemDetails:   sql.NullString{},
				AlertStatus:  NullAlertStatusTypes{},
				AlertDetails: sql.NullString{},
				Entity:       EntitiesRepository,
			},
		},
		{
			name: "Profile with one success and one failure rule evaluation",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID,
					EvalStatusTypesFailure, "this rule failed")
			},
			sc: statusCount{
				EvalStatusTypesSuccess: 1,
				EvalStatusTypesFailure: 1,
			},
			totalRows: 2,
			rule1Expected: &ListRuleEvaluationsByProfileIdRow{
				EvalStatus: NullEvalStatusTypes{
					EvalStatusTypes: EvalStatusTypesSuccess,
					Valid:           true,
				},
				EvalDetails: sql.NullString{
					String: "",
					Valid:  true,
				},
				RemStatus:    NullRemediationStatusTypes{},
				RemDetails:   sql.NullString{},
				AlertStatus:  NullAlertStatusTypes{},
				AlertDetails: sql.NullString{},
				Entity:       EntitiesRepository,
			},
			rule2Expected: &ListRuleEvaluationsByProfileIdRow{
				EvalStatus: NullEvalStatusTypes{
					EvalStatusTypes: EvalStatusTypesFailure,
					Valid:           true,
				},
				EvalDetails: sql.NullString{
					String: "this rule failed",
					Valid:  true,
				},
				RemStatus:    NullRemediationStatusTypes{},
				RemDetails:   sql.NullString{},
				AlertStatus:  NullAlertStatusTypes{},
				AlertDetails: sql.NullString{},
				Entity:       EntitiesRepository,
			},
		},
		{
			name: "Profile with one failed but remediated rule and an alert",
			ruleStatusSetupFn: func(profile Profile, randomEntities *testRandomEntities) {
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					EvalStatusTypesFailure, "this rule failed")
				upsertRemediationStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					RemediationStatusTypesSuccess, "this rule was remediated", json.RawMessage(`{"pr_number": "56"}`))
				upsertAlertStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID,
					AlertStatusTypesOn, "we alerted about this rule", json.RawMessage(`{"ghsa_id": "GHSA-xxxx-xxxx-xxxx"}`))
			},
			sc: statusCount{
				EvalStatusTypesFailure: 1,
			},
			totalRows: 1,
			rule1Expected: &ListRuleEvaluationsByProfileIdRow{
				EvalStatus: NullEvalStatusTypes{
					EvalStatusTypes: EvalStatusTypesFailure,
					Valid:           true,
				},
				EvalDetails: sql.NullString{
					String: "this rule failed",
					Valid:  true,
				},
				RemStatus: NullRemediationStatusTypes{
					RemediationStatusTypes: RemediationStatusTypesSuccess,
					Valid:                  true,
				},
				RemDetails: sql.NullString{
					String: "this rule was remediated",
					Valid:  true,
				},
				RemMetadata: pqtype.NullRawMessage{
					RawMessage: json.RawMessage(`{"pr_number": "56"}`),
					Valid:      true,
				},
				AlertStatus: NullAlertStatusTypes{
					AlertStatusTypes: AlertStatusTypesOn,
					Valid:            true,
				},
				AlertDetails: sql.NullString{
					String: "we alerted about this rule",
					Valid:  true,
				},
				AlertMetadata: pqtype.NullRawMessage{
					RawMessage: json.RawMessage(`{"ghsa_id": "GHSA-xxxx-xxxx-xxxx"}`),
					Valid:      true,
				},
				Entity: EntitiesRepository,
			},
		},
	}

	randomEntities := createTestRandomEntities(t)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.profile = createRandomProfile(t, randomEntities.proj.ID, []string{})
			require.NotEmpty(t, tt.profile)

			tt.ruleStatusSetupFn(tt.profile, randomEntities)
			evalStatusRows, err := testQueries.ListRuleEvaluationsByProfileId(context.Background(),
				ListRuleEvaluationsByProfileIdParams{
					ProfileID: tt.profile.ID,
				},
			)
			require.NoError(t, err)
			require.Len(t, evalStatusRows, tt.totalRows)
			require.Equal(t, getStatusCount(t, evalStatusRows), tt.sc)

			verifyRow(t, tt.rule1Expected, evalStatusRows, randomEntities.ruleType1, randomEntities)
			verifyRow(t, tt.rule2Expected, evalStatusRows, randomEntities.ruleType2, randomEntities)
		})
	}
}

func TestDeleteRuleEvaluations(t *testing.T) {
	t.Parallel()

	randomEntities := createTestRandomEntities(t)
	require.NotNil(t, randomEntities)

	profile := createRandomProfile(t, randomEntities.proj.ID, []string{})
	require.NotEmpty(t, profile)

	_, err := testQueries.UpsertProfileForEntity(context.Background(), UpsertProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          EntitiesRepository,
		ContextualRules: json.RawMessage(`{"key": "value"}`), // the content doesn't matter
	})
	require.NoError(t, err)

	id, err := testQueries.UpsertRuleEvaluations(context.Background(), UpsertRuleEvaluationsParams{
		ProfileID: profile.ID,
		RepositoryID: uuid.NullUUID{
			UUID:  randomEntities.repo.ID,
			Valid: true,
		},
		RuleTypeID: randomEntities.ruleType1.ID,
		RuleName:   randomEntities.ruleType1.Name,
		Entity:     EntitiesRepository,
	})
	require.NoError(t, err)
	require.NotNil(t, id)

	_, err = testQueries.UpsertRuleDetailsEval(context.Background(), UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     EvalStatusTypesFailure,
	})
	require.NoError(t, err)

	prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
	require.Equal(t, EvalStatusTypesFailure, prfStatusRow.ProfileStatus)

	err = testQueries.DeleteRuleStatusesForProfileAndRuleType(context.Background(), DeleteRuleStatusesForProfileAndRuleTypeParams{
		ProfileID:  profile.ID,
		RuleTypeID: randomEntities.ruleType1.ID,
		RuleName:   randomEntities.ruleType1.Name,
	})
	require.NoError(t, err)

	prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
	require.Equal(t, EvalStatusTypesPending, prfStatusRow.ProfileStatus)
}
