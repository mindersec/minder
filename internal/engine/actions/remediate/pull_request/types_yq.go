// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	gologging "gopkg.in/op/go-logging.v1"

	"github.com/mindersec/minder/internal/engine/interfaces"
	textutil "github.com/mindersec/minder/internal/util"
)

// TODO: document the YQ remediation model.

// unfortunately yqlib does seem to be using global variables...
func init() {
	// setting the log level to critical pretty much silences the logging
	gologging.SetLevel(gologging.CRITICAL, yqLibModule)
	yqlib.InitExpressionParser()
}

type patternType string

const (
	patternTypeGlob       patternType = "glob"
	patternTypeEvalOutput patternType = "eval_output"

	maxExpressionSize  int = 3000
	maxTargetListSize  int = 256 * 1024
	maxEvalOutputFiles int = 1000
)

const (
	yqLibModule = "yq-lib"
)

var _ fsModifier = (*yqExecute)(nil)

type yqExecuteConfig struct {
	Expression string `json:"expression"`
	Patterns   []struct {
		Pattern string `json:"pattern"`
		Type    string `json:"type"`
	}
}

type yqExecute struct {
	fsChangeSet

	config yqExecuteConfig
}

var _ modificationConstructor = newYqExecute

func newYqExecute(
	params *modificationConstructorParams,
) (fsModifier, error) {

	confMap := make(map[string]any)
	if params.prCfg.GetParams() != nil {
		confMap = params.prCfg.Params.AsMap()
	}

	rawConfig, err := json.Marshal(confMap)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal config")
	}

	var conf yqExecuteConfig
	err = json.Unmarshal(rawConfig, &conf)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal config")
	}

	return &yqExecute{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},

		config: conf,
	}, nil
}

// templateParams are the parameters for template expansion for the expression and target patterns
type templateParams struct {
	// Entity is the metadata for the entity which was evaluated
	Entity any
	// Profile is the parameters to be used in the template
	Profile map[string]any
	// Params are the rule instance parameters
	Params map[string]any
	// EvalResultOutput is the output from the rule evaluation
	EvalResultOutput any
}

func (yq *yqExecute) createFsModEntries(ctx context.Context, ent proto.Message, params interfaces.ActionsParams) error {
	templateData := templateParams{
		Entity:  ent,
		Profile: params.GetRule().Def,
		Params:  params.GetRule().Params,
	}
	if params.GetEvalResult() != nil {
		templateData.EvalResultOutput = params.GetEvalResult().Output
	}

	expression, err := expandExpression(ctx, yq.config.Expression, templateData)
	if err != nil {
		return err
	}

	matchingFiles, err := yq.resolveTargets(ctx, templateData)
	if err != nil {
		return err
	}

	for _, file := range matchingFiles {
		newContent, err := yq.executeYq(file, expression)
		if err != nil {
			return fmt.Errorf("cannot execute yq on %s: %w", file, err)
		}
		yq.entries = append(yq.entries, &fsEntry{
			Path:    file,
			Content: newContent,
			Mode:    filemode.Regular.String(),
		})
	}

	return nil
}

func expandExpression(ctx context.Context, expressionText string, data templateParams) (string, error) {
	if expressionText == "" {
		return "", nil
	}

	expressionBytes := new(bytes.Buffer)
	expressionTmpl, err := textutil.NewSafeTextTemplate(&expressionText, "expression")
	if err != nil {
		return "", fmt.Errorf("unable to parse templates in expression: %w", err)
	}
	if err := expressionTmpl.Execute(ctx, expressionBytes, data, maxExpressionSize); err != nil {
		return "", fmt.Errorf("unable to expand templates in expression: %w", err)
	}
	return expressionBytes.String(), nil
}

