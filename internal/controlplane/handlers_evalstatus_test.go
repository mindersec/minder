// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	entmodels "github.com/mindersec/minder/internal/entities/models"
	mockpropssvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/history"
	mockhistory "github.com/mindersec/minder/internal/history/mock"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestBuildEvalResultAlertFromLRERow(t *testing.T) {
	t.Parallel()
	d := time.Now()
	for _, tc := range []struct {
		name   string
		sut    *db.ListRuleEvaluationsByProfileIdRow
		efp    *entmodels.EntityWithProperties
		expect *minderv1.EvalResultAlert
	}{
		{
			name: "normal",
			sut: &db.ListRuleEvaluationsByProfileIdRow{
				AlertStatus:      db.AlertStatusTypesOn,
				AlertLastUpdated: d,
				AlertDetails:     "details go here",
				AlertMetadata:    []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`),
			},
			efp: entmodels.NewEntityWithPropertiesFromInstance(entmodels.EntityInstance{
				ID:   uuid.New(),
				Type: minderv1.Entity_ENTITY_REPOSITORIES,
				Name: "example/test",
			}, nil),
			expect: &minderv1.EvalResultAlert{
				Status:      string(db.AlertStatusTypesOn),
				LastUpdated: timestamppb.New(d),
				Details:     "details go here",
				Url:         "https://github.com/example/test/security/advisories/GHAS-advisory_ID_here",
			},
		},
		{
			name: "no-advisory",
			sut: &db.ListRuleEvaluationsByProfileIdRow{
				AlertStatus:      db.AlertStatusTypesOn,
				AlertLastUpdated: d,
				AlertDetails:     "details go here",
			},
			efp: entmodels.NewEntityWithPropertiesFromInstance(entmodels.EntityInstance{
				ID:   uuid.New(),
				Type: minderv1.Entity_ENTITY_REPOSITORIES,
				Name: "example/test",
			}, nil),
			expect: &minderv1.EvalResultAlert{
				Status:      string(db.AlertStatusTypesOn),
				LastUpdated: timestamppb.New(d),
				Details:     "details go here",
				Url:         "",
			},
		},
		{
			name: "no-repo-owner",
			sut: &db.ListRuleEvaluationsByProfileIdRow{
				AlertStatus:      db.AlertStatusTypesOn,
				AlertLastUpdated: d,
				AlertDetails:     "details go here",
				AlertMetadata:    []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`),
			},
			efp: entmodels.NewEntityWithPropertiesFromInstance(entmodels.EntityInstance{
				ID:   uuid.New(),
				Type: minderv1.Entity_ENTITY_REPOSITORIES,
				Name: "test",
			}, nil),
			expect: &minderv1.EvalResultAlert{
				Status:      string(db.AlertStatusTypesOn),
				LastUpdated: timestamppb.New(d),
				Details:     "details go here",
				Url:         "",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := buildEvalResultAlertFromLRERow(tc.sut, tc.efp)

			require.Equal(t, tc.expect.Details, res.Details)
			require.Equal(t, tc.expect.LastUpdated, res.LastUpdated)
			require.Equal(t, tc.expect.Status, res.Status)
			require.Equal(t, tc.expect.Url, res.Url)
		})
	}
}

func TestDBEntityToEntity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  db.Entities
		output minderv1.Entity
	}{
		{
			name:   "pull request",
			input:  db.EntitiesPullRequest,
			output: minderv1.Entity_ENTITY_PULL_REQUESTS,
		},
		{
			name:   "artifact",
			input:  db.EntitiesArtifact,
			output: minderv1.Entity_ENTITY_ARTIFACTS,
		},
		{
			name:   "repository",
			input:  db.EntitiesRepository,
			output: minderv1.Entity_ENTITY_REPOSITORIES,
		},
		{
			name:   "build environments",
			input:  db.EntitiesBuildEnvironment,
			output: minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
		},
		{
			name:   "release",
			input:  db.EntitiesRelease,
			output: minderv1.Entity_ENTITY_RELEASE,
		},
		{
			name:   "pipeline run",
			input:  db.EntitiesPipelineRun,
			output: minderv1.Entity_ENTITY_PIPELINE_RUN,
		},
		{
			name:   "task run",
			input:  db.EntitiesTaskRun,
			output: minderv1.Entity_ENTITY_TASK_RUN,
		},
		{
			name:   "build",
			input:  db.EntitiesBuild,
			output: minderv1.Entity_ENTITY_BUILD,
		},
		{
			name:   "default",
			input:  db.Entities("whatever"),
			output: minderv1.Entity_ENTITY_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := dbEntityToEntity(tt.input)
			require.Equal(t, tt.output, res)
		})
	}
}

