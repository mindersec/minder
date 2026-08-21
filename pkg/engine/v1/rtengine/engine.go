// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rtengine contains the rule type engine
package rtengine

import (
	"context"
	"fmt"
	"runtime/debug"
	"slices"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/eval"
	"github.com/mindersec/minder/internal/engine/ingestcache"
	"github.com/mindersec/minder/internal/engine/ingester"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	enginerr "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles"
)

// RuleMeta is the metadata for a rule
// TODO: We probably should care about a version
type RuleMeta struct {
	// Name is the name of the rule
	Name string
	// Project is the ID of the project that this rule is for
	Project string
}

// String returns a string representation of the rule meta
func (r *RuleMeta) String() string {
	return fmt.Sprintf("group/%s/%s", r.Project, r.Name)
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

	// supportedByProvider records whether the provider this engine was
	// built with implements every trait the rule type declares in
	// provider_traits. Computed once at construction time. This is false
	// both when a declared trait name is unknown (see
	// unknownProviderTraits) and when every trait name is known but the
	// provider doesn't implement one of them.
	supportedByProvider bool

	// unknownProviderTraits lists every entry in provider_traits that
	// doesn't match a known ProviderType name, e.g. a typo or a trait
	// renamed since the rule type was stored. Distinct from a known trait
	// the provider simply doesn't implement: unlike that case, this is a
	// rule type authoring problem the user can fix, so callers surface it
	// as an evaluation error rather than silently skipping the rule.
	unknownProviderTraits []string
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider interfaces.Provider,
	opts ...interfaces.Option,
) (*RuleTypeEngine, error) {
	if ruletype.Context.GetProject() == "" {
		return nil, fmt.Errorf("rule type context must have a project")
	}

	supportedByProvider := true
	var unknownProviderTraits []string
	for _, trait := range ruletype.GetDef().GetProviderTraits() {
		providerTrait, ok := minderv1.ProviderTypeFromString(trait)
		if !ok {
			// A trait name that doesn't map to any known ProviderType is
			// almost certainly a typo, or a trait renamed since the rule
			// type was stored. Rule type creation validates against this
			// (see ruletypes.validateProviderTraits), but rule types
			// already stored before that validation existed, or any path
			// that bypasses it, would otherwise fail this exact same way
			// with no signal anywhere that anything is wrong. Collect
			// every unknown name rather than stopping at the first, so
			// the caller can report them all at once: unlike a known
			// trait the provider doesn't implement, this is a rule type
			// authoring problem the user can fix.
			unknownProviderTraits = append(unknownProviderTraits, trait)
			supportedByProvider = false
			continue
		}
		if !provider.CanImplement(providerTrait) {
			supportedByProvider = false
		}
	}

	rval, err := profiles.NewRuleValidator(ruletype)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
	}

	ingest, err := ingester.NewRuleDataIngest(ruletype, provider)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule data ingest: %w", err)
	}

	evaluator, err := eval.NewRuleEvaluator(ctx, ruletype, provider, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule evaluator: %w", err)
	}

	rte := &RuleTypeEngine{
		Meta: RuleMeta{
			Name:    ruletype.Name,
			Project: ruletype.GetContext().GetProject(),
		},
		ruleValidator: rval,
		ingester:      ingest,
		ruleEvaluator: evaluator,
		ruletype:      ruletype,
		ingestCache:   ingestcache.NewNoopCache(),

		supportedByProvider:   supportedByProvider,
		unknownProviderTraits: unknownProviderTraits,
	}

	return rte, nil
}

// SupportedByProvider reports whether the provider this engine was built
// with implements every trait the rule type declares in provider_traits.
func (r *RuleTypeEngine) SupportedByProvider() bool {
	return r.supportedByProvider
}

// UnknownProviderTraits returns the entries in provider_traits that don't
// match any known ProviderType name, e.g. a typo or a trait renamed since
// the rule type was stored. Empty if every declared trait name is known,
// regardless of whether the provider implements it — see
// SupportedByProvider for that.
func (r *RuleTypeEngine) UnknownProviderTraits() []string {
	return slices.Clone(r.unknownProviderTraits)
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
	// Eval should never exit the entire process, so recover any panics within the rule evaluation engine.
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
	ingestData, ok := r.ingestCache.Get(r.ingester, entity, ruleParams)
	if !ok {
		var err error
		// Ingest the data needed for the rule evaluation
		ingestData, err = r.ingester.Ingest(ctx, entity, ruleParams)
		if err != nil {
			// Ingesting failed, so we can't evaluate the rule.
			// Note that for some types of ingesting the evalErr can already be set from the ingester.
			return nil, fmt.Errorf("error ingesting data: %w", err)
		}
		r.ingestCache.Set(r.ingester, entity, ruleParams, ingestData)
	} else {
		logger.Info().Str("id", r.GetID()).Msg("entity evaluation - ingest using cache")
	}
	logger.Info().Msg("entity evaluation - ingest completed")
	params.SetIngestResult(ingestData)

	// Process evaluation
	logger.Info().Msg("entity evaluation - evaluation started")
	res, err := r.ruleEvaluator.Eval(ctx, ruleDef, entity, ingestData)
	logger.Info().Msg("entity evaluation - evaluation completed")
	return res, err
}

// WithCustomIngester sets a custom ingester for the rule type engine. This is handy for testing
// but should not be used in production.
func (r *RuleTypeEngine) WithCustomIngester(ing interfaces.Ingester) *RuleTypeEngine {
	r.ingester = ing
	return r
}

// NewRuleEvaluator creates an Evaluator from the specified RuleType.
// The external caller is responsible for populating the ingested data in
// the Evaluator's Eval() method; the provider is used only for certain
// PR-based file content checks (trusty, vulncheck, and homoglyphs).
// Unlike NewRuleTypeEngine, ingestion data is not cached within the library.
func NewRuleEvaluator(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider interfaces.Provider,
	opts ...interfaces.Option,
) (interfaces.Evaluator, error) {
	return eval.NewRuleEvaluator(ctx, ruletype, provider, opts...)
}
