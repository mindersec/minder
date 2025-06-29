// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

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
	patternTypeGlob patternType = "glob"
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

// templateParams are the parameters for template expansion for the expression template
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
	matchingFiles := make([]string, 0)
	for _, pattern := range yq.config.Patterns {
		if pattern.Type != string(patternTypeGlob) {
			zerolog.Ctx(ctx).
				Warn().
				Str("pattern.Type", pattern.Type).
				Msg("unsupported pattern type")
			continue
		}

		patternMatches, err := util.Glob(yq.fs, pattern.Pattern)
		if err != nil {
			return fmt.Errorf("cannot get matching files: %w", err)
		}
		matchingFiles = append(matchingFiles, patternMatches...)
	}

	templateData := templateParams{
		Entity:  ent,
		Profile: params.GetRule().Def,
		Params:  params.GetRule().Params,
	}
	if params.GetEvalResult() != nil {
		templateData.EvalResultOutput = params.GetEvalResult().Output
	}

	expression := ""
	if yq.config.Expression != "" {
		expressionBytes := new(bytes.Buffer)
		expressionTmpl, err := textutil.NewSafeTextTemplate(&yq.config.Expression, "expression")
		if err != nil {
			return fmt.Errorf("unable to parse templates in expression: %w", err)
		}
		if err := expressionTmpl.Execute(ctx, expressionBytes, templateData, 3000); err != nil {
			return fmt.Errorf("unable to expand templates in expression: %w", err)
		}
		expression = expressionBytes.String()
	}

	for _, file := range matchingFiles {
		newContent, err := yq.executeYq(file, expression)
		if err != nil {
			return fmt.Errorf("cannot execute yq: %w", err)
		}
		yq.entries = append(yq.entries, &fsEntry{
			Path:    file,
			Content: newContent,
			Mode:    filemode.Regular.String(),
		})
	}

	return nil
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
		return "", fmt.Errorf("cannot read file: %w", err)
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
