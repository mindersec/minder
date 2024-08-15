package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func createRepoSelector(t *testing.T, profileId uuid.UUID, sel string, comment string) ProfileSelector {
	t.Helper()
	return createEntitySelector(t, profileId, NullEntities{Entities: EntitiesRepository, Valid: true}, sel, comment)
}

func createEntitySelector(t *testing.T, profileId uuid.UUID, ent NullEntities, sel string, comment string) ProfileSelector {
	t.Helper()
	dbSel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
		ProfileID: profileId,
		Entity:    ent,
		Selector:  sel,
		Comment:   comment,
	})
	require.NoError(t, err)
	require.NotEmpty(t, dbSel)

	return dbSel
}

func createRuleInstance(t *testing.T, profileId uuid.UUID, ruleTypeID uuid.UUID, projectID uuid.UUID) uuid.UUID {
	t.Helper()
	ruleInstance, err := testQueries.UpsertRuleInstance(context.Background(), UpsertRuleInstanceParams{
		ProfileID:  profileId,
		RuleTypeID: ruleTypeID,
		Name:       fmt.Sprintf("rule_instance-%s", ruleTypeID),
		EntityType: EntitiesRepository,
		Def:        []byte("{}"),
		Params:     []byte("{}"),
		ProjectID:  projectID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ruleInstance)

	return ruleInstance
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

func createRandomRuleInstance(
	t *testing.T,
	projectID uuid.UUID,
	profileID uuid.UUID,
	ruleTypeID uuid.UUID,
) uuid.UUID {
	t.Helper()
	seed := time.Now().UnixNano()
	name := rand.RandomName(seed)
	riID, err := testQueries.UpsertRuleInstance(
		context.Background(),
		UpsertRuleInstanceParams{
			ProfileID:  profileID,
			RuleTypeID: ruleTypeID,
			Name:       name,
			EntityType: EntitiesRepository,
			ProjectID:  projectID,
			Def:        json.RawMessage(`{}`),
			Params:     json.RawMessage(`{}`),
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, riID)

	return riID
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
	t *testing.T,
	profileID uuid.UUID,
	repoID uuid.UUID,
	ruleTypeID uuid.UUID,
	ruleID uuid.UUID,
	evalStatus EvalStatusTypes,
	details string,
) {
	t.Helper()

	id, err := testQueries.UpsertRuleEvaluations(context.Background(), UpsertRuleEvaluationsParams{
		ProfileID: profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  repoID,
			Valid: true,
		},
		RuleInstanceID: ruleID,
		RuleTypeID:     ruleTypeID,
		Entity:         EntitiesRepository,
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

func createRuleEntity(
	t *testing.T,
	repoID uuid.UUID,
	ruleID uuid.UUID,
) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	id, err := testQueries.InsertEvaluationRuleEntity(ctx,
		InsertEvaluationRuleEntityParams{
			RuleID: ruleID,
			RepositoryID: uuid.NullUUID{
				UUID:  repoID,
				Valid: true,
			},
			PullRequestID: uuid.NullUUID{},
			ArtifactID:    uuid.NullUUID{},
			EntityType:    EntitiesRepository,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, id)
	return id
}

func upsertEvalHistoryStatus(
	t *testing.T,
	profileID uuid.UUID,
	ruleEntityID uuid.UUID,
	evalStatus EvalStatusTypes,
	details string,
) {
	t.Helper()
	ctx := context.Background()

	id, err := testQueries.InsertEvaluationStatus(ctx,
		InsertEvaluationStatusParams{
			RuleEntityID: ruleEntityID,
			Status:       evalStatus,
			Details:      details,
			Checkpoint:   []byte("{}"),
		},
	)
	require.NoError(t, err)

	err = testQueries.UpsertLatestEvaluationStatus(ctx,
		UpsertLatestEvaluationStatusParams{
			RuleEntityID:        ruleEntityID,
			EvaluationHistoryID: id,
			ProfileID:           profileID,
		},
	)
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

func matchIdWithListLabelRow(t *testing.T, id uuid.UUID) func(r ListProfilesByProjectIDAndLabelRow) bool {
	t.Helper()

	return func(r ListProfilesByProjectIDAndLabelRow) bool {
		return r.Profile.ID == id
	}
}

func findRowWithLabels(t *testing.T, rows []ListProfilesByProjectIDAndLabelRow, id uuid.UUID) int {
	t.Helper()

	return slices.IndexFunc(rows, matchIdWithListLabelRow(t, id))
}

func findBulkRow(t *testing.T, rows []BulkGetProfilesByIDRow, id uuid.UUID) int {
	t.Helper()

	return slices.IndexFunc(rows, func(r BulkGetProfilesByIDRow) bool {
		return r.Profile.ID == id
	})
}

func TestProfileListWithSelectors(t *testing.T) {
	t.Parallel()

	randomEntities := createTestRandomEntities(t)

	noSelectors := createRandomProfile(t, randomEntities.proj.ID, []string{})
	oneSelectorProfile := createRandomProfile(t, randomEntities.proj.ID, []string{})
	oneSel := createRepoSelector(t, oneSelectorProfile.ID, "one_selector1", "multi word comment")

	multiSelectorProfile := createRandomProfile(t, randomEntities.proj.ID, []string{})
	mulitSel1 := createRepoSelector(t, multiSelectorProfile.ID, "multi_selector1", "multi_comment1")
	mulitSel2 := createRepoSelector(t, multiSelectorProfile.ID, "multi_selector2", "multi_comment2")
	mulitSel3 := createRepoSelector(t, multiSelectorProfile.ID, "multi_selector3", "multi_comment3")

	genericSelectorProfile := createRandomProfile(t, randomEntities.proj.ID, []string{})
	genericSel := createEntitySelector(t, genericSelectorProfile.ID, NullEntities{}, "gen_selector1", "gen_comment1")

	t.Run("list profiles with selectors using the label list", func(t *testing.T) {
		t.Parallel()

		rows, err := testQueries.ListProfilesByProjectIDAndLabel(
			context.Background(), ListProfilesByProjectIDAndLabelParams{
				ProjectID: randomEntities.proj.ID,
			})
		require.NoError(t, err)

		require.Len(t, rows, 4)

		noSelIdx := findRowWithLabels(t, rows, noSelectors.ID)
		require.True(t, noSelIdx >= 0, "noSelectors not found in rows")
		require.Empty(t, rows[noSelIdx].ProfilesWithSelectors)

		oneSelIdx := findRowWithLabels(t, rows, oneSelectorProfile.ID)
		require.True(t, oneSelIdx >= 0, "oneSelector not found in rows")
		require.Len(t, rows[oneSelIdx].ProfilesWithSelectors, 1)
		require.Contains(t, rows[oneSelIdx].ProfilesWithSelectors, oneSel)

		multiSelIdx := findRowWithLabels(t, rows, multiSelectorProfile.ID)
		require.True(t, multiSelIdx >= 0, "multiSelectorProfile not found in rows")
		require.Len(t, rows[multiSelIdx].ProfilesWithSelectors, 3)
		require.Subset(t, rows[multiSelIdx].ProfilesWithSelectors, []ProfileSelector{mulitSel1, mulitSel2, mulitSel3})

		genSelIdx := findRowWithLabels(t, rows, genericSelectorProfile.ID)
		require.Len(t, rows[genSelIdx].ProfilesWithSelectors, 1)
		require.Contains(t, rows[genSelIdx].ProfilesWithSelectors, genericSel)
	})

	t.Run("Get profile by project and ID", func(t *testing.T) {
		t.Parallel()

		oneResult, err := testQueries.GetProfileByProjectAndID(context.Background(), GetProfileByProjectAndIDParams{
			ProjectID: randomEntities.proj.ID,
			ID:        oneSelectorProfile.ID,
		})
		require.NoError(t, err)
		require.Len(t, oneResult, 1)
		require.Len(t, oneResult[0].ProfilesWithSelectors, 1)
		require.Contains(t, oneResult[0].ProfilesWithSelectors, oneSel)

		noResult, err := testQueries.GetProfileByProjectAndID(context.Background(), GetProfileByProjectAndIDParams{
			ProjectID: randomEntities.proj.ID,
			ID:        noSelectors.ID,
		})
		require.NoError(t, err)
		require.Len(t, noResult, 1)
		require.Len(t, noResult[0].ProfilesWithSelectors, 0)

		multiResult, err := testQueries.GetProfileByProjectAndID(context.Background(), GetProfileByProjectAndIDParams{
			ProjectID: randomEntities.proj.ID,
			ID:        multiSelectorProfile.ID,
		})
		require.NoError(t, err)
		require.Len(t, multiResult, 1)
		require.Len(t, multiResult[0].ProfilesWithSelectors, 3)
		require.Subset(t, multiResult[0].ProfilesWithSelectors, []ProfileSelector{mulitSel1, mulitSel2, mulitSel3})
	})

	t.Run("Bulk get profiles by ID with selectors", func(t *testing.T) {
		t.Parallel()

		profileIDs := []uuid.UUID{
			noSelectors.ID, oneSelectorProfile.ID, multiSelectorProfile.ID, genericSelectorProfile.ID,
		}

		rows, err := testQueries.BulkGetProfilesByID(context.Background(), profileIDs)
		require.NoError(t, err)
		require.Len(t, rows, len(profileIDs))

		noSelIdx := findBulkRow(t, rows, noSelectors.ID)
		require.True(t, noSelIdx >= 0, "noSelectors not found in rows")
		require.Empty(t, rows[noSelIdx].ProfilesWithSelectors)

		oneSelIdx := findBulkRow(t, rows, oneSelectorProfile.ID)
		require.True(t, oneSelIdx >= 0, "oneSelector not found in rows")
		require.Len(t, rows[oneSelIdx].ProfilesWithSelectors, 1)
		require.Contains(t, rows[oneSelIdx].ProfilesWithSelectors, oneSel)

		multiSelIdx := findBulkRow(t, rows, multiSelectorProfile.ID)
		require.True(t, multiSelIdx >= 0, "multiSelectorProfile not found in rows")
		require.Len(t, rows[multiSelIdx].ProfilesWithSelectors, 3)
		require.Subset(t, rows[multiSelIdx].ProfilesWithSelectors, []ProfileSelector{mulitSel1, mulitSel2, mulitSel3})

		genSelIdx := findBulkRow(t, rows, genericSelectorProfile.ID)
		require.Len(t, rows[genSelIdx].ProfilesWithSelectors, 1)
		require.Contains(t, rows[genSelIdx].ProfilesWithSelectors, genericSel)
	})
}

func TestProfileLabels(t *testing.T) {
	t.Parallel()

	randomEntities := createTestRandomEntities(t)

	health1 := createRandomProfile(t, randomEntities.proj.ID, []string{"stacklok:health", "managed"})
	require.NotEmpty(t, health1)
	health2 := createRandomProfile(t, randomEntities.proj.ID, []string{"stacklok:health", "obsolete", "managed"})
	require.NotEmpty(t, health2)
	obsolete := createRandomProfile(t, randomEntities.proj.ID, []string{"obsolete"})
	require.NotEmpty(t, obsolete)
	managed := createRandomProfile(t, randomEntities.proj.ID, []string{"managed"})
	require.NotEmpty(t, managed)
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
			expectedNames: []string{health1.Name, health2.Name, obsolete.Name, p1.Name, p2.Name, managed.Name},
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
			expectedNames: []string{health1.Name, p1.Name, p2.Name, managed.Name},
		},
		{
			name:          "excluding filters out any profile with exclude labels",
			includeLabels: []string{"*"},
			excludeLabels: []string{"stacklok:health", "managed"},
			expectedNames: []string{obsolete.Name, p1.Name, p2.Name},
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
			slices.Sort(names)
			slices.Sort(tt.expectedNames)
			require.True(t, slices.Equal(names, tt.expectedNames), "expected %v, got %v", tt.expectedNames, names)
		})
	}
}

func TestCreateProfileStatusSingleRuleTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		rule1StatusPre            EvalStatusTypes
		expectedStatusAfterSetup  EvalStatusTypes
		rule1StatusPost           EvalStatusTypes
		expectedStatusAfterModify EvalStatusTypes
	}{
		// transitions from skipped
		{
			name:                      "skipped -> skipped = skipped",
			rule1StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "skipped -> failure = failure",
			rule1StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "skipped -> error = error",
			rule1StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "skipped -> success = success",
			rule1StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},

		// transitions from error
		{
			name:                      "error -> skipped = skipped",
			rule1StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "error -> error = error",
			rule1StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "error -> failure = failure",
			rule1StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "error -> success = success",
			rule1StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},

		// transitions from failure
		{
			name:                      "failure -> skipped = skipped",
			rule1StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "failure -> error = error",
			rule1StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "failure -> failure = failure",
			rule1StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "failure -> success = success",
			rule1StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},

		// transitions from success
		{
			name:                      "success -> skipped = skipped",
			rule1StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "success -> error = error",
			rule1StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "success -> failure = failure",
			rule1StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "success -> success = success",
			rule1StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
	}

	randomEntities := createTestRandomEntities(t)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := createRandomProfile(t, randomEntities.proj.ID, []string{})
			ruleID := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
			require.NotEmpty(t, profile)

			ruleEntityID := createRuleEntity(t, randomEntities.repo.ID, ruleID)

			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID,
				tt.rule1StatusPre,
				"foo",
			)

			prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterSetup, prfStatusRow.ProfileStatus,
				"Status BEFORE transition is %s, expected %s",
				prfStatusRow.ProfileStatus, tt.expectedStatusAfterSetup,
			)

			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID,
				tt.rule1StatusPost,
				"foo",
			)
			prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterModify, prfStatusRow.ProfileStatus,
				"Status AFTER transition is %s, expected %s",
				prfStatusRow.ProfileStatus, tt.expectedStatusAfterModify,
			)

			err := testQueries.DeleteProfile(context.Background(), DeleteProfileParams{
				ID:        profile.ID,
				ProjectID: randomEntities.proj.ID,
			})
			require.NoError(t, err)
		})
	}
}

func TestCreateProfileStatusMultiRuleTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		rule1StatusPre            EvalStatusTypes
		rule2StatusPre            EvalStatusTypes
		expectedStatusAfterSetup  EvalStatusTypes
		rule1StatusPost           EvalStatusTypes
		rule2StatusPost           EvalStatusTypes
		expectedStatusAfterModify EvalStatusTypes
	}{
		{
			name:                      "0x00",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x01",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x02",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x03",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x04",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x05",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x06",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x07",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x08",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x09",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x0a",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x0b",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x0c",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x0d",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x0e",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x0f",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSkipped,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x10",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x11",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x12",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x13",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x14",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x15",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x16",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x17",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x18",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x19",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x1a",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x1b",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x1c",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x1d",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x1e",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x1f",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x20",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x21",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x22",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x23",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x24",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x25",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x26",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x27",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x28",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x29",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x2a",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x2b",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x2c",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x2d",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x2e",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x2f",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x30",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x31",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x32",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x33",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x34",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x35",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x36",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x37",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x38",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x39",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x3a",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x3b",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x3c",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x3d",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x3e",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x3f",
			rule1StatusPre:            EvalStatusTypesSkipped,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x40",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x41",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x42",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x43",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x44",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x45",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x46",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x47",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x48",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x49",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x4a",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x4b",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x4c",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x4d",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x4e",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x4f",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x50",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x51",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x52",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x53",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x54",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x55",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x56",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x57",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x58",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x59",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x5a",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x5b",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x5c",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x5d",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x5e",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x5f",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x60",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x61",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x62",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x63",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x64",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x65",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x66",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x67",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x68",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x69",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x6a",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x6b",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x6c",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x6d",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x6e",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x6f",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x70",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x71",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x72",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x73",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x74",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x75",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x76",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x77",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x78",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x79",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x7a",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x7b",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x7c",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x7d",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x7e",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x7f",
			rule1StatusPre:            EvalStatusTypesFailure,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x80",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x81",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x82",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x83",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x84",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x85",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x86",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x87",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x88",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x89",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x8a",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x8b",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x8c",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x8d",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x8e",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x8f",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x90",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0x91",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x92",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x93",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x94",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x95",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x96",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x97",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x98",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x99",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x9a",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x9b",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x9c",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0x9d",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0x9e",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0x9f",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xa0",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xa1",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xa2",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xa3",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xa4",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xa5",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xa6",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xa7",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xa8",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xa9",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xaa",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xab",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xac",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xad",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xae",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xaf",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xb0",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xb1",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xb2",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xb3",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xb4",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xb5",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xb6",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xb7",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xb8",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xb9",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xba",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xbb",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xbc",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xbd",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xbe",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xbf",
			rule1StatusPre:            EvalStatusTypesError,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xc0",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xc1",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xc2",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xc3",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xc4",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xc5",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xc6",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xc7",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xc8",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xc9",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xca",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xcb",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xcc",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xcd",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xce",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xcf",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSkipped,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xd0",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xd1",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xd2",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xd3",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xd4",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xd5",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xd6",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xd7",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xd8",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xd9",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xda",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xdb",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xdc",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xdd",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xde",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xdf",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesFailure,
			expectedStatusAfterSetup:  EvalStatusTypesFailure,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xe0",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xe1",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xe2",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xe3",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xe4",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xe5",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xe6",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xe7",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xe8",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xe9",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xea",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xeb",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xec",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xed",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xee",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xef",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesError,
			expectedStatusAfterSetup:  EvalStatusTypesError,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xf0",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name:                      "0xf1",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xf2",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xf3",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSkipped,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xf4",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xf5",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xf6",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xf7",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesFailure,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xf8",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xf9",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xfa",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xfb",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesError,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xfc",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSkipped,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name:                      "0xfd",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesFailure,
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name:                      "0xfe",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesError,
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name:                      "0xff",
			rule1StatusPre:            EvalStatusTypesSuccess,
			rule2StatusPre:            EvalStatusTypesSuccess,
			expectedStatusAfterSetup:  EvalStatusTypesSuccess,
			rule1StatusPost:           EvalStatusTypesSuccess,
			rule2StatusPost:           EvalStatusTypesSuccess,
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			randomEntities := createTestRandomEntities(t)

			profile := createRandomProfile(t, randomEntities.proj.ID, []string{})
			ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
			ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType2.ID, profile.ProjectID)
			require.NotEmpty(t, profile)

			ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID1)
			ruleEntityID2 := createRuleEntity(t, randomEntities.repo.ID, ruleID2)

			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID1,
				tt.rule1StatusPre,
				"foo",
			)
			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID2,
				tt.rule2StatusPre,
				"foo",
			)
			prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterSetup, prfStatusRow.ProfileStatus,
				"Status BEFORE transition is %s, expected %s",
				prfStatusRow.ProfileStatus, tt.expectedStatusAfterSetup,
			)

			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID1,
				tt.rule1StatusPost,
				"foo",
			)
			upsertEvalHistoryStatus(
				t,
				profile.ID,
				ruleEntityID2,
				tt.rule2StatusPost,
				"foo",
			)

			prfStatusRow = profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterModify, prfStatusRow.ProfileStatus,
				"Status AFTER transition is %s, expected %s",
				prfStatusRow.ProfileStatus, tt.expectedStatusAfterModify,
			)

			err := testQueries.DeleteProfile(context.Background(), DeleteProfileParams{
				ID:        profile.ID,
				ProjectID: randomEntities.proj.ID,
			})
			require.NoError(t, err)
		})
	}
}

