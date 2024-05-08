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

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

type artifactType string

const (
	artifactTypeContainer artifactType = "container"
	artifactTypeUnknown   artifactType = "unknown"
)

func newArtifactIngestType(s string) artifactType {
	switch strings.ToLower(s) {
	case "container":
		return artifactTypeContainer
	default:
		return artifactTypeUnknown
	}
}

type ingesterConfig struct {
	Name     string       `yaml:"name" json:"name" mapstructure:"name"`
	Tags     []string     `yaml:"tags" json:"tags" mapstructure:"tags"`
	Sigstore string       `yaml:"sigstore" json:"sigstore" mapstructure:"sigstore"`
	TagRegex string       `yaml:"tag_regex" json:"tag_regex" mapstructure:"tag_regex"`
	Type     artifactType `yaml:"type" json:"type" mapstructure:"type"`
}

func configFromParams(params map[string]any) (*ingesterConfig, error) {
	cfg := &ingesterConfig{}
	if err := mapstructure.Decode(params, cfg); err != nil {
		return nil, fmt.Errorf("error decoding ingester config: %w", err)
	}

	if cfg.Type == "" {
		cfg.Type = artifactTypeContainer
	}

	return cfg, nil
}
