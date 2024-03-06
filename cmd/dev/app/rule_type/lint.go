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

	"github.com/stacklok/minder/internal/engine/eval/rego"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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

	var errors []error
	walkerr := filepath.Walk(rtpathStr, func(path string, info os.FileInfo, walkerr error) error {
		if walkerr != nil {
			return fmt.Errorf("error walking path %s: %w", path, walkerr)
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		rt, err := readRuleTypeFromFile(path)
		if err != nil {
			errors = append(errors, fmt.Errorf("error reading rule type from file %s: %w", path, err))
			return nil
		}

		if err := rt.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("error validating rule type: %w", err))
			return nil
		}

		// get file name without extension
		ruleName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if rt.Name != ruleName {
			errors = append(errors, fmt.Errorf("rule type name does not match file name: %s != %s", rt.Name, ruleName))
			return nil
		}

		if rt.Def.Eval.Type == rego.RegoEvalType && !skipRego {
			if err := validateRegoRule(ctx, rt.Def.Eval.Rego, rtpathStr, cmd.OutOrStdout()); err != nil {
				errors = append(errors, fmt.Errorf("failed validating rego rule: %w", err))
				return nil
			}
		}

		return nil
	})

	if walkerr != nil {
		return fmt.Errorf("error walking path %s: %w", rtpathStr, walkerr)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
		return fmt.Errorf("failed linting rule type")
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
