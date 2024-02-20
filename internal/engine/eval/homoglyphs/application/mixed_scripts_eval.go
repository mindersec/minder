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

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
)

// MixedScriptsEvaluator is the evaluator for the mixed scripts rule type
type MixedScriptsEvaluator struct {
	processor     domain.HomoglyphProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewMixedScriptEvaluator creates a new mixed scripts evaluator
func NewMixedScriptEvaluator(pbuild *providers.ProviderBuilder) (*MixedScriptsEvaluator, error) {
	if pbuild == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	ghClient, err := pbuild.GetGitHub()
	if err != nil {
		return nil, fmt.Errorf("could not fetch GitHub client: %w", err)
	}

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
	return evaluateHomoglyphs(ctx, mse.processor, res, mse.reviewHandler)
}
