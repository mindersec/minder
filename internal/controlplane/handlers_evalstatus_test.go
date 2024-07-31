//
// Copyright 2024 Stacklok, Inc.
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
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestBuildEvalResultAlertFromLRERow(t *testing.T) {
	t.Parallel()
	d := time.Now()
	for _, tc := range []struct {
		name   string
		sut    *db.ListRuleEvaluationsByProfileIdRow
		expect *minderv1.EvalResultAlert
	}{
		{
			name: "normal",
			sut: &db.ListRuleEvaluationsByProfileIdRow{
				AlertStatus:      db.AlertStatusTypesOn,
				AlertLastUpdated: d,
				AlertDetails:     "details go here",
				AlertMetadata:    []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`),
				RepoOwner:        "example",
				RepoName:         "test",
			},
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
				RepoOwner:        "",
				RepoName:         "test",
			},
			expect: &minderv1.EvalResultAlert{
				Status:      string(db.AlertStatusTypesOn),
				LastUpdated: timestamppb.New(d),
				Details:     "details go here",
				Url:         "",
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := buildEvalResultAlertFromLRERow(tc.sut)

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

func TestGetEntityName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dbEnt  db.Entities
		row    db.ListEvaluationHistoryRow
		output string
		err    bool
	}{
		{
			name:  "pull request",
			dbEnt: db.EntitiesPullRequest,
			row: db.ListEvaluationHistoryRow{
				RepoOwner: sql.NullString{
					Valid:  true,
					String: "stacklok",
				},
				RepoName: sql.NullString{
					Valid:  true,
					String: "minder",
				},
				PrNumber: sql.NullInt64{
					Valid: true,
					Int64: 12345,
				},
			},
			output: "stacklok/minder#12345",
		},
		{
			name:  "pull request no repo owner",
			dbEnt: db.EntitiesPullRequest,
			row: db.ListEvaluationHistoryRow{
				RepoName: sql.NullString{
					Valid:  true,
					String: "minder",
				},
				PrNumber: sql.NullInt64{
					Valid: true,
					Int64: 12345,
				},
			},
			err: true,
		},
		{
			name:  "pull request no repo name",
			dbEnt: db.EntitiesPullRequest,
			row: db.ListEvaluationHistoryRow{
				RepoOwner: sql.NullString{
					Valid:  true,
					String: "stacklok",
				},
				PrNumber: sql.NullInt64{
					Valid: true,
					Int64: 12345,
				},
			},
			err: true,
		},
		{
			name:  "pull request no pr number",
			dbEnt: db.EntitiesPullRequest,
			row: db.ListEvaluationHistoryRow{
				RepoOwner: sql.NullString{
					Valid:  true,
					String: "stacklok",
				},
				RepoName: sql.NullString{
					Valid:  true,
					String: "minder",
				},
			},
			err: true,
		},
		{
			name:  "artifact",
			dbEnt: db.EntitiesArtifact,
			row: db.ListEvaluationHistoryRow{
				ArtifactName: sql.NullString{
					Valid:  true,
					String: "artifact name",
				},
			},
			output: "artifact name",
		},
		{
			name:  "repository",
			dbEnt: db.EntitiesRepository,
			row: db.ListEvaluationHistoryRow{
				RepoOwner: sql.NullString{
					Valid:  true,
					String: "stacklok",
				},
				RepoName: sql.NullString{
					Valid:  true,
					String: "minder",
				},
			},
			output: "stacklok/minder",
		},
		{
			name:  "repository no repo owner",
			dbEnt: db.EntitiesRepository,
			row: db.ListEvaluationHistoryRow{
				RepoName: sql.NullString{
					Valid:  true,
					String: "minder",
				},
			},
			err: true,
		},
		{
			name:  "repository no repo name",
			dbEnt: db.EntitiesRepository,
			row: db.ListEvaluationHistoryRow{
				RepoOwner: sql.NullString{
					Valid:  true,
					String: "stacklok",
				},
			},
			err: true,
		},
		{
			name:  "build environments",
			dbEnt: db.EntitiesBuildEnvironment,
			row:   db.ListEvaluationHistoryRow{},
			err:   true,
		},
		{
			name:  "default",
			dbEnt: db.Entities("whatever"),
			row:   db.ListEvaluationHistoryRow{},
			err:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := getEntityName(
				tt.dbEnt,
				tt.row.RepoOwner,
				tt.row.RepoName,
				tt.row.PrNumber,
				tt.row.ArtifactName,
			)

			if tt.err {
				require.Error(t, err)
				require.Equal(t, "", res)
				return
			}

			require.NoError(t, err)
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
		rows   []db.ListEvaluationHistoryRow
		checkf func(*testing.T, db.ListEvaluationHistoryRow, *minderv1.EvaluationHistory)
		err    bool
	}{
		{
			name: "empty",
			rows: []db.ListEvaluationHistoryRow{},
		},
		{
			name: "happy path",
			rows: []db.ListEvaluationHistoryRow{
				{
					EvaluationID: uuid1,
					EvaluatedAt:  now,
					EntityType:   db.EntitiesRepository,
					EntityID:     entityid1,
					RepoOwner:    nullStr("stacklok"),
					RepoName:     nullStr("minder"),
					ProjectID:    uuid.NullUUID{},
					RuleType:     "rule_type",
					RuleName:     "rule_name",
					RuleSeverity: "unknown",
					ProfileName:  "profile_name",
				},
			},
		},
		{
			name: "order preserved",
			rows: []db.ListEvaluationHistoryRow{
				{
					EvaluationID: uuid1,
					EvaluatedAt:  now,
					EntityType:   db.EntitiesRepository,
					EntityID:     entityid1,
					RepoOwner:    nullStr("stacklok"),
					RepoName:     nullStr("minder"),
					ProjectID:    uuid.NullUUID{},
					RuleType:     "rule_type",
					RuleName:     "rule_name",
					RuleSeverity: "unknown",
					ProfileName:  "profile_name",
				},
				{
					EvaluationID: uuid2,
					EvaluatedAt:  now,
					EntityType:   db.EntitiesRepository,
					EntityID:     entityid2,
					RepoOwner:    nullStr("stacklok"),
					RepoName:     nullStr("frizbee"),
					ProjectID:    uuid.NullUUID{},
					RuleType:     "rule_type",
					RuleName:     "rule_name",
					RuleSeverity: "unknown",
					ProfileName:  "profile_name",
				},
			},
		},
		{
			name: "optional alert",
			rows: []db.ListEvaluationHistoryRow{
				{
					EvaluationID: uuid1,
					EvaluatedAt:  now,
					EntityType:   db.EntitiesRepository,
					EntityID:     entityid1,
					RepoOwner:    nullStr("stacklok"),
					RepoName:     nullStr("minder"),
					ProjectID:    uuid.NullUUID{},
					RuleType:     "rule_type",
					RuleName:     "rule_name",
					RuleSeverity: "unknown",
					ProfileName:  "profile_name",
					AlertStatus:  nullAlertStatusOK(),
					AlertDetails: nullStr("alert details"),
				},
			},
		},
		{
			name: "optional remediation",
			rows: []db.ListEvaluationHistoryRow{
				{
					EvaluationID:       uuid1,
					EvaluatedAt:        now,
					EntityType:         db.EntitiesRepository,
					EntityID:           entityid1,
					RepoOwner:          nullStr("stacklok"),
					RepoName:           nullStr("minder"),
					ProjectID:          uuid.NullUUID{},
					RuleType:           "rule_type",
					RuleName:           "rule_name",
					RuleSeverity:       "unknown",
					ProfileName:        "profile_name",
					RemediationStatus:  nullRemediationStatusTypesSuccess(),
					RemediationDetails: nullStr("remediation details"),
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
				require.Equal(t, row.EvaluationID.String(), item.Id)
				require.Equal(t, row.EvaluatedAt, item.EvaluatedAt.AsTime())
				require.Equal(t, row.EntityID.String(), item.Entity.Id)
				require.Equal(t, dbEntityToEntity(row.EntityType), item.Entity.Type)
				require.Equal(t, row.RuleType, item.Rule.RuleType)
				require.Equal(t, row.RuleName, item.Rule.Name)
				sev, err := dbSeverityToSeverity(row.RuleSeverity)
				require.NoError(t, err)
				require.Equal(t, sev, item.Rule.Severity)
				require.Equal(t, row.ProfileName, item.Rule.Profile)

				require.Equal(t, row.AlertStatus.Valid, item.Alert != nil)
				if row.AlertStatus.Valid {
					require.Equal(t,
						string(row.AlertStatus.AlertStatusTypes),
						item.Alert.Status,
					)
					require.Equal(t,
						row.AlertDetails.String,
						item.Alert.Details,
					)
				}

				require.Equal(t, row.RemediationStatus.Valid, item.Remediation != nil)
				if row.RemediationStatus.Valid {
					require.Equal(t,
						string(row.RemediationStatus.RemediationStatusTypes),
						item.Remediation.Status,
					)
					require.Equal(t,
						string(row.RemediationDetails.String),
						item.Remediation.Details,
					)
				}
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
