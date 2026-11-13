// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package remediationparams contains validation for method-specific pull
// request remediation parameters. It is shared between rule type validation
// and remediation execution so both reject the same invalid configurations.
package remediationparams

import (
	"encoding/json"
	"fmt"

	textutil "github.com/mindersec/minder/internal/util"
)

const (
	// MethodContent replaces file contents with templates.
	MethodContent = "minder.content"
	// MethodReplaceTagsWithSHA replaces action tags with SHA digests.
	MethodReplaceTagsWithSHA = "minder.actions.replace_tags_with_sha"
	// MethodYQEvaluate evaluates a yq expression on selected files.
	MethodYQEvaluate = "minder.yq.evaluate"
	// MethodDelete deletes files matching glob patterns.
	MethodDelete = "minder.delete"
)

// PatternType is the type of a file selector in a pull request remediation.
type PatternType string

const (
	// PatternTypeGlob selects files using a glob pattern.
	PatternTypeGlob PatternType = "glob"
	// PatternTypeEvalOutput selects files from rendered evaluation output.
	PatternTypeEvalOutput PatternType = "eval_output"
)

// Pattern selects files for a pull request remediation.
type Pattern struct {
	Pattern string `json:"pattern"`
	Type    string `json:"type"`
}

// YQConfig is the configuration for the minder.yq.evaluate method.
type YQConfig struct {
	Expression string    `json:"expression"`
	Patterns   []Pattern `json:"patterns"`
}

// DeleteConfig is the configuration for the minder.delete method.
type DeleteConfig struct {
	Patterns []Pattern `json:"patterns"`
}

// ValidatePullRequestParams validates the method-specific parameters of a
// pull request remediation.
func ValidatePullRequestParams(method string, params map[string]any) error {
	switch method {
	case MethodYQEvaluate:
		_, err := ParseYQConfig(params)
		return err
	case MethodDelete:
		_, err := ParseDeleteConfig(params)
		return err
	default:
		return nil
	}
}

// ParseYQConfig parses and validates minder.yq.evaluate parameters.
func ParseYQConfig(params map[string]any) (*YQConfig, error) {
	rawConfig, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal config")
	}

	var conf YQConfig
	if err := json.Unmarshal(rawConfig, &conf); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	if err := validateYQPatterns(conf.Patterns); err != nil {
		return nil, err
	}

	return &conf, nil
}

func validateYQPatterns(patterns []Pattern) error {
	if len(patterns) == 0 {
		return fmt.Errorf("yq modification requires at least one valid pattern")
	}

	for i, pattern := range patterns {
		if pattern.Pattern == "" {
			return fmt.Errorf("patterns[%d] missing required field: pattern", i)
		}
		if pattern.Type == "" {
			return fmt.Errorf("patterns[%d] missing required field: type", i)
		}

		switch pattern.Type {
		case string(PatternTypeGlob):
		case string(PatternTypeEvalOutput):
			if _, err := textutil.NewSafeTextTemplate(&pattern.Pattern, "eval_output_pattern"); err != nil {
				return fmt.Errorf("patterns[%d] invalid eval_output template: %w", i, err)
			}
		default:
			return fmt.Errorf("patterns[%d] unsupported type %q (supported: %q, %q)",
				i, pattern.Type, PatternTypeGlob, PatternTypeEvalOutput)
		}
	}

	return nil
}

// ParseDeleteConfig parses and validates minder.delete parameters.
func ParseDeleteConfig(params map[string]any) (*DeleteConfig, error) {
	if params == nil {
		return nil, fmt.Errorf("delete modification requires params")
	}

	patternsRaw, ok := params["patterns"]
	if !ok {
		return nil, fmt.Errorf("delete modification requires patterns")
	}

	patternsList, ok := patternsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("patterns must be a list")
	}

	conf := DeleteConfig{}
	for i, p := range patternsList {
		pm, ok := p.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("patterns[%d] must be an object", i)
		}
		pattern, _ := pm["pattern"].(string)
		typ, _ := pm["type"].(string)
		if pattern == "" {
			return nil, fmt.Errorf("patterns[%d] missing required field: pattern", i)
		}
		if typ == "" {
			return nil, fmt.Errorf("patterns[%d] missing required field: type", i)
		}
		if typ != string(PatternTypeGlob) {
			return nil, fmt.Errorf("patterns[%d] unsupported type %q (only %q is supported)", i, typ, PatternTypeGlob)
		}
		conf.Patterns = append(conf.Patterns, Pattern{Pattern: pattern, Type: typ})
	}

	if len(conf.Patterns) == 0 {
		return nil, fmt.Errorf("delete modification requires at least one valid pattern")
	}

	return &conf, nil
}
