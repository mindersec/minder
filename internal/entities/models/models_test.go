// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package models_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/models"
)

func TestDbPropsToModel_ErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		dbProps   []db.Property
		expectErr bool
	}{
		{
			name: "returns error on malformed JSON payload",
			dbProps: []db.Property{
				{
					Key:   "repo_name",
					Value: json.RawMessage(`{"invalid_json": `),
				},
			},
			expectErr: true,
		},
		{
			name: "returns error on unsupported schema version",
			dbProps: []db.Property{
				{
					Key:   "repo_name",
					Value: json.RawMessage(`{"version": "v2", "value": "some-repo"}`),
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := models.DbPropsToModel(tc.dbProps)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
		})
	}
}

func TestDbPropToModel_ErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		dbProp    db.Property
		expectErr bool
	}{
		{
			name: "returns error on malformed JSON payload",
			dbProp: db.Property{
				Key:   "repo_name",
				Value: json.RawMessage(`{not_json}`),
			},
			expectErr: true,
		},
		{
			name: "returns error on unsupported schema version",
			dbProp: db.Property{
				Key:   "repo_name",
				Value: json.RawMessage(`{"version": "v999", "value": "test"}`),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := models.DbPropToModel(tc.dbProp)
			if tc.expectErr {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
		})
	}
}
