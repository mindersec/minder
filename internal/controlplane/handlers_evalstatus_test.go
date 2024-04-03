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

	"github.com/sqlc-dev/pqtype"
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
				AlertStatus: db.NullAlertStatusTypes{
					AlertStatusTypes: db.AlertStatusTypesOn,
				},
				AlertLastUpdated: sql.NullTime{
					Time: d,
				},
				AlertDetails: sql.NullString{
					String: "details go here",
				},
				AlertMetadata: pqtype.NullRawMessage{
					Valid:      true,
					RawMessage: []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`),
				},
				RepoOwner: "example",
				RepoName:  "test",
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
				AlertStatus: db.NullAlertStatusTypes{
					AlertStatusTypes: db.AlertStatusTypesOn,
				},
				AlertLastUpdated: sql.NullTime{
					Time: d,
				},
				AlertDetails: sql.NullString{
					String: "details go here",
				},
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
				AlertStatus: db.NullAlertStatusTypes{
					AlertStatusTypes: db.AlertStatusTypesOn,
				},
				AlertLastUpdated: sql.NullTime{
					Time: d,
				},
				AlertDetails: sql.NullString{
					String: "details go here",
				},
				AlertMetadata: pqtype.NullRawMessage{
					RawMessage: []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`),
				},
				RepoOwner: "",
				RepoName:  "test",
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