func TestFromEvaluationHistoryRows(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	entityid1 := uuid.MustParse("00000000-0000-0000-0000-000000000011")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	entityid2 := uuid.MustParse("00000000-0000-0000-0000-000000000022")

	tests := []struct {
		name   string
		rows   []*history.OneEvalHistoryAndEntity
		checkf func(*testing.T, db.ListEvaluationHistoryRow, *minderv1.EvaluationHistory)
		err    bool
	}{
		{
			name: "empty",
			rows: []*history.OneEvalHistoryAndEntity{},
		},
		{
			name: "happy path",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityid1,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: uuid1,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityid1,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
					},
				},
			},
		},
		{
			name: "order preserved",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityid1,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: uuid1,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityid1,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
					},
				},
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityid2,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "stacklok/frizbee",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: uuid2,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityid2,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
					},
				},
			},
		},
		{
			name: "optional alert",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityid1,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: uuid1,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityid1,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
						AlertStatus:  nullAlertStatusOK(),
						AlertDetails: nullStr("alert details"),
					},
				},
			},
		},
		{
			name: "optional remediation",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityid1,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID:       uuid1,
						EvaluatedAt:        now,
						EntityType:         db.EntitiesRepository,
						EntityID:           entityid1,
						ProjectID:          uuid.New(),
						RuleType:           "rule_type",
						RuleName:           "rule_name",
						RuleSeverity:       "unknown",
						ProfileName:        "profile_name",
						RemediationStatus:  nullRemediationStatusTypesSuccess(),
						RemediationDetails: nullStr("remediation details"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := fromEvaluationHistoryRows(context.Background(), tt.rows)

			if tt.err {
				require.Error(t, err)
				require.Equal(t, nil, res)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tt.rows), len(res))
			for i := 0; i < len(tt.rows); i++ {
				row := tt.rows[i]
				item := res[i]
				require.Equal(t, row.EvalHistoryRow.EvaluationID.String(), item.Id)
				require.Equal(t, row.EvalHistoryRow.EvaluatedAt, item.EvaluatedAt.AsTime())
				require.Equal(t, row.Entity.ID.String(), item.Entity.Id)
				require.Equal(t, row.Entity.Type, item.Entity.Type)
				require.Equal(t, row.EvalHistoryRow.RuleType, item.Rule.RuleType)
				require.Equal(t, row.EvalHistoryRow.RuleName, item.Rule.Name)
				sev, err := dbSeverityToSeverity(row.EvalHistoryRow.RuleSeverity)
				require.NoError(t, err)
				require.Equal(t, sev, item.Rule.Severity)
				require.Equal(t, row.EvalHistoryRow.ProfileName, item.Rule.Profile)

				require.Equal(t, row.EvalHistoryRow.AlertStatus.Valid, item.Alert != nil)
				if row.EvalHistoryRow.AlertStatus.Valid {
					require.Equal(t,
						string(row.EvalHistoryRow.AlertStatus.AlertStatusTypes),
						item.Alert.Status,
					)
					require.Equal(t,
						row.EvalHistoryRow.AlertDetails.String,
						item.Alert.Details,
					)
				}

				require.Equal(t, row.EvalHistoryRow.RemediationStatus.Valid, item.Remediation != nil)
				if row.EvalHistoryRow.RemediationStatus.Valid {
					require.Equal(t,
						string(row.EvalHistoryRow.RemediationStatus.RemediationStatusTypes),
						item.Remediation.Status,
					)
					require.Equal(t,
						string(row.EvalHistoryRow.RemediationDetails.String),
						item.Remediation.Details,
					)
				}

				// Verify that existing history rows do not set output
				require.Nil(t, item.Status.Output)
			}
		})
	}
}

