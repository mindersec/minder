// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rtengine contains the rule type engine
package rtengine

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval"
	"github.com/mindersec/minder/internal/engine/ingestcache"
	"github.com/mindersec/minder/internal/engine/ingester"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
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
	ingester interfaces.Ingester

	// ruleEvaluator is the rule evaluator
	ruleEvaluator interfaces.Evaluator

	ruleValidator *profiles.RuleValidator

	ruletype *minderv1.RuleType

	ingestCache ingestcache.Cache
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider provinfv1.Provider,
	experiments openfeature.IClient,
	opts ...eoptions.Option,
) (*RuleTypeEngine, error) {
	rval, err := profiles.NewRuleValidator(ruletype)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
	}

	ingest, err := ingester.NewRuleDataIngest(ruletype, provider)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule data ingest: %w", err)
	}

	evaluator, err := eval.NewRuleEvaluator(ctx, ruletype, provider, experiments, opts...)
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
	entity protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
	params interfaces.ResultSink,
) (res *interfaces.EvaluationResult, finalErr error) {
	logger := zerolog.Ctx(ctx)
	defer func() {
		if r := recover(); r != nil {
			logger.Error().Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in rule type engine")
			finalErr = enginerr.ErrInternal
		}
	}()

	// The rule type has already been validated at creation time. However,
	// re-validating it here is a good idea to ensure that the rule type
	// has not been tampered with. Also, this sets the defaults for the
	// rule definition.
	if ruleDef != nil {
		if err := r.ruleValidator.ValidateRuleDefAgainstSchema(ruleDef); err != nil {
			return nil, fmt.Errorf("rule definition validation failed: %w", err)
		}
	}

	if ruleParams != nil {
		if err := r.ruleValidator.ValidateParamsAgainstSchema(ruleParams); err != nil {
			return nil, fmt.Errorf("rule parameters validation failed: %w", err)
		}
	}

	logger.Info().Msg("entity evaluation - ingest started")
	// Try looking at the ingesting cache first
	result, ok := r.ingestCache.Get(r.ingester, entity, ruleParams)
	if !ok {
		var err error
		// Ingest the data needed for the rule evaluation
		result, err = r.ingester.Ingest(ctx, entity, ruleParams)
		if err != nil {
			// Ingesting failed, so we can't evaluate the rule.
			// Note that for some types of ingesting the evalErr can already be set from the ingester.
			return nil, fmt.Errorf("error ingesting data: %w", err)
		}
		r.ingestCache.Set(r.ingester, entity, ruleParams, result)
	} else {
		logger.Info().Str("id", r.GetID()).Msg("entity evaluation - ingest using cache")
	}
	logger.Info().Msg("entity evaluation - ingest completed")
	params.SetIngestResult(result)

	// Process evaluation
	logger.Info().Msg("entity evaluation - evaluation started")
	res, err := r.ruleEvaluator.Eval(ctx, ruleDef, entity, result)
	logger.Info().Msg("entity evaluation - evaluation completed")
	return res, err
}

// WithCustomIngester sets a custom ingester for the rule type engine. This is handy for testing
// but should not be used in production.
func (r *RuleTypeEngine) WithCustomIngester(ing interfaces.Ingester) *RuleTypeEngine {
	r.ingester = ing
	return r
}
