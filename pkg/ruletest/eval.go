// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/engine/v1/rtengine"
	"github.com/mindersec/minder/pkg/fileconvert"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

func (tr *testCaseRunner) builtinEval(
	thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var ruleNameOrPath string
	var entityDict *starlark.Dict
	var profileDict *starlark.Dict
	var mockHttpDict *starlark.Dict

	err := starlark.UnpackArgs("eval", args, kwargs,
		"rule", &ruleNameOrPath, "entity?", &entityDict, "profile?", &profileDict, "mock_http?", &mockHttpDict)
	if err != nil {
		return nil, err
	}

	var rt *minderv1.RuleType

	if tr.ruleTypes != nil {
		if ruleType, ok := tr.ruleTypes[ruleNameOrPath]; ok {
			rt = ruleType
		}
	}

	if rt == nil {
		rulePath := ruleNameOrPath
		if !filepath.IsAbs(rulePath) {
			callerFrame := thread.CallFrame(1)
			if callerFile := callerFrame.Pos.Filename(); callerFile != "" {
				rulePath = filepath.Join(filepath.Dir(callerFile), rulePath)
			}
		}

		decoder, closer := fileconvert.DecoderForFile(rulePath)
		if decoder == nil {
			return nil, fmt.Errorf("error opening file: %s (or rule not found in loaded rule types)", rulePath)
		}
		defer closer.Close()

		rt, err = fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
		if err != nil {
			return nil, fmt.Errorf("failed to parse rule type: %w", err)
		}
	}

	profileMap, err := dictToGoMap(profileDict)
	if err != nil {
		return nil, fmt.Errorf("invalid profile argument: %w", err)
	}

	entityMap, err := dictToGoMap(entityDict)
	if err != nil {
		return nil, fmt.Errorf("invalid entity argument: %w", err)
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

	tk := tkv1.NewTestKit(tkv1.WithHandlerFunc(mockHandler.ServeHTTP))

	rte, err := rtengine.NewRuleTypeEngine(ctx, rt, tk)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rule type engine: %w", err)
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

//nolint:gocyclo // this is a simple switch over many entity types
func mapToProto(entityType string, entityMap map[string]any) (proto.Message, error) {
	if len(entityMap) == 0 {
		return nil, nil
	}

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
		// Some entities like PullRequest or BuildEnvironment may not have concrete protobuf structs available here.
		// For mocking purposes, returning nil is acceptable if the template doesn't strict check them,
		// but returning an error is safer to flag unsupported mocking right now.
		return nil, fmt.Errorf("unsupported entity type for mapping to proto: %s", entityType)
	}

	if err := unmarshaller.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	return msg, nil
}
