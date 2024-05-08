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

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"

	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	// SummaryScore is the score to use for the summary score
	SummaryScore = "score"
	// DefaultScore is the default score to use
	DefaultScore = ""
)

type ecosystemConfig struct {
	Name string `json:"name" mapstructure:"name" validate:"required"`

	// Score is the score to use for the ecosystem. The actual score
	// evaluated depends on the `evaluate_score` field.
	Score float64 `json:"score" mapstructure:"score" validate:"required"`

	// EvaluateScore tells the trusty executor which score to use
	// for evaluation. This is useful when the trusty API returns.
	// The default is the summary score. If `score` or an empty string, the
	// summary score is used.
	// If `evaluate_score` is set to something else (e.g. `provenance`)
	// then that score is used, which comes from the details field.
	EvaluateScore string `json:"evaluate_score" mapstructure:"evaluate_score"`
}

// config is the configuration for the vulncheck evaluator
type config struct {
	Action          pr_actions.Action `json:"action" mapstructure:"action" validate:"required"`
	EcosystemConfig []ecosystemConfig `json:"ecosystem_config" mapstructure:"ecosystem_config" validate:"required"`
}

func defaultConfig() *config {
	return &config{
		Action: pr_actions.ActionSummary,
		EcosystemConfig: []ecosystemConfig{
			{
				Name:  "npm",
				Score: 5.0,
			},
			{
				Name:  "pypi",
				Score: 5.0,
			},
			{
				Name:  "go",
				Score: 5.0,
			},
		},
	}
}

func parseConfig(ruleCfg map[string]any) (*config, error) {
	if len(ruleCfg) == 0 {
		return defaultConfig(), nil
	}

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

func (c *config) getEcosystemConfig(ecosystem pb.DepEcosystem) *ecosystemConfig {
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

func (ec *ecosystemConfig) getScoreSource() string {
	if ec.EvaluateScore == DefaultScore || ec.EvaluateScore == SummaryScore {
		return SummaryScore
	}

	return ec.EvaluateScore
}

func (ec *ecosystemConfig) getScore(inSummary ScoreSummary) (float64, error) {
	if inSummary.Score != nil && (ec.EvaluateScore == DefaultScore || ec.EvaluateScore == SummaryScore) {
		return *inSummary.Score, nil
	}

	// If the score is not the summary score, then it must be in the details
	rawScore, ok := inSummary.Description[ec.EvaluateScore]
	if !ok {
		return 0, fmt.Errorf("score %s not found in details", ec.EvaluateScore)
	}

	s, ok := rawScore.(float64)
	if !ok {
		return 0, fmt.Errorf("score %s is not a float64", ec.EvaluateScore)
	}

	return s, nil
}
