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
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	"github.com/stacklok/mediator/internal/engine/eval/pr_actions"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

type ecosystemConfig struct {
	Name string `json:"name" mapstructure:"name" validate:"required"`
	//nolint:lll
	Score float64 `json:"score" mapstructure:"score" validate:"required"`
}

// config is the configuration for the vulncheck evaluator
type config struct {
	Action          pr_actions.Action `json:"action" mapstructure:"action" validate:"required"`
	EcosystemConfig []ecosystemConfig `json:"ecosystem_config" mapstructure:"ecosystem_config" validate:"required"`
}

func parseConfig(ruleCfg map[string]any) (*config, error) {
	if ruleCfg == nil {
		return nil, errors.New("config was missing")
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
