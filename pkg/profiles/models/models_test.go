// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package models_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
)

func TestRuleFromPB(t *testing.T) {
	t.Parallel()

	defStruct, err := structpb.NewStruct(map[string]any{
		"key": "value",
	})
	require.NoError(t, err)

	paramsStruct, err := structpb.NewStruct(map[string]any{
		"param1": "val1",
	})
	require.NoError(t, err)

	emptyStruct, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)

	ruleTypeID := uuid.New()

	tests := []struct {
		name       string
		ruleTypeID uuid.UUID
		pbRule     *minderv1.Profile_Rule
		expected   models.RuleInstance
	}{
		{
			name:       "converts all fields correctly",
			ruleTypeID: ruleTypeID,
			pbRule: &minderv1.Profile_Rule{
				Name:   "test-rule",
				Type:   "rule-type-1",
				Def:    defStruct,
				Params: paramsStruct,
			},
			expected: models.RuleInstance{
				ID:         uuid.Nil,
				Name:       "test-rule",
				Def:        map[string]any{"key": "value"},
				Params:     map[string]any{"param1": "val1"},
				RuleTypeID: ruleTypeID,
			},
		},
		{
			name:       "ID is always nil",
			ruleTypeID: uuid.New(),
			pbRule: &minderv1.Profile_Rule{
				Name:   "another-rule",
				Type:   "rule-type-2",
				Def:    emptyStruct,
				Params: emptyStruct,
			},
			expected: models.RuleInstance{
				ID:     uuid.Nil,
				Name:   "another-rule",
				Def:    map[string]any{},
				Params: map[string]any{},
			},
		},
		{
			name:       "empty name preserved",
			ruleTypeID: ruleTypeID,
			pbRule: &minderv1.Profile_Rule{
				Name:   "",
				Def:    emptyStruct,
				Params: emptyStruct,
			},
			expected: models.RuleInstance{
				ID:         uuid.Nil,
				Name:       "",
				Def:        map[string]any{},
				Params:     map[string]any{},
				RuleTypeID: ruleTypeID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := models.RuleFromPB(tt.ruleTypeID, tt.pbRule)

			require.Equal(t, uuid.Nil, result.ID, "ID should always be uuid.Nil")
			require.Equal(t, tt.expected.Name, result.Name)
			require.Equal(t, tt.expected.Def, result.Def)
			require.Equal(t, tt.expected.Params, result.Params)
			require.Equal(t, tt.ruleTypeID, result.RuleTypeID)
		})
	}
}

func TestRuleFromDB(t *testing.T) {
	t.Parallel()

	ruleID := uuid.New()
	ruleTypeID := uuid.New()

	tests := []struct {
		name      string
		dbRule    db.RuleInstance
		expected  models.RuleInstance
		expectErr bool
	}{
		{
			name: "valid rule with populated fields",
			dbRule: db.RuleInstance{
				ID:         ruleID,
				Name:       "db-rule",
				RuleTypeID: ruleTypeID,
				Def:        json.RawMessage(`{"enabled": true}`),
				Params:     json.RawMessage(`{"threshold": 10}`),
			},
			expected: models.RuleInstance{
				ID:         ruleID,
				Name:       "db-rule",
				RuleTypeID: ruleTypeID,
				Def:        map[string]any{"enabled": true},
				Params:     map[string]any{"threshold": float64(10)},
			},
		},
		{
			name: "valid rule with empty JSON objects",
			dbRule: db.RuleInstance{
				ID:         ruleID,
				Name:       "empty-rule",
				RuleTypeID: ruleTypeID,
				Def:        json.RawMessage(`{}`),
				Params:     json.RawMessage(`{}`),
			},
			expected: models.RuleInstance{
				ID:         ruleID,
				Name:       "empty-rule",
				RuleTypeID: ruleTypeID,
				Def:        map[string]any{},
				Params:     map[string]any{},
			},
		},
		{
			name: "invalid def JSON returns error",
			dbRule: db.RuleInstance{
				ID:         ruleID,
				Name:       "bad-def",
				RuleTypeID: ruleTypeID,
				Def:        json.RawMessage(`{invalid`),
				Params:     json.RawMessage(`{}`),
			},
			expectErr: true,
		},
		{
			name: "invalid params JSON returns error",
			dbRule: db.RuleInstance{
				ID:         ruleID,
				Name:       "bad-params",
				RuleTypeID: ruleTypeID,
				Def:        json.RawMessage(`{}`),
				Params:     json.RawMessage(`not-json`),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := models.RuleFromDB(tt.dbRule)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected.ID, result.ID)
			require.Equal(t, tt.expected.Name, result.Name)
			require.Equal(t, tt.expected.Def, result.Def)
			require.Equal(t, tt.expected.Params, result.Params)
			require.Equal(t, tt.expected.RuleTypeID, result.RuleTypeID)
		})
	}
}

