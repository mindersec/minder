// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rego provides the rego rule evaluator
package rego

import (
	"context"
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown/print"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	eoptions "github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/flags"
)

const (
	// RegoEvalType is the type of the rego evaluator
	RegoEvalType = "rego"
	// MinderRegoFile is the default rego file for minder.
	MinderRegoFile = "minder.rego"
	// RegoQueryPrefix is the prefix for rego queries
	RegoQueryPrefix = "data.minder"
)

const (
	// EnablePrintEnvVar is the environment variable to enable print statements
	EnablePrintEnvVar = "REGO_ENABLE_PRINT"
)

// Evaluator is the evaluator for rego rules
// It initializes the rego engine and evaluates the rules
// The default rego package is "minder"
type Evaluator struct {
	cfg          *Config
	featureFlags flags.Interface
	regoOpts     []func(*rego.Rego)
	reseval      resultEvaluator
	datasources  *v1datasources.DataSourceRegistry
}

// Input is the input for the rego evaluator
type Input struct {
	// Profile is the values set for the profile
	Profile map[string]any `json:"profile"`
	// Ingested is the values set for the ingested data
	Ingested any `json:"ingested"`
	// Properties contains the entity's properties as defined by
	// the provider
	Properties map[string]any `json:"properties"`
	// OutputFormat is the format to output violations in
	OutputFormat EvalOutputFormat `json:"output_format"`
}

type hook struct {
}

func (*hook) Print(_ print.Context, msg string) error {
	fmt.Println(msg)
	return nil
}

var _ print.Hook = (*hook)(nil)

// NewRegoEvaluator creates a new rego evaluator
func NewRegoEvaluator(
	cfg *minderv1.RuleType_Definition_Eval_Rego,
	opts ...interfaces.Option,
) (*Evaluator, error) {
	c, err := parseConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse rego config: %w", err)
	}

	re := c.getEvalType()

	eval := &Evaluator{
		cfg:     c,
		reseval: re,
		regoOpts: []func(*rego.Rego){
			rego.Query(RegoQueryPrefix),
			rego.Module(MinderRegoFile, c.Def),
			rego.Strict(true),
		},
	}

	for _, opt := range opts {
		if err := opt(eval); err != nil {
			return nil, err
		}
	}

	if os.Getenv(EnablePrintEnvVar) == "true" {
		h := &hook{}
		eval.regoOpts = append(eval.regoOpts,
			rego.EnablePrintStatements(true),
			rego.PrintHook(h),
		)
	}

	return eval, nil
}

var _ eoptions.SupportsFlags = (*Evaluator)(nil)

func (e *Evaluator) newRegoFromOptions(opts ...func(*rego.Rego)) *rego.Rego {
	return rego.New(append(e.regoOpts, opts...)...)
}

// SetFlagsClient implements the SupportsFlags interface.
func (e *Evaluator) SetFlagsClient(client flags.Interface) error {
	e.featureFlags = client
	return nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(
	ctx context.Context, pol map[string]any, entity protoreflect.ProtoMessage, res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	// The rego engine is actually able to handle nil
	// objects quite gracefully, so we don't need to check
	// this explicitly.
	obj := res.Object

	// Register options to expose functions
	regoFuncOptions := []func(*rego.Rego){
		// TODO: figure out a Rego V1 migration path (https://github.com/mindersec/minder/issues/5262)
		rego.SetRegoVersion(ast.RegoV0),
	}

	// Initialize the built-in minder library rego functions
	regoFuncOptions = append(regoFuncOptions, instantiateRegoLib(ctx, e.featureFlags, res)...)

	// If the evaluator has data sources defined, expose their functions
	regoFuncOptions = append(regoFuncOptions, buildDataSourceOptions(res, e.datasources)...)

	// Create the rego object
	r := e.newRegoFromOptions(
		regoFuncOptions...,
	)

	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not prepare Rego: %w", err)
	}

	input := &Input{
		Profile:      pol,
		Ingested:     obj,
		OutputFormat: e.cfg.ViolationFormat,
	}

	enrichInputWithEntityProps(input, entity)
	rs, err := pq.Eval(ctx, rego.EvalInput(input), rego.EvalHTTPRoundTripper(LimitedDialer))
	if err != nil {
		return nil, fmt.Errorf("error evaluating profile. Might be wrong input: %w", err)
	}

	return e.reseval.parseResult(rs, entity)
}

type propertiesFetcher interface {
	GetProperties() *structpb.Struct
}

func enrichInputWithEntityProps(
	input *Input,
	entity protoreflect.ProtoMessage,
) {
	if inner, ok := entity.(propertiesFetcher); ok {
		input.Properties = inner.GetProperties().AsMap()
	}
}