func TestCreateProfileStatusStoredProcedure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		ruleStatusSetupFn         func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID)
		expectedStatusAfterSetup  EvalStatusTypes
		ruleStatusModifyFn        func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID)
		expectedStatusAfterModify EvalStatusTypes
	}{
		{
			name: "Profile with no rule evaluations, should be pending",
			ruleStatusSetupFn: func(_ Profile, _ uuid.UUID, _ uuid.UUID) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(_ Profile, _ uuid.UUID, _ uuid.UUID) {
				// noop
			},
			expectedStatusAfterModify: EvalStatusTypesPending,
		},
		{
			name: "Profile with only success rule evaluation, should be success",
			ruleStatusSetupFn: func(_ Profile, _ uuid.UUID, _ uuid.UUID) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name: "Profile with all skipped evaluations should be skipped",
			ruleStatusSetupFn: func(_ Profile, _ uuid.UUID, _ uuid.UUID) {
				// noop
			},
			expectedStatusAfterSetup: EvalStatusTypesPending,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesSkipped,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesSkipped,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesSkipped,
		},
		{
			name: "Profile with one success and failure rule evaluation, should be failure",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesFailure,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Profile with one success and one error results in error",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, _ uuid.UUID, ruleEntityID uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesError,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name: "Profile with one failure and one error results in error",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesFailure,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, _ uuid.UUID, ruleEntityID uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesError,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesError,
		},
		{
			name: "Inserting success in addition to failure should result in failure",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesFailure,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, _ uuid.UUID, ruleEntityID uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Overwriting all to success results in success",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesFailure,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesFailure,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID1 uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesSuccess,
		},
		{
			name: "Overwriting one to failure results in failure",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesSuccess,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesSuccess,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesSuccess,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID1 uuid.UUID, ruleEntityID2 uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesFailure,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesFailure,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Skipped then failure results in failure",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSkipped,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesSkipped,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesFailure,
					"",
				)
			},
			expectedStatusAfterModify: EvalStatusTypesFailure,
		},
		{
			name: "Skipped then success results in success",
			ruleStatusSetupFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSkipped,
					"",
				)
			},
			expectedStatusAfterSetup: EvalStatusTypesSkipped,
			ruleStatusModifyFn: func(profile Profile, ruleEntityID uuid.UUID, _ uuid.UUID) {
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID,
					EvalStatusTypesSuccess,
					"",
				)
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

			ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
			ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType2.ID, profile.ProjectID)
			require.NotEmpty(t, profile)

			ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID1)
			ruleEntityID2 := createRuleEntity(t, randomEntities.repo.ID, ruleID2)

			tt.ruleStatusSetupFn(profile, ruleEntityID1, ruleEntityID2)
			prfStatusRow := profileIDStatusByIdAndProject(t, profile.ID, randomEntities.proj.ID)
			require.Equal(t, tt.expectedStatusAfterSetup, prfStatusRow.ProfileStatus)

			tt.ruleStatusModifyFn(profile, ruleEntityID1, ruleEntityID2)
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
				ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
				ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType2.ID, profile.ProjectID)
				require.NotEmpty(t, profile)

				ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID1)
				ruleEntityID2 := createRuleEntity(t, randomEntities.repo.ID, ruleID2)
				ruleEntityID3 := createRuleEntity(t, delRepo.ID, ruleID1)
				ruleEntityID4 := createRuleEntity(t, delRepo.ID, ruleID2)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesSuccess,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesSuccess,
					"",
				)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID3,
					EvalStatusTypesFailure,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID4,
					EvalStatusTypesSuccess,
					"",
				)
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
				ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
				ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType2.ID, profile.ProjectID)
				require.NotEmpty(t, profile)

				ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID1)
				ruleEntityID2 := createRuleEntity(t, randomEntities.repo.ID, ruleID2)
				ruleEntityID3 := createRuleEntity(t, delRepo.ID, ruleID1)
				ruleEntityID4 := createRuleEntity(t, delRepo.ID, ruleID2)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesFailure,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesSuccess,
					"",
				)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID3,
					EvalStatusTypesSuccess,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID4,
					EvalStatusTypesError,
					"",
				)
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
				ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
				ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType2.ID, profile.ProjectID)
				require.NotEmpty(t, profile)

				ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID1)
				ruleEntityID2 := createRuleEntity(t, randomEntities.repo.ID, ruleID2)
				ruleEntityID3 := createRuleEntity(t, delRepo.ID, ruleID1)
				ruleEntityID4 := createRuleEntity(t, delRepo.ID, ruleID2)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesFailure,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesError,
					"",
				)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID3,
					EvalStatusTypesSuccess,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID4,
					EvalStatusTypesError,
					"",
				)
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
				ruleID := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)

				ruleEntityID1 := createRuleEntity(t, randomEntities.repo.ID, ruleID)
				ruleEntityID2 := createRuleEntity(t, delRepo.ID, ruleID)

				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID1,
					EvalStatusTypesSkipped,
					"",
				)
				upsertEvalHistoryStatus(
					t,
					profile.ID,
					ruleEntityID2,
					EvalStatusTypesFailure,
					"",
				)
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

	require.Equal(t, randomEntities.prov.Name, row.Provider)
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
				ruleID := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)

				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID, ruleID,
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
				ruleID1 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)
				ruleID2 := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)

				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID, ruleID1,
					EvalStatusTypesSuccess, "")
				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType2.ID, ruleID2,
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
				ruleID := createRuleInstance(t, profile.ID, randomEntities.ruleType1.ID, profile.ProjectID)

				upsertEvalStatus(
					t, profile.ID, randomEntities.repo.ID, randomEntities.ruleType1.ID, ruleID,
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
