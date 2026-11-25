// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/spf13/cobra"
	"github.com/styrainc/regal/pkg/linter"
	"github.com/styrainc/regal/pkg/rules"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/engine/actions/remediate/rest"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

// CmdLint is the command for linting a rule type definition
func CmdLint() *cobra.Command {
	var lintCmd = &cobra.Command{
		Use:          "lint",
		Short:        "lint a rule type definition",
		Long:         `The 'rule type lint' subcommand allows you lint a rule type definition`,
		RunE:         lintCmdRun,
		SilenceUsage: true,
	}
	lintCmd.Flags().StringP("rule-type", "r", "", "file to read rule type definition from")
	lintCmd.Flags().BoolP("skip-rego", "s", false, "skip rego rule validation")

	if err := lintCmd.MarkFlagRequired("rule-type"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	return lintCmd
}

func lintCmdRun(cmd *cobra.Command, _ []string) error {
	rtpath := cmd.Flag("rule-type")
	skipRego := cmd.Flag("skip-rego").Value.String() == "true"

	ctx := cmd.Context()
	rtpathStr := rtpath.Value.String()

	files, err := util.ExpandFileArgs(nil, rtpathStr)
	if err != nil {
		return fmt.Errorf("error expanding file args: %w", err)
	}

	var lintErrors []error
	for _, f := range files {
		if shouldSkipFile(f.Path) {
			continue
		}

		rt, err := readRuleTypeForLint(f.Path)
		if err != nil && f.Expanded && isNotRuleType(f.Path, err) {
			cmd.PrintErrf("Skipping file %s: not a rule type\n", f.Path)
			continue
		}
		if err != nil {
			lintErrors = append(lintErrors, fmt.Errorf("error reading rule type from file %s: %w", f.Path, err))
			continue
		}

		if err := rt.Validate(); err != nil {
			lintErrors = append(lintErrors, fmt.Errorf("error validating rule type from file %s: %w", f.Path, err))
			continue
		}
		// get file name without extension
		ruleName := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
		if rt.Name != ruleName {
			lintErrors = append(lintErrors, fmt.Errorf("rule type name does not match file name: %s != %s", rt.Name, ruleName))
			continue
		}

		if err := validateDefinition(ctx, rt, f.Path, rtpathStr, skipRego, cmd.OutOrStdout()); err != nil {
			lintErrors = append(lintErrors, err)
			continue
		}
	}

	if len(lintErrors) > 0 {
		for _, err := range lintErrors {
			cmd.PrintErrf("%s\n", err)
		}
		return fmt.Errorf("failed linting rule type")
	}

	return nil
}

// validateDefinition runs the checks that depend on the rule type's evaluation
// and remediation engines.
func validateDefinition(
	ctx context.Context, rt *minderv1.RuleType, path, rtpath string, skipRego bool, out io.Writer,
) error {
	if rt.Def.Eval.Type == rego.RegoEvalType && !skipRego {
		if err := validateRegoRule(ctx, rt.Def.Eval.Rego, rtpath, out); err != nil {
			return fmt.Errorf("failed validating rego rule from file %s: %w", path, err)
		}
	}

	if rt.Def.GetRemediate().GetType() == rest.RemediateType {
		if err := validateRestRemediationBody(rt.Def.GetRemediate().GetRest()); err != nil {
			return fmt.Errorf("failed validating rest remediation from file %s: %w", path, err)
		}
	}

	return nil
}

func shouldSkipFile(f string) bool {
	// if the file is not json, yaml or rego, skip it
	// Get file extension
	ext := filepath.Ext(f)
	switch ext {
	case ".yaml", ".yml", ".json", ".rego":
		return false
	default:
		fmt.Fprintf(os.Stderr, "Skipping file %s: not a yaml, json or rego file\n", f)
		return true
	}
}

// readRuleTypeForLint reads a rule type from a file. Rego files are decoded the
// same way `minder ruletype create` decodes them.
func readRuleTypeForLint(fpath string) (*minderv1.RuleType, error) {
	if filepath.Ext(fpath) != ".rego" {
		return readRuleTypeFromFile(fpath)
	}

	decoder, closer := fileconvert.DecoderForFile(nil, fpath)
	if decoder == nil {
		return nil, fmt.Errorf("error opening file: %s", fpath)
	}
	defer closer.Close()

	return fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
}

// isNotRuleType reports whether a file that failed to load was never meant to be
// a rule type, so that it can be skipped when found in a directory.
func isNotRuleType(fpath string, err error) bool {
	if minderv1.YouMayHaveTheWrongResource(err) || errors.Is(err, fileconvert.ErrNoRegoMetadata) {
		return true
	}
	return filepath.Ext(fpath) == ".rego" && isAnnotatedPolicy(fpath)
}

// isAnnotatedPolicy reports whether a Rego file has readable METADATA that sets
// nothing specific to rule types, such as an annotated helper policy.
func isAnnotatedPolicy(fpath string) bool {
	contents, err := os.ReadFile(filepath.Clean(fpath))
	if err != nil {
		return false
	}
	isRuleType, err := fileconvert.HasRuleTypeMetadata(contents)
	return err == nil && !isRuleType
}

// validateRestRemediationBody checks the body of a rest remediation the way the
// remediator uses it: the body is a text template, and once rendered it is
// decoded as a JSON object. A body without template actions renders to itself,
// so it must already be a JSON object. A body with template actions depends on
// the entity and profile it is rendered for, so only its template syntax is
// checked here.
func validateRestRemediationBody(r *minderv1.RestType) error {
	body := r.GetBody()
	if body == "" {
		return nil
	}

	tmpl, err := template.New("body").Funcs(util.TemplateFuncs).Parse(body)
	if err != nil {
		return fmt.Errorf("body is not a valid template: %w", err)
	}
	if tmpl.Tree != nil {
		for _, node := range tmpl.Root.Nodes {
			if node.Type() != parse.NodeText {
				return nil
			}
		}
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(body), &obj); err != nil {
		return fmt.Errorf("body is not a valid JSON object: %w", err)
	}
	return nil
}

func validateRegoRule(ctx context.Context, r *minderv1.RuleType_Definition_Eval_Rego, path string, out io.Writer) error {
	if r == nil {
		return fmt.Errorf("rego rule is nil")
	}

	if r.Def == "" {
		return fmt.Errorf("rego rule definition is empty")
	}

	inputs, err := rules.InputFromTextWithOptions(path, r.Def, ast.ParserOptions{RegoVersion: ast.RegoV1})
	if err != nil {
		if _, v0Err := ast.ParseModuleWithOpts(path, r.Def, ast.ParserOptions{RegoVersion: ast.RegoV0}); v0Err == nil {
			return fmt.Errorf("%s", rego.V0MigrationMessage)
		}
		return fmt.Errorf("failed parsing rego rule: %w", err)
	}

	lint := linter.NewLinter().WithInputModules(&inputs)

	res, err := lint.Lint(ctx)
	if err != nil {
		return fmt.Errorf("failed linting rego rule: %w", err)
	}

	if err := yaml.NewEncoder(out).Encode(res); err != nil {
		return fmt.Errorf("failed writing lint results: %w", err)
	}

	return nil
}