func nullStr(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func nullAlertStatusOK() db.NullAlertStatusTypes {
	return db.NullAlertStatusTypes{
		AlertStatusTypes: db.AlertStatusTypesOn,
		Valid:            true,
	}
}

func nullRemediationStatusTypesSuccess() db.NullRemediationStatusTypes {
	return db.NullRemediationStatusTypes{
		RemediationStatusTypes: db.RemediationStatusTypesSuccess,
		Valid:                  true,
	}
}

func TestGetEvaluationHistoryIncludeOutputs(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	evalID := uuid.New()
	entityID := uuid.New()

	baseRow := db.GetEvaluationHistoryRow{
		EvaluationID:     evalID,
		EvaluatedAt:      time.Now().UTC(),
		EntityType:       db.EntitiesRepository,
		EntityID:         entityID,
		EntityName:       "mindersec/minder",
		ProjectID:        projectID,
		RuleType:         "rule_type",
		RuleName:         "rule_name",
		RuleSeverity:     db.SeverityUnknown,
		ProfileName:      "profile_name",
		EvaluationStatus: db.EvalStatusTypesSuccess,
	}

	tests := []struct {
		name         string
		outputErr    error
		outputRow    db.EvaluationOutput
		expectOutput *structpb.Value
	}{
		{
			name:         "include_outputs with sql.ErrNoRows",
			outputErr:    sql.ErrNoRows,
			expectOutput: nil,
		},
		{
			name:      "include_outputs with array output",
			outputErr: nil,
			outputRow: db.EvaluationOutput{
				ID: evalID,
				Output: pqtype.NullRawMessage{
					RawMessage: json.RawMessage(`[1,2,3]`),
					Valid:      true,
				},
			},
			expectOutput: func() *structpb.Value {
				v := &structpb.Value{}
				err := protojson.Unmarshal([]byte(`[1,2,3]`), v)
				require.NoError(t, err)
				return v
			}(),
		},
		{
			name:      "include_outputs with object output",
			outputErr: nil,
			outputRow: db.EvaluationOutput{
				ID: evalID,
				Output: pqtype.NullRawMessage{
					RawMessage: json.RawMessage(`{"a":{"b":"c"}}`),
					Valid:      true,
				},
			},
			expectOutput: func() *structpb.Value {
				v := &structpb.Value{}
				err := protojson.Unmarshal([]byte(`{"a":{"b":"c"}}`), v)
				require.NoError(t, err)
				return v
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockStore.EXPECT().
				GetEvaluationHistory(gomock.Any(), db.GetEvaluationHistoryParams{
					EvaluationID: evalID,
					ProjectID:    projectID,
				}).
				Return(baseRow, nil)
			mockStore.EXPECT().
				GetEvaluationOutput(gomock.Any(), evalID).
				Return(tt.outputRow, tt.outputErr)

			server := Server{store: mockStore}

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			resp, err := server.GetEvaluationHistory(ctx, &minderv1.GetEvaluationHistoryRequest{
				Id:             evalID.String(),
				IncludeOutputs: true,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.Evaluation)
			require.NotNil(t, resp.Evaluation.Status)

			if tt.expectOutput == nil {
				require.Nil(t, resp.Evaluation.Status.Output)
			} else {
				require.NotNil(t, resp.Evaluation.Status.Output)
				require.True(t, protojson.Format(tt.expectOutput) == protojson.Format(resp.Evaluation.Status.Output),
					"expected output %s, got %s",
					protojson.Format(tt.expectOutput),
					protojson.Format(resp.Evaluation.Status.Output),
				)
			}
		})
	}
}

func TestFromEvaluationHistoryRowsWithOutput(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	evalID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	entityID := uuid.MustParse("00000000-0000-0000-0000-000000000011")

	outputJSON := json.RawMessage(`{"finding":"something_wrong","severity":"high"}`)
	expectedOutput := &structpb.Value{}
	require.NoError(t, protojson.Unmarshal(outputJSON, expectedOutput))

	tests := []struct {
		name   string
		rows   []*history.OneEvalHistoryAndEntity
		expect *structpb.Value
	}{
		{
			name: "output included",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityID,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: evalID,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityID,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
						EvalOutput: pqtype.NullRawMessage{
							RawMessage: outputJSON,
							Valid:      true,
						},
					},
				},
			},
			expect: expectedOutput,
		},
		{
			name: "output null when not requested",
			rows: []*history.OneEvalHistoryAndEntity{
				{
					EntityWithProperties: entmodels.NewEntityWithPropertiesFromInstance(
						entmodels.EntityInstance{
							ID:   entityID,
							Type: minderv1.Entity_ENTITY_REPOSITORIES,
							Name: "mindersec/minder",
						}, nil),
					EvalHistoryRow: db.ListEvaluationHistoryRow{
						EvaluationID: evalID,
						EvaluatedAt:  now,
						EntityType:   db.EntitiesRepository,
						EntityID:     entityID,
						ProjectID:    uuid.New(),
						RuleType:     "rule_type",
						RuleName:     "rule_name",
						RuleSeverity: "unknown",
						ProfileName:  "profile_name",
						EvalOutput: pqtype.NullRawMessage{
							Valid: false,
						},
					},
				},
			},
			expect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := fromEvaluationHistoryRows(context.Background(), tt.rows)
			require.NoError(t, err)
			require.Len(t, res, 1)

			if tt.expect == nil {
				require.Nil(t, res[0].Status.Output)
			} else {
				require.NotNil(t, res[0].Status.Output)
				require.True(t,
					protojson.Format(tt.expect) == protojson.Format(res[0].Status.Output),
					"expected output %s, got %s",
					protojson.Format(tt.expect),
					protojson.Format(res[0].Status.Output),
				)
			}
		})
	}
}

