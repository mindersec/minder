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

package ruletypes_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/ruletypes"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// both create and update are bundled together since the testing harness is
// basically the same
func TestRuleTypeService(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		RuleType      *pb.RuleType
		DBSetup       dbf.DBMockBuilder
		ExpectedError string
		TestMethod    method
	}{
		{
			Name:          "CreateRuleType rejects nil rule",
			RuleType:      nil,
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rule with empty name",
			RuleType:      newRuleType(withBasicStructure, withRuleName("")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rule with invalid  name",
			RuleType:      newRuleType(withBasicStructure, withRuleName("I'm a little teapot")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rejects attempt to overwrite an existing rule",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: ruletypes.ErrRuleAlreadyExists.Error(),
			DBSetup:       dbf.NewDBMock(withSuccessfulGet),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType returns error on rule type lookup failure",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to get rule type",
			DBSetup:       dbf.NewDBMock(withFailedGet),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType returns error when unable to create rule type in database",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to create rule type",
			DBSetup:       dbf.NewDBMock(withNotFoundGet, withFailedCreate),
			TestMethod:    create,
		},
		{
			Name:       "CreateRuleType successfully creates a new rule type",
			RuleType:   newRuleType(withBasicStructure),
			DBSetup:    dbf.NewDBMock(withNotFoundGet, withSuccessfulCreate),
			TestMethod: create,
		},
		{
			Name:          "UpdateRuleType rejects malformed rule",
			RuleType:      newRuleType(),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    update,
		},
		{
			Name:          "UpdateRuleType rejects attempt to update non existent rule",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: ruletypes.ErrRuleNotFound.Error(),
			DBSetup:       dbf.NewDBMock(withNotFoundGet),
			TestMethod:    update,
		},
		{
			Name:          "UpdateRuleType returns error on rule type lookup failure",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to get rule type",
			DBSetup:       dbf.NewDBMock(withFailedGet),
			TestMethod:    update,
		},
		{
			Name:          "UpdateRuleType rejects update with incompatible rule schema",
			RuleType:      newRuleType(withBasicStructure, withIncompatibleDef),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			DBSetup:       dbf.NewDBMock(withSuccessfulGet),
			TestMethod:    update,
		},
		{
			Name:          "UpdateRuleType rejects update with incompatible param schema",
			RuleType:      newRuleType(withBasicStructure, withIncompatibleParams),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			DBSetup:       dbf.NewDBMock(withSuccessfulGet),
			TestMethod:    update,
		},
		{
			Name:          "UpdateRuleType returns error when unable to update rule type in database",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to update rule type",
			DBSetup:       dbf.NewDBMock(withSuccessfulGet, withFailedUpdate),
			TestMethod:    update,
		},
		{
			Name:       "UpdateRuleType successfully updates an existing rule",
			RuleType:   newRuleType(withBasicStructure),
			DBSetup:    dbf.NewDBMock(withSuccessfulGet, withSuccessfulUpdate),
			TestMethod: update,
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.DBSetup != nil {
				store = scenario.DBSetup(ctrl)
			}

			var err error
			var res *pb.RuleType
			svc := ruletypes.NewRuleTypeService(store)
			if scenario.TestMethod == create {
				res, err = svc.CreateRuleType(ctx, projectID, provider, scenario.RuleType)
			} else if scenario.TestMethod == update {
				res, err = svc.UpdateRuleType(ctx, projectID, provider, scenario.RuleType)
			} else {
				t.Fatal("unexpected method value")
			}

			if scenario.ExpectedError == "" {
				// due to the presence of autogenerated UUIDs and timestamps,
				// limit our assertions to the subset we deliberately set
				require.Equal(t, scenario.RuleType.Id, res.Id)
				require.Equal(t, scenario.RuleType.Description, res.Description)
				require.Equal(t, scenario.RuleType.Name, res.Name)
				require.Equal(t, scenario.RuleType.Severity.Value, res.Severity.Value)
				require.Nil(t, err)
			} else {
				require.Nil(t, res)
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

type method int

const (
	create method = iota
	update
	ruleName    = "rule_type"
	description = "this is my awesome rule"
)

var (
	ruleTypeID = uuid.New()
	projectID  = uuid.New()
	provider   = db.Provider{
		ID:   uuid.New(),
		Name: github.Github,
	}
	errDefault  = errors.New("oh no")
	oldRuleType = db.RuleType{
		ID:            ruleTypeID,
		Name:          ruleName,
		Definition:    []byte(`{}`),
		SeverityValue: "low",
	}
	expectation = db.RuleType{
		ID:            ruleTypeID,
		Name:          ruleName,
		Definition:    []byte(`{}`),
		SeverityValue: "high",
		Description:   description,
	}
	incompatibleSchema = &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"required": {
				Kind: &structpb.Value_StringValue{
					StringValue: "foobar",
				},
			},
		},
	}
)

func newRuleType(opts ...func(*pb.RuleType)) *pb.RuleType {
	ruleType := &pb.RuleType{}
	for _, opt := range opts {
		opt(ruleType)
	}
	return ruleType
}

func withBasicStructure(ruleType *pb.RuleType) {
	ruleType.Id = ptr.Ptr(ruleTypeID.String())
	ruleType.Name = ruleName
	ruleType.Description = description
	ruleType.Def = &pb.RuleType_Definition{
		InEntity:   string(pb.RepositoryEntity),
		RuleSchema: &structpb.Struct{},
		Ingest:     &pb.RuleType_Definition_Ingest{},
		Eval:       &pb.RuleType_Definition_Eval{},
	}
	ruleType.Severity = &pb.Severity{Value: pb.Severity_VALUE_HIGH}
}

func withRuleName(name string) func(ruleType *pb.RuleType) {
	return func(ruleType *pb.RuleType) {
		ruleType.Name = name
	}
}

func withIncompatibleDef(ruleType *pb.RuleType) {
	ruleType.Def.RuleSchema = incompatibleSchema
}

func withIncompatibleParams(ruleType *pb.RuleType) {
	ruleType.Def.ParamSchema = incompatibleSchema
}

func withSuccessfulGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(oldRuleType, nil)
}

func withNotFoundGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(db.RuleType{}, sql.ErrNoRows)
}

func withFailedGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(db.RuleType{}, errDefault)
}

func withSuccessfulCreate(mock dbf.DBMock) {
	mock.EXPECT().
		CreateRuleType(gomock.Any(), gomock.Any()).
		Return(expectation, nil)
}

func withFailedCreate(mock dbf.DBMock) {
	mock.EXPECT().
		CreateRuleType(gomock.Any(), gomock.Any()).
		Return(db.RuleType{}, errDefault)
}

func withSuccessfulUpdate(mock dbf.DBMock) {
	mock.EXPECT().
		UpdateRuleType(gomock.Any(), gomock.Any()).
		Return(expectation, nil)
}

func withFailedUpdate(mock dbf.DBMock) {
	mock.EXPECT().
		UpdateRuleType(gomock.Any(), gomock.Any()).
		Return(db.RuleType{}, errDefault)
}
