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

// Package rtengine contains the rule type engine
package rtengine

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/entities"
	enginerr "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval"
	"github.com/stacklok/minder/internal/engine/ingestcache"
	"github.com/stacklok/minder/internal/engine/ingester"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/profiles"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// RuleMeta is the metadata for a rule
// TODO: We probably should care about a version
type RuleMeta struct {
	// Name is the name of the rule
	Name string
	// Organization is the ID of the organization that this rule is for
	Organization *string
	// Project is the ID of the project that this rule is for
	Project *string
}

// String returns a string representation of the rule meta
func (r *RuleMeta) String() string {
	if r.Project != nil {
		return fmt.Sprintf("group/%s/%s", *r.Project, r.Name)
	}
	return fmt.Sprintf("org/%s/%s", *r.Organization, r.Name)
}

// RuleTypeEngine is the engine for a rule type. It builds the multiple
// sections of the rule type and instantiates the needed drivers for
// them.
type RuleTypeEngine struct {
	Meta RuleMeta

	// ingester is the rule data ingest engine
	ingester engif.Ingester

	// ruleEvaluator is the rule evaluator
	ruleEvaluator engif.Evaluator

	ruleValidator *profiles.RuleValidator

	ruletype *minderv1.RuleType

	ingestCache ingestcache.Cache
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider provinfv1.Provider,
) (*RuleTypeEngine, error) {
	rval, err := profiles.NewRuleValidator(ruletype)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
	}

	ingest, err := ingester.NewRuleDataIngest(ruletype, provider)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule data ingest: %w", err)
	}

	evaluator, err := eval.NewRuleEvaluator(ctx, ruletype, provider)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule evaluator: %w", err)
	}

	rte := &RuleTypeEngine{
		Meta: RuleMeta{
			Name: ruletype.Name,
		},
		ruleValidator: rval,
		ingester:      ingest,
		ruleEvaluator: evaluator,
		ruletype:      ruletype,
		ingestCache:   ingestcache.NewNoopCache(),
	}

	if ruletype.Context.Project != nil && *ruletype.Context.Project != "" {
		prj := strings.Clone(*ruletype.Context.Project)
		rte.Meta.Project = &prj
	} else {
		return nil, fmt.Errorf("rule type context must have a project")
	}

	return rte, nil
}

// WithIngesterCache sets the ingester cache for the rule type engine
func (r *RuleTypeEngine) WithIngesterCache(ingestCache ingestcache.Cache) *RuleTypeEngine {
	r.ingestCache = ingestCache
	return r
}

// GetID returns the ID of the rule type. The ID is meant to be
// a serializable unique identifier for the rule type.
func (r *RuleTypeEngine) GetID() string {
	return r.Meta.String()
}

// GetRuleInstanceValidator returns the rule instance validator for this rule type.
// By instance we mean a rule that has been instantiated in a profile from a given rule type.
func (r *RuleTypeEngine) GetRuleInstanceValidator() *profiles.RuleValidator {
	return r.ruleValidator
}

// GetRuleType returns the rule type PB structure.
func (r *RuleTypeEngine) GetRuleType() *minderv1.RuleType {
	return r.ruletype
}

// Eval runs the rule type engine against the given entity
func (r *RuleTypeEngine) Eval(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	params engif.EvalParamsReadWriter,
) (finalErr error) {
	defer func() {
		if r := recover(); r != nil {
			zerolog.Ctx(ctx).Error().Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in rule type engine")
			finalErr = enginerr.ErrInternal
		}
	}()

	logger := zerolog.Ctx(ctx).With().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String()).Logger()

	logger.Info().Msg("entity evaluation - ingest started")
	// Try looking at the ingesting cache first
	result, ok := r.ingestCache.Get(r.ingester, inf.Entity, params.GetRule().Params)
	if !ok {
		var err error
		// Ingest the data needed for the rule evaluation
		result, err = r.ingester.Ingest(ctx, inf.Entity, params.GetRule().Params)
		if err != nil {
			// Ingesting failed, so we can't evaluate the rule.
			// Note that for some types of ingesting the evalErr can already be set from the ingester.
			return fmt.Errorf("error ingesting data: %w", err)
		}
		r.ingestCache.Set(r.ingester, inf.Entity, params.GetRule().Params, result)
	} else {
		logger.Info().Str("id", r.GetID()).Msg("entity evaluation - ingest using cache")
	}
	logger.Info().Msg("entity evaluation - ingest completed")
	params.SetIngestResult(result)

	// Process evaluation
	logger.Info().Msg("entity evaluation - evaluation started")
	err := r.ruleEvaluator.Eval(ctx, params.GetRule().Def, result)
	logger.Info().Msg("entity evaluation - evaluation completed")
	return err
}

// GetRulesFromProfileOfType returns the rules from the profile of the given type
func GetRulesFromProfileOfType(p *minderv1.Profile, rt *minderv1.RuleType) ([]*minderv1.Profile_Rule, error) {
	contextualRules, err := profiles.GetRulesForEntity(p, minderv1.EntityFromString(rt.Def.InEntity))
	if err != nil {
		return nil, fmt.Errorf("error getting rules for entity: %w", err)
	}

	rules := []*minderv1.Profile_Rule{}
	err = profiles.TraverseRules(contextualRules, func(r *minderv1.Profile_Rule) error {
		if r.Type == rt.Name {
			rules = append(rules, r)
		}
		return nil
	})

	// This shouldn't happen
	if err != nil {
		return nil, fmt.Errorf("error traversing rules: %w", err)
	}

	return rules, nil
}
