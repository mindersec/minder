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
	"github.com/mindersec/minder/internal/history"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
			res, err := fromEvaluationHistoryRows(tt.rows)

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
