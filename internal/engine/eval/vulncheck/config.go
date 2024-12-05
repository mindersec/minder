// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"

	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	pbinternal "github.com/mindersec/minder/internal/proto"
)

type vulnDbType string

const (
	vulnDbTypeOsv vulnDbType = "osv"
	defaultAction            = pr_actions.ActionReviewPr
)

var (
	defaultEcosystemConfig = []ecosystemConfig{
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
	}
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
