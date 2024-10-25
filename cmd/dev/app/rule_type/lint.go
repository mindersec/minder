// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/styrainc/regal/pkg/linter"
	"github.com/styrainc/regal/pkg/rules"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

	files, err := util.ExpandFileArgs(rtpathStr)
	if err != nil {
		return fmt.Errorf("error expanding file args: %w", err)
	}

	var errors []error
	for _, f := range files {
		if shouldSkipFile(f.Path) {
			continue
		}

		rt, err := readRuleTypeFromFile(f.Path)
		if err != nil && f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
			cmd.PrintErrf("Skipping file %s: not a rule type\n", f.Path)
			continue
		}
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading rule type from file %s: %w", f.Path, err))
			continue
		}

		if err := rt.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("error validating rule type from file %s: %w", f.Path, err))
			continue
		}
		// get file name without extension
		ruleName := strings.TrimSuffix(filepath.Base(f.Path), filepath.Ext(f.Path))
		if rt.Name != ruleName {
			errors = append(errors, fmt.Errorf("rule type name does not match file name: %s != %s", rt.Name, ruleName))
			continue
		}

		if rt.Def.Eval.Type == rego.RegoEvalType && !skipRego {
			if err := validateRegoRule(ctx, rt.Def.Eval.Rego, rtpathStr, cmd.OutOrStdout()); err != nil {
				errors = append(errors, fmt.Errorf("failed validating rego rule from file %s: %w", f.Path, err))
				continue
			}
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			cmd.PrintErrf("%s\n", err)
		}
		return fmt.Errorf("failed linting rule type")
	}

	return nil
}

func shouldSkipFile(f string) bool {
	// if the file is not json or yaml, skip it
	// Get file extension
	ext := filepath.Ext(f)
	switch ext {
	case ".yaml", ".yml", ".json":
		return false
	default:
		fmt.Fprintf(os.Stderr, "Skipping file %s: not a yaml or json file\n", f)
		return true
	}
}

func validateRegoRule(ctx context.Context, r *minderv1.RuleType_Definition_Eval_Rego, path string, out io.Writer) error {
	if r == nil {
		return fmt.Errorf("rego rule is nil")
	}

	if r.Def == "" {
		return fmt.Errorf("rego rule definition is empty")
	}

	inputs, err := rules.InputFromText(path, r.Def)
	if err != nil {
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
