// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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
	"bytes"
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	// RegoEvalType is the type of the rego evaluator
	RegoEvalType = "rego"
)

// Evaluator is the evaluator for rego rules
// It initializes the rego engine and evaluates the rules
// The default rego package is "mediator"
// The default rego query is "data.mediator.allow"
type Evaluator struct {
	cfg      *Config
	regoOpts []func(*rego.Rego)
	reseval  resultEvaluator
}

// Input is the input for the rego evaluator
type Input struct {
	// Policy is the values set for the policy
	Policy map[string]any `json:"policy"`
	// Ingested is the values set for the ingested data
	Ingested any `json:"ingested"`
}

// NewRegoEvaluator creates a new rego evaluator
func NewRegoEvaluator(cfg *pb.RuleType_Definition_Eval_Rego) (*Evaluator, error) {
	c, err := parseConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse rego config: %w", err)
	}

	re := c.getEvalType()

	return &Evaluator{
		cfg:     c,
		reseval: re,
		regoOpts: []func(*rego.Rego){
			re.getQuery(),
			rego.Module("mediator.rego", c.Def),
			rego.Strict(true),
		},
	}, nil
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

	var buf bytes.Buffer
	r := e.newRegoFromOptions(
		rego.Dump(&buf),
	)
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("could not prepare Rego: %w", err)
	}

	rs, err := pq.Eval(ctx, rego.EvalInput(&Input{
		Policy:   pol,
		Ingested: obj,
	}))
	if err != nil {
		return fmt.Errorf("error evaluating policy. Might be wrong input: %w", err)
	}

	return e.reseval.parseResult(rs)
}
