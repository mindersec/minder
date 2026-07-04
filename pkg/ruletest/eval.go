// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	eoptions "github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/engine/v1/rtengine"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

func (tr *testCaseRunner) builtinEval(
	_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var ruleName string
	var entityDict *starlark.Dict
	var profileDict *starlark.Dict
	var mockHttpDict *starlark.Dict
	var mockFSDict *starlark.Dict
	var datasourcesDict *starlark.Dict

	err := starlark.UnpackArgs("eval", args, kwargs,
		"rule", &ruleName, "entity?", &entityDict, "profile?", &profileDict, "mock_http?", &mockHttpDict, "mock_fs?", &mockFSDict, "data_sources?", &datasourcesDict)
	if err != nil {
		return nil, err
	}

	mockFSMap, err := parseMockFSDict(mockFSDict)
	if err != nil {
		return nil, err
	}

	rt, err := tr.lookupRuleType(ruleName)
	if err != nil {
		return nil, err
	}

	profileMap, err := dictToGoMap(profileDict)
	if err != nil {
		return nil, fmt.Errorf("invalid profile argument: %w", err)
	}

	entityMap, err := dictToGoMap(entityDict)
	if err != nil {
		return nil, fmt.Errorf("invalid entity argument: %w", err)
	}

	dsRegistry, err := buildMockDataSourceRegistry(datasourcesDict)
	if err != nil {
		return nil, fmt.Errorf("invalid data_sources argument: %w", err)
	}

	entityProto, err := mapToProto(rt.Def.InEntity, entityMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity map to proto: %w", err)
	}

	mockHandler, err := buildMockHTTPHandler(mockHttpDict)
	if err != nil {
		return nil, fmt.Errorf("invalid mock_http configuration: %w", err)
	}

	ctx := context.Background()

	tkOpts := []tkv1.Option{tkv1.WithHandlerFunc(mockHandler.ServeHTTP)}
	if len(mockFSMap) > 0 {
		tkOpts = append(tkOpts, tkv1.WithGitFiles(mockFSMap))
	}
	tk := tkv1.NewTestKit(tkOpts...)

	rte, err := rtengine.NewRuleTypeEngine(ctx, rt, tk, eoptions.WithDataSources(dsRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rule type engine: %w", err)
	}

	if tk.ShouldOverrideIngest() {
		rte.WithCustomIngester(tk)
	}

	res, err := rte.Eval(ctx, entityProto, profileMap, nil, &stubResultSink{})

	// Because Eval returns the error, we pass that error to formatEvalResult
	// We ignore res for now as we just want the error
	_ = res
	return formatEvalResult(err), nil
}

type stubResultSink struct{}

func (*stubResultSink) SetIngestResult(*interfaces.Ingested) {}

func formatEvalResult(evalErr error) *starlark.Dict {
	result := starlark.NewDict(2)
	status, msg := "", ""

	switch {
	case evalErr == nil:
		status = "pass"
	case errors.Is(evalErr, interfaces.ErrEvaluationFailed):
		status = "fail"
		msg = evalErr.Error()
		var details interfaces.EvalError
		if errors.As(evalErr, &details) {
			msg = fmt.Sprintf("%s: %s", msg, details.Details())
		}
	case errors.Is(evalErr, interfaces.ErrEvaluationSkipped):
		status = "skip"
		msg = evalErr.Error()
	default:
		status = "error"
		msg = evalErr.Error()
	}

	_ = result.SetKey(starlark.String("status"), starlark.String(status))
	_ = result.SetKey(starlark.String("message"), starlark.String(msg))
	return result
}

func parseMockFSDict(mockFSDict *starlark.Dict) (map[string]string, error) {
	mockFSMap := make(map[string]string)
	if mockFSDict != nil {
		for _, item := range mockFSDict.Items() {
			k, v := item[0], item[1]
			ks, ok1 := k.(starlark.String)
			vs, ok2 := v.(starlark.String)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("mock_fs keys and values must be strings")
			}
			mockFSMap[string(ks)] = string(vs)
		}
	}
	return mockFSMap, nil
}

func (tr *testCaseRunner) lookupRuleType(ruleName string) (*minderv1.RuleType, error) {
	if tr.ruleTypes != nil {
		if ruleType := tr.ruleTypes[ruleName]; ruleType != nil {
			return ruleType, nil
		}
	}
	return nil, fmt.Errorf("rule %q not found; make sure the rule type YAML is in the same directory as the test file", ruleName)
}

//nolint:gocyclo // this is a simple switch over many entity types
func mapToProto(entityType string, entityMap map[string]any) (proto.Message, error) {
	b, err := json.Marshal(entityMap)
	if err != nil {
		return nil, err
	}

	unmarshaller := protojson.UnmarshalOptions{DiscardUnknown: true}
	entEnum := minderv1.EntityFromString(entityType)

	var msg proto.Message

	switch entEnum {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		msg = &minderv1.Repository{}
	case minderv1.Entity_ENTITY_ARTIFACTS:
		msg = &minderv1.Artifact{}
	case minderv1.Entity_ENTITY_RELEASE:
		msg = &minderv1.Release{}
	case minderv1.Entity_ENTITY_PIPELINE_RUN:
		msg = &minderv1.PipelineRun{}
	case minderv1.Entity_ENTITY_TASK_RUN:
		msg = &minderv1.TaskRun{}
	case minderv1.Entity_ENTITY_BUILD:
		msg = &minderv1.Build{}
	case minderv1.Entity_ENTITY_UNSPECIFIED,
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
		minderv1.Entity_ENTITY_PULL_REQUESTS:
		fallthrough
	default:
		return nil, fmt.Errorf("unsupported entity type for mapping to proto: %s", entityType)
	}

	if err := unmarshaller.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	return msg, nil
}