func TestActionOptFromDB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dbState  db.NullActionType
		expected models.ActionOpt
	}{
		{
			name:     "null value returns unknown",
			dbState:  db.NullActionType{Valid: false},
			expected: models.ActionOptUnknown,
		},
		{
			name:     "on action type",
			dbState:  db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
			expected: models.ActionOptOn,
		},
		{
			name:     "off action type",
			dbState:  db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
			expected: models.ActionOptOff,
		},
		{
			name:     "dry_run action type",
			dbState:  db.NullActionType{ActionType: db.ActionTypeDryRun, Valid: true},
			expected: models.ActionOptDryRun,
		},
		{
			name:     "unrecognized action type returns unknown",
			dbState:  db.NullActionType{ActionType: db.ActionType("bogus"), Valid: true},
			expected: models.ActionOptUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := models.ActionOptFromDB(tt.dbState)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestActionOptOrDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		actionOpt  models.ActionOpt
		defaultVal models.ActionOpt
		expected   models.ActionOpt
	}{
		{
			name:       "unknown returns default",
			actionOpt:  models.ActionOptUnknown,
			defaultVal: models.ActionOptOn,
			expected:   models.ActionOptOn,
		},
		{
			name:       "on is preserved",
			actionOpt:  models.ActionOptOn,
			defaultVal: models.ActionOptOff,
			expected:   models.ActionOptOn,
		},
		{
			name:       "off is preserved",
			actionOpt:  models.ActionOptOff,
			defaultVal: models.ActionOptOn,
			expected:   models.ActionOptOff,
		},
		{
			name:       "dry_run is preserved",
			actionOpt:  models.ActionOptDryRun,
			defaultVal: models.ActionOptOn,
			expected:   models.ActionOptDryRun,
		},
		{
			name:       "unknown with unknown default returns unknown",
			actionOpt:  models.ActionOptUnknown,
			defaultVal: models.ActionOptUnknown,
			expected:   models.ActionOptUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := models.ActionOptOrDefault(tt.actionOpt, tt.defaultVal)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestActionOptString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		opt      models.ActionOpt
		expected string
	}{
		{
			name:     "on",
			opt:      models.ActionOptOn,
			expected: "on",
		},
		{
			name:     "off",
			opt:      models.ActionOptOff,
			expected: "off",
		},
		{
			name:     "dry_run",
			opt:      models.ActionOptDryRun,
			expected: "dry_run",
		},
		{
			name:     "unknown",
			opt:      models.ActionOptUnknown,
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.expected, tt.opt.String())
		})
	}
}

func TestSelectorSliceFromDB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []db.ProfileSelector
		expected []models.ProfileSelector
	}{
		{
			name:     "nil input returns empty slice",
			input:    nil,
			expected: []models.ProfileSelector{},
		},
		{
			name:     "empty input returns empty slice",
			input:    []db.ProfileSelector{},
			expected: []models.ProfileSelector{},
		},
		{
			name: "single selector with valid entity",
			input: []db.ProfileSelector{
				{
					Entity:   db.NullEntities{Entities: db.EntitiesRepository, Valid: true},
					Selector: "repo == 'my-repo'",
				},
			},
			expected: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repo == 'my-repo'",
				},
			},
		},
		{
			name: "selector with null entity",
			input: []db.ProfileSelector{
				{
					Entity:   db.NullEntities{Valid: false},
					Selector: "any-selector",
				},
			},
			expected: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_UNSPECIFIED,
					Selector: "any-selector",
				},
			},
		},
		{
			name: "multiple selectors",
			input: []db.ProfileSelector{
				{
					Entity:   db.NullEntities{Entities: db.EntitiesRepository, Valid: true},
					Selector: "selector-1",
				},
				{
					Entity:   db.NullEntities{Entities: db.EntitiesArtifact, Valid: true},
					Selector: "selector-2",
				},
			},
			expected: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "selector-1",
				},
				{
					Entity:   minderv1.Entity_ENTITY_ARTIFACTS,
					Selector: "selector-2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := models.SelectorSliceFromDB(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
