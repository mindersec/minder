// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletypes_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	"github.com/mindersec/minder/internal/util/ptr"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// both create and update are bundled together since the testing harness is
// basically the same
func TestRuleTypeService(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name           string
		RuleType       *pb.RuleType
		DBSetup        dbf.DBMockBuilder
		ExpectedError  string
		TestMethod     method
		SubscriptionID uuid.UUID
	}{
		{
			Name:          "CreateRuleType rejects nil rule",
			RuleType:      nil,
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		// TODO: these tests should live with the validator, not this service
		{
			Name:          "CreateRuleType rejects rule with empty name",
			RuleType:      newRuleType(withBasicStructure, withRuleName("")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rejects rule with invalid name",
			RuleType:      newRuleType(withBasicStructure, withRuleName("I'm a little teapot")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rejects rule with multiple slashes",
			RuleType:      newRuleType(withBasicStructure, withRuleName("I'm a little teapot")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rejects rule where part of a namespaced name is invalid",
			RuleType:      newRuleType(withBasicStructure, withRuleName("validnamespace/I'm a little teapot")),
			ExpectedError: ruletypes.ErrRuleTypeInvalid.Error(),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType rejects attempt to create a namespaced rule when no subscription ID is passed",
			RuleType:      newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			ExpectedError: "cannot create a rule type, data source or profile with a namespace through the API",
			DBSetup:       dbf.NewDBMock(),
			TestMethod:    create,
		},
		{
			Name:           "CreateRuleType rejects attempt to create a non-namespaced rule when no subscription ID is passed",
			RuleType:       newRuleType(withBasicStructure),
			ExpectedError:  "rule types, data sources and profiles from subscriptions must have namespaced names",
			DBSetup:        dbf.NewDBMock(),
			SubscriptionID: subscriptionID,
			TestMethod:     create,
		},
		{
			Name:          "CreateRuleType rejects attempt to overwrite an existing rule",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: ruletypes.ErrRuleAlreadyExists.Error(),
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withSuccessfulGet),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType returns error on rule type lookup failure",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to get rule type",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withFailedGet),
			TestMethod:    create,
		},
		{
			Name:          "CreateRuleType returns error when unable to create rule type in database",
			RuleType:      newRuleType(withBasicStructure),
			ExpectedError: "failed to create rule type",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withFailedCreate),
			TestMethod:    create,
		},
		{
			Name:       "CreateRuleType successfully creates a new rule type",
			RuleType:   newRuleType(withBasicStructure),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod: create,
		},
		{
			Name:           "CreateRuleType successfully creates a new namespaced rule type",
			RuleType:       newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			DBSetup:        dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulNamespaceCreate, withSuccessfulDeleteRuleTypeDataSource),
			SubscriptionID: subscriptionID,
			TestMethod:     create,
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
			Name:          "UpdateRuleType rejects attempt to update a rule type from a bundle",
			RuleType:      newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			ExpectedError: "attempted to edit a rule type or profile which belongs to a bundle",
			DBSetup:       dbf.NewDBMock(withSuccessfulNamespaceGet),
			TestMethod:    update,
		},
		{
			Name:           "UpdateRuleType rejects attempt to another subscription's rule types",
			RuleType:       newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			ExpectedError:  "attempted to edit a rule type or profile which belongs to a bundle",
			DBSetup:        dbf.NewDBMock(withSuccessfulNamespaceGet),
			SubscriptionID: uuid.New(),
			TestMethod:     update,
		},
		{
			Name:           "UpdateSubscriptionRuleType rejects attempt to update a customer rule",
			RuleType:       newRuleType(withBasicStructure),
			ExpectedError:  "attempted to edit a customer rule type or profile with bundle operation",
			DBSetup:        dbf.NewDBMock(withSuccessfulGet),
			SubscriptionID: subscriptionID,
			TestMethod:     update,
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
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod: update,
		},
		{
			Name:           "UpdateRuleType successfully updates an existing rule with subscription",
			RuleType:       newRuleType(withBasicStructure),
			DBSetup:        dbf.NewDBMock(withHierarchyGet, withSuccessfulNamespaceGet, withSuccessfulUpdate, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod:     update,
			SubscriptionID: subscriptionID,
		},
		{
			Name:           "UpsertRuleType successfully creates a new namespaced rule type",
			RuleType:       newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			DBSetup:        dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulNamespaceCreate, withSuccessfulDeleteRuleTypeDataSource),
			SubscriptionID: subscriptionID,
			TestMethod:     upsert,
		},
		{
			Name:           "UpsertRuleType successfully updates an existing rule",
			RuleType:       newRuleType(withBasicStructure, withRuleName(namespacedRuleName)),
			DBSetup:        dbf.NewDBMock(withHierarchyGet, withHierarchyGet, withSuccessfulNamespaceGet, withSuccessfulUpdate, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod:     upsert,
			SubscriptionID: subscriptionID,
		},
		{
			Name:       "CreateRuleType with EvaluationFailureMessage",
			RuleType:   newRuleType(withBasicStructure, withEvaluationFailureMessage(shortFailureMessage)),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreateWithEvaluationFailureMessage, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod: create,
		},
		{
			Name:       "UpdateRuleType with EvaluationFailureMessage",
			RuleType:   newRuleType(withBasicStructure, withEvaluationFailureMessage(shortFailureMessage)),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdateWithEvaluationFailureMessage, withSuccessfulDeleteRuleTypeDataSource),
			TestMethod: update,
		},
		{
			Name:       "CreateRuleType with Data Sources",
			RuleType:   newRuleType(withBasicStructure, withDataSources),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withSuccessfulGetDataSourcesByName(1), withSuccessfulDeleteRuleTypeDataSource, withSuccessfulAddRuleTypeDataSourceReference),
			TestMethod: create,
		},
		{
			Name:       "UpdateRuleType with Data Sources",
			RuleType:   newRuleType(withBasicStructure, withDataSources),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withSuccessfulGetDataSourcesByName(1), withSuccessfulDeleteRuleTypeDataSource, withSuccessfulAddRuleTypeDataSourceReference),
			TestMethod: update,
		},
		{
			Name:          "CreateRuleType with Data Sources not found",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "data source not found",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withNotFoundGetDataSourcesByName),
			TestMethod:    create,
		},
		{
			Name:          "UpdateRuleType with Data Sources not found",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "data source not found",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withNotFoundGetDataSourcesByName),
			TestMethod:    update,
		},
		{
			Name:          "CreateRuleType with Data Sources failed get",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "failed getting data sources",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withFailedGetDataSourcesByName),
			TestMethod:    create,
		},
		{
			Name:          "UpdateRuleType with Data Sources failed get",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "failed getting data sources",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withFailedGetDataSourcesByName),
			TestMethod:    update,
		},
		{
			Name:          "CreateRuleType with Data Sources failed delete",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "error deleting references to data source",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withSuccessfulGetDataSourcesByName(1), withFailedDeleteRuleTypeDataSource),
			TestMethod:    create,
		},
		{
			Name:          "UpdateRuleType with Data Sources failed delete",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "error adding references to data source",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withSuccessfulGetDataSourcesByName(1), withSuccessfulDeleteRuleTypeDataSource, withFailedAddRuleTypeDataSourceReference),
			TestMethod:    update,
		},
		{
			Name:          "CreateRuleType with Data Sources failed add",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "error adding references to data source",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withSuccessfulGetDataSourcesByName(1), withSuccessfulDeleteRuleTypeDataSource, withFailedAddRuleTypeDataSourceReference),
			TestMethod:    create,
		},
		{
			Name:          "UpdateRuleType with Data Sources failed add",
			RuleType:      newRuleType(withBasicStructure, withDataSources),
			ExpectedError: "error adding references to data source",
			DBSetup:       dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withSuccessfulGetDataSourcesByName(1), withSuccessfulDeleteRuleTypeDataSource, withFailedAddRuleTypeDataSourceReference),
			TestMethod:    update,
		},
		{
			Name:       "CreateRuleType with Data Sources multiple adds",
			RuleType:   newRuleType(withBasicStructure, withDataSources, withDataSources),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withNotFoundGet, withSuccessfulCreate, withSuccessfulGetDataSourcesByName(2), withSuccessfulDeleteRuleTypeDataSource, withSuccessfulAddRuleTypeDataSourceReference),
			TestMethod: create,
		},
		{
			Name:       "UpdateRuleType with Data Sources multiple add",
			RuleType:   newRuleType(withBasicStructure, withDataSources, withDataSources),
			DBSetup:    dbf.NewDBMock(withHierarchyGet, withSuccessfulGet, withSuccessfulUpdate, withSuccessfulGetDataSourcesByName(2), withSuccessfulDeleteRuleTypeDataSource, withSuccessfulAddRuleTypeDataSourceReference),
			TestMethod: update,
		},
	}

	for _, scenario := range scenarios {
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
			svc := ruletypes.NewRuleTypeService()
			switch scenario.TestMethod {
			case create:
				res, err = svc.CreateRuleType(
					ctx,
					projectID,
					scenario.SubscriptionID,
					scenario.RuleType,
					store,
				)
			case update:
				res, err = svc.UpdateRuleType(
					ctx,
					projectID,
					scenario.SubscriptionID,
					scenario.RuleType,
					store,
				)
			case upsert:
				err = svc.UpsertRuleType(
					ctx,
					projectID,
					scenario.SubscriptionID,
					scenario.RuleType,
					store,
				)
			default:
				t.Fatal("unexpected method value")
			}

			if scenario.ExpectedError == "" {
				// due to the presence of autogenerated UUIDs and timestamps,
				// limit our assertions to the subset we deliberately set
				require.NoError(t, err)
				if scenario.TestMethod != upsert {
					require.Equal(t, scenario.RuleType.Id, res.Id)
					require.Equal(t, scenario.RuleType.Description, res.Description)
					require.Equal(t, scenario.RuleType.Name, res.Name)
					// By default this should be the name
					require.Equal(t, scenario.RuleType.Name, res.DisplayName)
					require.Equal(t, scenario.RuleType.Severity.Value, res.Severity.Value)
					require.Equal(t, scenario.RuleType.ShortFailureMessage, res.ShortFailureMessage)
				}
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
	upsert
	ruleName            = "rule_type"
	namespacedRuleName  = "namespace/rule_type"
	description         = "this is my awesome rule"
	shortFailureMessage = "Custom failure message"
)

var (
	ruleTypeID                          = uuid.New()
	projectID                           = uuid.New()
	subscriptionID                      = uuid.New()
	errDefault                          = errors.New("oh no")
	oldRuleType                         = newDBRuleType("low", uuid.Nil, "")
	namespacedOldRuleType               = newDBRuleType("low", subscriptionID, "")
	expectation                         = newDBRuleType("high", uuid.Nil, "")
	namespacedExpectation               = newDBRuleType("high", subscriptionID, "")
	expectationEvaluationFailureMessage = newDBRuleType("high", uuid.Nil, shortFailureMessage)
	incompatibleSchema                  = &structpb.Struct{
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

func withDataSources(ruleType *pb.RuleType) {
	datasources := []*pb.DataSourceReference{
		{
			// We just need a random string
			Name: fmt.Sprintf("foo-%s", uuid.New().String()),
		},
	}
	ruleType.Def.Eval.DataSources = datasources
}

func withEvaluationFailureMessage(message string) func(ruleType *pb.RuleType) {
	return func(ruleType *pb.RuleType) {
		ruleType.ShortFailureMessage = message
	}
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

func withHierarchyGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetParentProjects(gomock.Any(), gomock.Any()).
		Return([]uuid.UUID{uuid.New()}, nil)
}

func withSuccessfulGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(oldRuleType, nil)
}

func withSuccessfulNamespaceGet(mock dbf.DBMock) {
	mock.EXPECT().
		GetRuleTypeByName(gomock.Any(), gomock.Any()).
		Return(namespacedOldRuleType, nil).
		MaxTimes(2)
}

func withSuccessfulCreateWithEvaluationFailureMessage(mock dbf.DBMock) {
	mock.EXPECT().
		CreateRuleType(gomock.Any(), gomock.Any()).
		Return(expectationEvaluationFailureMessage, nil)
}

func withSuccessfulUpdateWithEvaluationFailureMessage(mock dbf.DBMock) {
	mock.EXPECT().
		UpdateRuleType(gomock.Any(), gomock.Any()).
		Return(expectationEvaluationFailureMessage, nil)
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

func withSuccessfulNamespaceCreate(mock dbf.DBMock) {
	mock.EXPECT().
		CreateRuleType(gomock.Any(), gomock.Any()).
		Return(namespacedExpectation, nil)
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

func withSuccessfulGetDataSourcesByName(times int) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		call := mock.EXPECT().
			GetDataSourceByName(gomock.Any(), gomock.Any())
		for i := 0; i < times; i++ {
			call = call.Return(db.DataSource{Name: fmt.Sprintf("foo-%d", i)}, nil)
		}
	}
}

func withNotFoundGetDataSourcesByName(mock dbf.DBMock) {
	mock.EXPECT().
		GetDataSourceByName(gomock.Any(), gomock.Any()).
		Return(db.DataSource{}, sql.ErrNoRows)
}

func withFailedGetDataSourcesByName(mock dbf.DBMock) {
	mock.EXPECT().
		GetDataSourceByName(gomock.Any(), gomock.Any()).
		Return(db.DataSource{}, errDefault)
}

func withSuccessfulDeleteRuleTypeDataSource(mock dbf.DBMock) {
	mock.EXPECT().
		DeleteRuleTypeDataSource(gomock.Any(), gomock.Any()).
		Return(nil)
}

func withFailedDeleteRuleTypeDataSource(mock dbf.DBMock) {
	mock.EXPECT().
		DeleteRuleTypeDataSource(gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func withSuccessfulAddRuleTypeDataSourceReference(mock dbf.DBMock) {
	mock.EXPECT().
		AddRuleTypeDataSourceReference(gomock.Any(), gomock.Any()).
		Return(db.RuleTypeDataSource{}, nil)
}

func withFailedAddRuleTypeDataSourceReference(mock dbf.DBMock) {
	mock.EXPECT().
		AddRuleTypeDataSourceReference(gomock.Any(), gomock.Any()).
		Return(db.RuleTypeDataSource{}, errDefault)
}

func newDBRuleType(severity db.Severity, subscriptionID uuid.UUID, failureMessage string) db.RuleType {
	name := ruleName
	if subscriptionID != uuid.Nil {
		name = namespacedRuleName
	}
	if failureMessage == "" {
		failureMessage = fmt.Sprintf("Rule %s evaluation failed", name)
	}
	return db.RuleType{
		ID:                  ruleTypeID,
		Name:                name,
		Definition:          []byte(`{}`),
		SeverityValue:       severity,
		Description:         description,
		SubscriptionID:      uuid.NullUUID{Valid: subscriptionID != uuid.Nil, UUID: subscriptionID},
		ShortFailureMessage: failureMessage,
	}
}
