// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package application

import (
	"context"
	"fmt"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
)

// InvisibleCharactersEvaluator is an evaluator for the invisible characters rule type
type InvisibleCharactersEvaluator struct {
	processor     domain.HomoglyphProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewInvisibleCharactersEvaluator creates a new invisible characters evaluator
func NewInvisibleCharactersEvaluator(provider provinfv1.Provider) (*InvisibleCharactersEvaluator, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	ghClient, err := provinfv1.As[provinfv1.GitHub](provider)
	if err != nil {
		return nil, fmt.Errorf("could not fetch GitHub client: %w", err)
	}

	return &InvisibleCharactersEvaluator{
		processor:     domain.NewInvisibleCharactersProcessor(),
		reviewHandler: communication.NewGhReviewPrHandler(ghClient),
	}, nil
}

// Eval evaluates the invisible characters rule type
func (ice *InvisibleCharactersEvaluator) Eval(ctx context.Context, _ map[string]any, res *engif.Result) error {
	hasFoundViolations, err := evaluateHomoglyphs(ctx, ice.processor, res, ice.reviewHandler)
	if err != nil {
		return err
	}

	if hasFoundViolations {
		return evalerrors.NewErrEvaluationFailed("found invisible characters violations")
	}

	return nil
}
