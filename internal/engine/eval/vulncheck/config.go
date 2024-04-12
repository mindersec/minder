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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type vulnDbType string

const (
	vulnDbTypeOsv vulnDbType = "osv"
)

type packageRepository struct {
	Url string `json:"url" mapstructure:"url" validate:"required"`
}

type ecosystemConfig struct {
	Name string `json:"name" mapstructure:"name" validate:"required"`
	//nolint:lll
	DbType vulnDbType `json:"vulnerability_database_type" mapstructure:"vulnerability_database_type" validate:"required"`
	//nolint:lll
	DbEndpoint        string            `json:"vulnerability_database_endpoint" mapstructure:"vulnerability_database_endpoint" validate:"required"`
	PackageRepository packageRepository `json:"package_repository" mapstructure:"package_repository" validate:"required"`
	SumRepository     packageRepository `json:"sum_repository" mapstructure:"sum_repository" validate:"required"`
}

// config is the configuration for the vulncheck evaluator
type config struct {
	Action          pr_actions.Action `json:"action" mapstructure:"action" validate:"required"`
	EcosystemConfig []ecosystemConfig `json:"ecosystem_config" mapstructure:"ecosystem_config" validate:"required"`
}

func defaultConfig() *config {
	return &config{
		Action: pr_actions.ActionReviewPr,
		EcosystemConfig: []ecosystemConfig{
			{
				Name:       "npm",
				DbType:     vulnDbTypeOsv,
				DbEndpoint: "https://api.osv.dev/v1/query",
				PackageRepository: packageRepository{
					Url: "https://registry.npmjs.org",
				},
			},
			{
				Name:       "pypi",
				DbType:     vulnDbTypeOsv,
				DbEndpoint: "https://api.osv.dev/v1/query",
				PackageRepository: packageRepository{
					Url: "https://pypi.org/pypi",
				},
			},
			{
				Name:       "go",
				DbType:     vulnDbTypeOsv,
				DbEndpoint: "https://api.osv.dev/v1/query",
				PackageRepository: packageRepository{
					Url: "https://proxy.golang.org",
				},
				SumRepository: packageRepository{
					Url: "https://sum.golang.org",
				},
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
