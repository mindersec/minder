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

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// MixedScriptsEvaluator is the evaluator for the mixed scripts rule type
type MixedScriptsEvaluator struct {
	processor     domain.HomoglyphProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewMixedScriptEvaluator creates a new mixed scripts evaluator
func NewMixedScriptEvaluator(ghClient provifv1.GitHub) (*MixedScriptsEvaluator, error) {
	msProcessor, err := domain.NewMixedScriptsProcessor()
	if err != nil {
		return nil, fmt.Errorf("could not create mixed scripts processor: %w", err)
	}

	return &MixedScriptsEvaluator{
		processor:     msProcessor,
		reviewHandler: communication.NewGhReviewPrHandler(ghClient),
	}, nil
}

// Eval evaluates the mixed scripts rule type
func (mse *MixedScriptsEvaluator) Eval(ctx context.Context, _ map[string]any, res *engif.Result) error {
	hasFoundViolations, err := evaluateHomoglyphs(ctx, mse.processor, res, mse.reviewHandler)
	if err != nil {
		return err
	}

	if hasFoundViolations {
		return evalerrors.NewErrEvaluationFailed("found mixed scripts violations")
	}

	return nil
}