func TestListEvaluationResultsIncludeOutputs(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	profileID := uuid.New()
	entityID := uuid.New()
	ruleTypeID := uuid.New()

	outputJSON := json.RawMessage(`{"finding":"something_wrong"}`)
	expectedOutput := &structpb.Value{}
	require.NoError(t, protojson.Unmarshal(outputJSON, expectedOutput))

	tests := []struct {
		name           string
		includeOutputs bool
		evalOutput     pqtype.NullRawMessage
		expectOutput   bool
	}{
		{
			name:           "include_outputs=true returns output",
			includeOutputs: true,
			evalOutput:     pqtype.NullRawMessage{RawMessage: outputJSON, Valid: true},
			expectOutput:   true,
		},
		{
			name:           "include_outputs=false omits output",
			includeOutputs: false,
			evalOutput:     pqtype.NullRawMessage{Valid: false},
			expectOutput:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockProps := mockpropssvc.NewMockPropertiesService(ctrl)

			efp := entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderv1.Entity_ENTITY_REPOSITORIES,
					Name: "mindersec/minder",
				}, nil)

			mockProps.EXPECT().
				EntityWithPropertiesByID(gomock.Any(), entityID, gomock.Any()).
				Return(efp, nil)
			mockProps.EXPECT().
				RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
			mockStore.EXPECT().
				GetProfileStatusByProject(gomock.Any(), projectID).
				Return([]db.GetProfileStatusByProjectRow{
					{ID: profileID, ProfileStatus: db.EvalStatusTypesSuccess},
				}, nil)
			mockStore.EXPECT().
				ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).
				Return([]db.ListProfilesByProjectIDAndLabelRow{
					{Profile: db.Profile{ID: profileID, Name: "test-profile", UpdatedAt: time.Now()}},
				}, nil)
			mockStore.EXPECT().
				ListRuleEvaluationsByProfileId(gomock.Any(), db.ListRuleEvaluationsByProfileIdParams{
					ProfileID:      profileID,
					IncludeOutputs: tt.includeOutputs,
				}).
				Return([]db.ListRuleEvaluationsByProfileIdRow{
					{
						RuleEvaluationID:      uuid.New(),
						EntityType:            db.EntitiesRepository,
						EntityID:              entityID,
						EntityName:            "mindersec/minder",
						ProjectID:             projectID,
						RuleTypeID:            ruleTypeID,
						RuleName:              "my_rule",
						RuleTypeName:          "rule_type_a",
						RuleTypeReleasePhase:  db.ReleaseStatusAlpha,
						EvalStatus:            db.EvalStatusTypesFailure,
						EvalLastUpdated:       time.Now(),
						RemStatus:             db.RemediationStatusTypesSkipped,
						RemLastUpdated:        time.Now(),
						AlertStatus:           db.AlertStatusTypesOff,
						AlertLastUpdated:      time.Now(),
						RuleTypeSeverityValue: db.SeverityMedium,
						EvalOutput:            tt.evalOutput,
					},
				}, nil)

			server := Server{store: mockStore, props: mockProps}
			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			resp, err := server.ListEvaluationResults(ctx, &minderv1.ListEvaluationResultsRequest{
				IncludeOutputs: tt.includeOutputs,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)

			var gotOutput *structpb.Value
			for _, ent := range resp.Entities {
				for _, prof := range ent.Profiles {
					for _, res := range prof.Results {
						if res.Output != nil {
							gotOutput = res.Output
						}
					}
				}
			}

			if tt.expectOutput {
				require.NotNil(t, gotOutput)
				require.True(t,
					protojson.Format(expectedOutput) == protojson.Format(gotOutput),
					"expected output %s, got %s",
					protojson.Format(expectedOutput),
					protojson.Format(gotOutput),
				)
			} else {
				require.Nil(t, gotOutput)
			}
		})
	}
}

