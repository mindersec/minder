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
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

type vulnDbType string

const (
	vulnDbTypeOsv vulnDbType = "osv"
)

type action string

const (
	actionReviewPr     action = "reject_pr"
	actionComment      action = "comment"
	actionCommitStatus action = "commit_status"
	actionPolicyOnly   action = "policy_only"
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
}

// config is the configuration for the vulncheck evaluator
type config struct {
	Action          action            `json:"action" mapstructure:"action" validate:"required"`
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

func pbEcosystemAsString(ecosystem pb.DepEcosystem) string {
	switch ecosystem {
	case pb.DepEcosystem_DEP_ECOSYSTEM_NPM:
		return "npm"
	case pb.DepEcosystem_DEP_ECOSYSTEM_UNSPECIFIED:
		// this shouldn't happen
		return ""
	default:
		return ""
	}
}

func (c *config) getEcosystemConfig(ecosystem pb.DepEcosystem) *ecosystemConfig {
	for _, eco := range c.EcosystemConfig {
		sEco := pbEcosystemAsString(ecosystem)
		if sEco == "" {
			continue
		}

		if eco.Name == sEco {
			return &eco
		}
	}

	return nil
}
