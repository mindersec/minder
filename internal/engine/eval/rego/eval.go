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
// Package rule provides the CLI subcommand for managing rules

// Package rego provides the rego rule evaluator
package rego

import (
	"context"
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/topdown/print"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
	eoptions "github.com/stacklok/minder/internal/engine/options"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	cfg      *Config
	regoOpts []func(*rego.Rego)
	reseval  resultEvaluator
}

// Input is the input for the rego evaluator
type Input struct {
	// Profile is the values set for the profile
	Profile map[string]any `json:"profile"`
	// Ingested is the values set for the ingested data
	Ingested any `json:"ingested"`
	// OutputFormat is the format to output violations in
	OutputFormat ConstraintsViolationsFormat `json:"output_format"`
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
	opts ...eoptions.Option,
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
			re.getQuery(),
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

func (e *Evaluator) newRegoFromOptions(opts ...func(*rego.Rego)) *rego.Rego {
	return rego.New(append(e.regoOpts, opts...)...)
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(ctx context.Context, pol map[string]any, res *engif.Result) error {
	// The rego engine is actually able to handle nil
	// objects quite gracefully, so we don't need to check
	// this explicitly.
	obj := res.Object

	libFuncs := instantiateRegoLib(res)
	r := e.newRegoFromOptions(
		libFuncs...,
	)
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("could not prepare Rego: %w", err)
	}

	rs, err := pq.Eval(ctx, rego.EvalInput(&Input{
		Profile:      pol,
		Ingested:     obj,
		OutputFormat: e.cfg.ViolationFormat,
	}))
	if err != nil {
		return fmt.Errorf("error evaluating profile. Might be wrong input: %w", err)
	}

	return e.reseval.parseResult(rs)
}