func TestListEvaluationHistoryIncludeOutputs(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	evalID := uuid.New()
	entityID := uuid.New()

	outputJSON := json.RawMessage(`{"finding":"detected_issue"}`)
	expectedOutput := &structpb.Value{}
	require.NoError(t, protojson.Unmarshal(outputJSON, expectedOutput))

	now := time.Now().UTC()

	tests := []struct {
		name           string
		includeOutputs bool
		evalOutput     pqtype.NullRawMessage
		expectOutput   bool
	}{
		{
			name:           "include_outputs=true returns output",
			includeOutputs: true,
			evalOutput:     pqtype.NullRawMessage{RawMessage: outputJSON, Valid: true},
			expectOutput:   true,
		},
		{
			name:           "include_outputs=false omits output",
			includeOutputs: false,
			evalOutput:     pqtype.NullRawMessage{Valid: false},
			expectOutput:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockHist := mockhistory.NewMockEvaluationHistoryService(ctrl)

			efp := entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderv1.Entity_ENTITY_REPOSITORIES,
					Name: "mindersec/minder",
				}, nil)

			mockStore.EXPECT().BeginTransaction().Return(nil, nil)
			mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
			mockStore.EXPECT().Rollback(gomock.Any()).Return(nil)
			mockStore.EXPECT().Commit(gomock.Any()).Return(nil)

			mockHist.EXPECT().
				ListEvaluationHistory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), tt.includeOutputs).
				Return(&history.ListEvaluationHistoryResult{
					Data: []*history.OneEvalHistoryAndEntity{
						{
							EntityWithProperties: efp,
							EvalHistoryRow: db.ListEvaluationHistoryRow{
								EvaluationID:      evalID,
								EvaluatedAt:       now,
								EntityType:        db.EntitiesRepository,
								EntityID:          entityID,
								ProjectID:         projectID,
								RuleType:          "rule_type",
								RuleName:          "rule_name",
								RuleSeverity:      "unknown",
								ProfileName:       "profile_name",
								EvaluationStatus:  db.EvalStatusTypesFailure,
								EvaluationDetails: "some details",
								EvalOutput:        tt.evalOutput,
							},
						},
					},
					Next: []byte("+1234567890"),
					Prev: []byte("-1234567890"),
				}, nil)

			server := Server{store: mockStore, history: mockHist}
			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			resp, err := server.ListEvaluationHistory(ctx, &minderv1.ListEvaluationHistoryRequest{
				IncludeOutputs: tt.includeOutputs,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Data, 1)
			require.NotNil(t, resp.Data[0].Status)

			if tt.expectOutput {
				require.NotNil(t, resp.Data[0].Status.Output)
				require.True(t,
					protojson.Format(expectedOutput) == protojson.Format(resp.Data[0].Status.Output),
					"expected output %s, got %s",
					protojson.Format(expectedOutput),
					protojson.Format(resp.Data[0].Status.Output),
				)
			} else {
				require.Nil(t, resp.Data[0].Status.Output)
			}
		})
	}
}

