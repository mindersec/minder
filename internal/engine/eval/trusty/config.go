// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"

	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	pbinternal "github.com/mindersec/minder/internal/proto"
)

var (
	// SummaryScore is the score to use for the summary score
	SummaryScore = "score"
	// DefaultScore is the default score to use
	DefaultScore           = ""
	defaultAction          = pr_actions.ActionReviewPr
	defaultEcosystemConfig = []ecosystemConfig{
		{
			Name:            "npm",
			Score:           5.0,
			Provenance:      5.0,
			Activity:        5.0,
			AllowMalicious:  false,
			AllowDeprecated: false,
		},
		{
			Name:            "pypi",
			Score:           5.0,
			Provenance:      5.0,
			Activity:        5.0,
			AllowDeprecated: false,
		},
		{
			Name:            "go",
			Score:           5.0,
			Provenance:      5.0,
			Activity:        5.0,
			AllowDeprecated: false,
		},
	}
)

type ecosystemConfig struct {
	Name string `json:"name" mapstructure:"name" validate:"required"`

	// Score is the score to use for the ecosystem. The actual score
	// evaluated depends on the `evaluate_score` field.
	Score float64 `json:"score" mapstructure:"score" validate:"required"`

	// The provenance field contains the minimal provenance score
	// to consider the origin of the package as trusted.
	Provenance float64 `json:"provenance" mapstructure:"provenance"`

	// Activity is the minimal activity score that minder needs to find to
	// consider the package as trustworthy.
	Activity float64 `json:"activity" mapstructure:"activity"`

	// AllowMalicious disables blocking PRs introducing malicious dependencies
	AllowMalicious bool `json:"allow_malicious" mapstructure:"allow_malicious"`

	// AllowDeprecated disables blocking pull requests introducing deprecated packages
	AllowDeprecated bool `json:"allow_deprecated" mapstructure:"allow_deprecated"`
}

// config is the configuration for the trusty evaluator
type config struct {
	Action          pr_actions.Action `json:"action" mapstructure:"action" validate:"required"`
	EcosystemConfig []ecosystemConfig `json:"ecosystem_config" mapstructure:"ecosystem_config" validate:"required"`
}

func populateDefaultsIfEmpty(ruleCfg map[string]any) {
	if ruleCfg["ecosystem_config"] == nil {
		ruleCfg["ecosystem_config"] = defaultEcosystemConfig
	} else if ecoCfg, ok := ruleCfg["ecosystem_config"].([]interface{}); ok && len(ecoCfg) == 0 {
		ruleCfg["ecosystem_config"] = defaultEcosystemConfig
	}
	if ruleCfg["action"] == nil {
		ruleCfg["action"] = defaultAction
	}
}

func parseConfig(ruleCfg map[string]any) (*config, error) {
	populateDefaultsIfEmpty(ruleCfg)

	var conf config
	validate := validator.New(validator.WithRequiredStructEnabled())

	if err := mapstructure.Decode(ruleCfg, &conf); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	if err := validate.Struct(&conf); err != nil {
		return nil, fmt.Errorf("config failed validation: %w", err)
	}

	return &conf, nil
}

func (c *config) getEcosystemConfig(ecosystem pbinternal.DepEcosystem) *ecosystemConfig {
	sEco := ecosystem.AsString()
	if sEco == "" {
		return nil
	}
	sEco = strings.ToLower(sEco)

	for _, eco := range c.EcosystemConfig {
		if strings.ToLower(eco.Name) == sEco {
			return &eco
		}
	}

	return nil
}
