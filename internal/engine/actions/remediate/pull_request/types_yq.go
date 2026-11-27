// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
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
	patternTypeExact      patternType = "exact"
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

type yqPattern struct {
	Pattern        string `json:"pattern"`
	Type           string `json:"type"`
	CreateIfNeeded bool   `json:"create_if_needed"`
}

type yqExecuteConfig struct {
	Expression string      `json:"expression"`
	Patterns   []yqPattern `json:"patterns"`
}

type yqTarget struct {
	path   string
	create bool
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
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	if err := validateYqPatterns(conf.Patterns); err != nil {
		return nil, err
	}

	return &yqExecute{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},

		config: conf,
	}, nil
}

func validateYqPatterns(patterns []yqPattern) error {
	for i, pattern := range patterns {
		if pattern.Pattern == "" {
			return fmt.Errorf("patterns[%d] missing required field: pattern", i)
		}
		if pattern.Type == "" {
			return fmt.Errorf("patterns[%d] missing required field: type", i)
		}

		switch pattern.Type {
		case string(patternTypeExact):
			if !validExactTargetPath(pattern.Pattern) {
				return fmt.Errorf("invalid exact path %q: must be relative to the repository", pattern.Pattern)
			}
		case string(patternTypeGlob):
		case string(patternTypeEvalOutput):
			if _, err := textutil.NewSafeTextTemplate(&pattern.Pattern, "eval_output_pattern"); err != nil {
				return fmt.Errorf("patterns[%d] invalid eval_output template: %w", i, err)
			}
		default:
			return fmt.Errorf("patterns[%d] unsupported type %q (supported: %q, %q, %q)",
				i, pattern.Type, patternTypeGlob, patternTypeEvalOutput, patternTypeExact)
		}
	}

	return nil
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
		newContent, err := yq.executeYq(file.path, expression, file.create)
		if err != nil {
			return fmt.Errorf("cannot execute yq on %s: %w", file.path, err)
		}
		yq.entries = append(yq.entries, &fsEntry{
			Path:    file.path,
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

func (yq *yqExecute) resolveTargets(ctx context.Context, data templateParams) ([]yqTarget, error) {
	targets := make([]yqTarget, 0)
	seen := make(map[string]int)

	addTarget := func(path string, create bool) error {
		normalized, err := normalizeTargetPath(path)
		if err != nil {
			return err
		}
		if index, ok := seen[normalized]; ok {
			targets[index].create = targets[index].create || create
			return nil
		}
		seen[normalized] = len(targets)
		targets = append(targets, yqTarget{path: normalized, create: create})
		return nil
	}

	for _, pattern := range yq.config.Patterns {
		var paths []string
		var err error
		create := false
		switch pattern.Type {
		case string(patternTypeExact):
			if !validExactTargetPath(pattern.Pattern) {
				return nil, fmt.Errorf("invalid exact path %q: must be relative to the repository", pattern.Pattern)
			}
			paths = []string{pattern.Pattern}
			create = pattern.CreateIfNeeded
		case string(patternTypeGlob):
			paths, err = util.Glob(yq.fs, pattern.Pattern)
		case string(patternTypeEvalOutput):
			paths, err = resolveEvalOutputPattern(ctx, pattern.Pattern, data)
		default:
			return nil, fmt.Errorf("unsupported pattern type %q", pattern.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("cannot resolve %s targets: %w", pattern.Type, err)
		}
		for _, path := range paths {
			if err := addTarget(path, create); err != nil {
				return nil, fmt.Errorf("invalid %s target: %w", pattern.Type, err)
			}
		}
	}

	if err := yq.validateTargets(targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func validExactTargetPath(path string) bool {
	return fs.ValidPath(path) && path != "." && !strings.ContainsAny(path, "\\:\x00")
}

func (yq *yqExecute) validateTargets(targets []yqTarget) error {
	for _, target := range targets {
		info, err := yq.fs.Stat(target.path)
		if err != nil {
			// Let executeYq decide how to handle missing files so that
			// eval_output targets can later support create-if-missing.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("cannot stat target %s: %w", target.path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("target %s is a directory", target.path)
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

func (yq *yqExecute) executeYq(filename, expression string, create bool) (string, error) {
	file, err := yq.fs.Open(filename)
	var input io.Reader = file
	if err != nil {
		if !create || !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("cannot read file %s: %w", filename, err)
		}
		// An empty stream has no documents for yq to evaluate.
		input = strings.NewReader("null\n")
	} else {
		defer file.Close()
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
	_, err = parser.Evaluate(filename, input, expressionNode, printer, decoder)
	if err != nil {
		return "", fmt.Errorf("cannot evaluate expression: %w", err)
	}

	return out.String(), nil
}