func (yq *yqExecute) resolveTargets(ctx context.Context, data templateParams) ([]string, error) {
	targets := make([]string, 0)
	seen := make(map[string]struct{})

	addTarget := func(path string) error {
		normalized, err := normalizeTargetPath(path)
		if err != nil {
			return err
		}
		if _, ok := seen[normalized]; ok {
			return nil
		}
		seen[normalized] = struct{}{}
		targets = append(targets, normalized)
		return nil
	}

	for _, pattern := range yq.config.Patterns {
		switch pattern.Type {
		case string(patternTypeGlob):
			patternMatches, err := util.Glob(yq.fs, pattern.Pattern)
			if err != nil {
				return nil, fmt.Errorf("cannot get matching files: %w", err)
			}
			for _, match := range patternMatches {
				if err := addTarget(match); err != nil {
					return nil, err
				}
			}
		case string(patternTypeEvalOutput):
			paths, err := resolveEvalOutputPattern(ctx, pattern.Pattern, data)
			if err != nil {
				return nil, err
			}
			for _, path := range paths {
				if err := addTarget(path); err != nil {
					return nil, fmt.Errorf("invalid eval_output target: %w", err)
				}
			}
		default:
			zerolog.Ctx(ctx).
				Warn().
				Str("pattern.Type", pattern.Type).
				Msg("unsupported pattern type")
		}
	}

	if err := yq.validateTargets(targets); err != nil {
		return nil, err
	}

	return targets, nil
}

func (yq *yqExecute) validateTargets(targets []string) error {
	for _, target := range targets {
		info, err := yq.fs.Stat(target)
		if err != nil {
			// Let executeYq decide how to handle missing files so that
			// eval_output targets can later support create-if-missing.
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("cannot stat target %s: %w", target, err)
		}
		if info.IsDir() {
			return fmt.Errorf("target %s is a directory", target)
		}
	}

	return nil
}

func resolveEvalOutputPattern(ctx context.Context, patternText string, data templateParams) ([]string, error) {
	if patternText == "" {
		return nil, fmt.Errorf("eval_output pattern cannot be empty")
	}

	patternTmpl, err := textutil.NewSafeTextTemplate(&patternText, "eval_output_pattern")
	if err != nil {
		return nil, fmt.Errorf("cannot parse eval_output pattern template: %w", err)
	}

	rendered := new(bytes.Buffer)
	if err := patternTmpl.Execute(ctx, rendered, data, maxTargetListSize); err != nil {
		return nil, fmt.Errorf("cannot expand eval_output pattern template: %w", err)
	}

	trimmed := strings.TrimSpace(rendered.String())
	if !strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("eval_output pattern must render a JSON array of strings")
	}

	var paths []string
	if err := json.Unmarshal([]byte(trimmed), &paths); err != nil {
		return nil, fmt.Errorf("cannot parse eval_output pattern as a JSON array of strings: %w", err)
	}
	if len(paths) > maxEvalOutputFiles {
		return nil, fmt.Errorf(
			"eval_output pattern returned %d paths, maximum is %d", len(paths), maxEvalOutputFiles)
	}
	for _, path := range paths {
		if len(path) > PathBytesLimit {
			return nil, fmt.Errorf("eval_output path exceeds %d bytes: %q", PathBytesLimit, path)
		}
	}

	return paths, nil
}

func normalizeTargetPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("target path cannot be empty")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("target path %q must be relative to the repository root", path)
	}

	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("target path %q must stay within the repository root", path)
	}

	return cleaned, nil
}

func (yq *yqExecute) modifyFs() ([]*fsEntry, error) {
	err := yq.writeEntries()
	if err != nil {
		return nil, fmt.Errorf("cannot write entries: %w", err)
	}
	return yq.entries, nil
}

func (yq *yqExecute) executeYq(filename, expression string) (string, error) {
	file, err := yq.fs.Open(filename)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", filename, err)
	}

	out := new(bytes.Buffer)
	encoder := yqlib.NewYamlEncoder(yqlib.NewDefaultYamlPreferences())
	printer := yqlib.NewPrinter(encoder, yqlib.NewSinglePrinterWriter(out))

	expParser := yqlib.ExpressionParser
	expressionNode, err := expParser.ParseExpression(expression)
	if err != nil {
		return "", fmt.Errorf("cannot parse expression: %w", err)
	}

	decoder := yqlib.NewYamlDecoder(yqlib.NewDefaultYamlPreferences())
	parser := yqlib.NewStreamEvaluator()
	_, err = parser.Evaluate(filename, file, expressionNode, printer, decoder)
	if err != nil {
		return "", fmt.Errorf("cannot evaluate expression: %w", err)
	}

	return out.String(), nil
}
