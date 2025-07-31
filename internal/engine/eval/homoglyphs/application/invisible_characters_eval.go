// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"

	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/domain"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// InvisibleCharactersEvaluator is an evaluator for the invisible characters rule type
type InvisibleCharactersEvaluator struct {
	processor     domain.HomoglyphProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewInvisibleCharactersEvaluator creates a new invisible characters evaluator
func NewInvisibleCharactersEvaluator(
	_ context.Context,
	ghClient interfaces.GitHubIssuePRClient,
	opts ...interfaces.Option,
) (*InvisibleCharactersEvaluator, error) {
	evaluator := &InvisibleCharactersEvaluator{
		processor:     domain.NewInvisibleCharactersProcessor(),
		reviewHandler: communication.NewGhReviewPrHandler(ghClient),
	}

	for _, opt := range opts {
		if err := opt(evaluator); err != nil {
			return nil, err
		}
	}

	return evaluator, nil
}

// Eval evaluates the invisible characters rule type
func (ice *InvisibleCharactersEvaluator) Eval(
	ctx context.Context,
	_ map[string]any,
	_ protoreflect.ProtoMessage,
	res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	violations, err := evaluateHomoglyphs(ctx, ice.processor, res, ice.reviewHandler)
	if err != nil {
		return nil, err
	}

	if len(violations) > 0 {
		return nil, evalerrors.NewDetailedErrEvaluationFailed(
			templates.InvisibleCharactersTemplate,
			map[string]any{"violations": violations},
			"found invisible characters violations",
		)
	}

	return &interfaces.EvaluationResult{}, nil
}
