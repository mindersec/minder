// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/domain"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// MixedScriptsEvaluator is the evaluator for the mixed scripts rule type
type MixedScriptsEvaluator struct {
	processor     domain.HomoglyphProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewMixedScriptEvaluator creates a new mixed scripts evaluator
func NewMixedScriptEvaluator(
	ctx context.Context,
	ghClient interfaces.GitHubIssuePRClient,
	opts ...interfaces.Option,
) (*MixedScriptsEvaluator, error) {
	msProcessor, err := domain.NewMixedScriptsProcessor(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create mixed scripts processor: %w", err)
	}

	evaluator := &MixedScriptsEvaluator{
		processor:     msProcessor,
		reviewHandler: communication.NewGhReviewPrHandler(ghClient),
	}

	for _, opt := range opts {
		if err := opt(evaluator); err != nil {
			return nil, err
		}
	}

	return evaluator, nil
}

// Eval evaluates the mixed scripts rule type
func (mse *MixedScriptsEvaluator) Eval(
	ctx context.Context,
	_ map[string]any,
	_ protoreflect.ProtoMessage,
	res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	violations, err := evaluateHomoglyphs(ctx, mse.processor, res, mse.reviewHandler)
	if err != nil {
		return nil, err
	}

	if len(violations) > 0 {
		return nil, evalerrors.NewDetailedErrEvaluationFailed(
			templates.MixedScriptsTemplate,
			map[string]any{"violations": violations},
			"found mixed scripts violations",
		)
	}

	return &interfaces.EvaluationResult{}, nil
}
