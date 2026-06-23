// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"go.starlark.net/starlark"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/engine/v1/rtengine"
	"github.com/mindersec/minder/pkg/fileconvert"
)

func (tr *testCaseRunner) builtinEval(
	thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var rulePath string
	var entityDict *starlark.Dict
	var profileDict *starlark.Dict

	err := starlark.UnpackArgs("eval", args, kwargs,
		"rule", &rulePath, "entity?", &entityDict, "profile?", &profileDict)
	if err != nil {
		return nil, err
	}

	if !filepath.IsAbs(rulePath) {
		callerFrame := thread.CallFrame(1)
		if callerFile := callerFrame.Pos.Filename(); callerFile != "" {
			rulePath = filepath.Join(filepath.Dir(callerFile), rulePath)
		}
	}

	decoder, closer := fileconvert.DecoderForFile(rulePath)
	if decoder == nil {
		return nil, fmt.Errorf("error opening file: %s", rulePath)
	}
	defer closer.Close()

	rt, err := fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule type: %w", err)
	}

	profileMap, err := dictToGoMap(profileDict)
	if err != nil {
		return nil, fmt.Errorf("invalid profile argument: %w", err)
	}

	entityMap, err := dictToGoMap(entityDict)
	if err != nil {
		return nil, fmt.Errorf("invalid entity argument: %w", err)
	}

	ctx := context.Background()
	evaluator, err := rtengine.NewRuleEvaluator(ctx, rt, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rule evaluator: %w", err)
	}

	ingested := &interfaces.Ingested{Object: entityMap}
	_, evalErr := evaluator.Eval(ctx, profileMap, nil, ingested)

	return formatEvalResult(evalErr), nil
}

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

	// We ignore the error from SetKey because result is a new Dict we just created
	// and we know it's not frozen, and the keys are valid strings.
	_ = result.SetKey(starlark.String("status"), starlark.String(status))
	_ = result.SetKey(starlark.String("message"), starlark.String(msg))
	return result
}
