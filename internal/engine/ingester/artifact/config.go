// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"

	artif "github.com/mindersec/minder/internal/providers/artifact"
)

type artifactType string

const (
	artifactTypeContainer artifactType = "container"
	artifactTypeUnknown   artifactType = "unknown"
)

func newArtifactIngestType(s string) artifactType {
	switch strings.ToLower(s) {
	case string(artifactTypeContainer):
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

// ValidateParams checks that params are valid for the artifact ingester.
func ValidateParams(params map[string]any) error {
	cfg, err := configFromParams(params)
	if err != nil {
		return err
	}

	if _, err := artif.BuildFilter(cfg.Tags, cfg.TagRegex, artif.MatchAllTags); err != nil {
		return fmt.Errorf("invalid artifact ingester params: %w", err)
	}

	return nil
}
