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

package rego

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Config is the configuration for the rego evaluator
type Config struct {
	// Type is the type of evaluation to perform
	Type EvaluationType `json:"type" mapstructure:"type" validate:"required"`
	// Def is the definition of the profile
	Def             string                      `json:"def" mapstructure:"def" validate:"required"`
	ViolationFormat ConstraintsViolationsFormat `json:"violation_format" mapstructure:"violationFormat"`
}

func (c *Config) getEvalType() resultEvaluator {
	switch c.Type {
	case DenyByDefaultEvaluationType:
		return &denyByDefaultEvaluator{}
	case ConstraintsEvaluationType:
		return &constraintsEvaluator{
			format: c.ViolationFormat,
		}
	}

	return nil
}

func parseConfig(cfg *minderv1.RuleType_Definition_Eval_Rego) (*Config, error) {
	if cfg == nil {
		return nil, errors.New("config was missing")
	}

	var conf Config
	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := mapstructure.Decode(cfg, &conf); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	if err := validate.Struct(&conf); err != nil {
		return nil, fmt.Errorf("config failed validation: %w", err)
	}

	if cfg.ViolationFormat == nil {
		conf.ViolationFormat = ConstraintsViolationsOutputText
	}

	typ := conf.getEvalType()
	if typ == nil {
		return nil, fmt.Errorf("unknown evaluation type: %s", conf.Type)
	}

	return &conf, nil
}