func TestListEvaluationResultsEntityInfo(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	profileID := uuid.New()
	entityID := uuid.New()
	ruleTypeID := uuid.New()

	tests := []struct {
		name       string
		entityType db.Entities
		expectKeys []string
	}{
		{
			name:       "artifact entity info",
			entityType: db.EntitiesArtifact,
			expectKeys: []string{"artifact_name", "artifact_type", "entity_id", "entity_type", "provider", "artifact_id"},
		},
		{
			name:       "pull request entity info",
			entityType: db.EntitiesPullRequest,
			expectKeys: []string{"entity_id", "entity_type", "provider"}, // PR doesn't have specialized keys in getRuleEvalEntityInfo yet besides name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockProps := mockpropssvc.NewMockPropertiesService(ctrl)

			minderEntityType := minderv1.Entity_ENTITY_REPOSITORIES
			if tt.entityType == db.EntitiesArtifact {
				minderEntityType = minderv1.Entity_ENTITY_ARTIFACTS
			} else if tt.entityType == db.EntitiesPullRequest {
				minderEntityType = minderv1.Entity_ENTITY_PULL_REQUESTS
			}

			props := map[string]any{
				properties.PropertyName: "test-entity",
			}
			if tt.entityType == db.EntitiesArtifact {
				props[ghprop.ArtifactPropertyName] = "test-artifact"
				props[ghprop.ArtifactPropertyType] = "container"
			}

			psObj := properties.NewProperties(props)
			efp := entmodels.NewEntityWithPropertiesFromInstance(
				entmodels.EntityInstance{
					ID:   entityID,
					Type: minderEntityType,
					Name: "test-entity",
				}, psObj)

			mockProps.EXPECT().
				EntityWithPropertiesByID(gomock.Any(), entityID, gomock.Any()).
				Return(efp, nil)
			mockProps.EXPECT().
				RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
			mockStore.EXPECT().
				GetProfileStatusByProject(gomock.Any(), projectID).
				Return([]db.GetProfileStatusByProjectRow{
					{ID: profileID, ProfileStatus: db.EvalStatusTypesSuccess},
				}, nil)
			mockStore.EXPECT().
				ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).
				Return([]db.ListProfilesByProjectIDAndLabelRow{
					{Profile: db.Profile{ID: profileID, Name: "test-profile", UpdatedAt: time.Now()}},
				}, nil)
			mockStore.EXPECT().
				ListRuleEvaluationsByProfileId(gomock.Any(), gomock.Any()).
				Return([]db.ListRuleEvaluationsByProfileIdRow{
					{
						RuleEvaluationID:      uuid.New(),
						EntityType:            tt.entityType,
						EntityID:              entityID,
						EntityName:            "test-entity",
						ProjectID:             projectID,
						RuleTypeID:            ruleTypeID,
						RuleName:              "my_rule",
						RuleTypeName:          "rule_type_a",
						RuleTypeReleasePhase:  db.ReleaseStatusAlpha,
						EvalStatus:            db.EvalStatusTypesFailure,
						EvalLastUpdated:       time.Now(),
						RemStatus:             db.RemediationStatusTypesSkipped,
						RemLastUpdated:        time.Now(),
						AlertStatus:           db.AlertStatusTypesOff,
						AlertLastUpdated:      time.Now(),
						RuleTypeSeverityValue: db.SeverityMedium,
						Provider:              "github",
					},
				}, nil)

			server := Server{store: mockStore, props: mockProps}
			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			resp, err := server.ListEvaluationResults(ctx, &minderv1.ListEvaluationResultsRequest{})

			require.NoError(t, err)
			require.NotNil(t, resp)

			found := false
			for _, ent := range resp.Entities {
				for _, prof := range ent.Profiles {
					for _, res := range prof.Results {
						found = true
						for _, key := range tt.expectKeys {
							_, ok := res.EntityInfo[key]
							require.True(t, ok, "key %s not found in EntityInfo", key)
						}
					}
				}
			}
			require.True(t, found, "no results found in response")
		})
	}
}
