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

package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestEntitlementNotAvailable(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)

	_, err := testQueries.GetFeatureInProject(context.Background(), GetFeatureInProjectParams{
		Feature:   "unexistent_feature",
		ProjectID: proj.ID,
	})
	require.Error(t, err, "expected error when feature does not exist")
	require.ErrorIs(t, err, sql.ErrNoRows, "expected no rows error when feature does not exist")
}

func TestEntitlementAvailable(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)

	t.Log("inserting test feature")
	insertRes, err := testQueries.db.ExecContext(context.Background(),
		"INSERT INTO features (name, settings) VALUES ($1, $2)",
		"test_feature", `{"foo": "bar"}`)
	require.NoError(t, err, "expected no error when inserting feature")

	rows, err := insertRes.RowsAffected()
	require.Equal(t, rows, int64(1), "expected one row to be affected")
	require.NoError(t, err, "expected no error when getting rows affected")

	t.Log("inserting test entitlement")
	execRes, err := testQueries.db.ExecContext(context.Background(),
		"INSERT INTO entitlements (project_id, feature) VALUES ($1, $2)",
		proj.ID, "test_feature")
	require.NoError(t, err, "expected no error when inserting entitlement")

	rows, err = execRes.RowsAffected()
	require.Equal(t, rows, int64(1), "expected one row to be affected")
	require.NoError(t, err, "expected no error when getting rows affected")

	settings, err := testQueries.GetFeatureInProject(context.Background(), GetFeatureInProjectParams{
		Feature:   "test_feature",
		ProjectID: proj.ID,
	})
	require.NoError(t, err, "expected no error when getting feature in project")

	settingsMap := make(map[string]string)
	require.NoError(t, json.Unmarshal(settings, &settingsMap), "expected no error when unmarshalling settings")

	require.Equal(t, "bar", settingsMap["foo"], "expected settings to be equal")
}
