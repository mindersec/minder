// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/pkg/ruletest"
)

// CmdTest returns the test cobra command
func CmdTest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [directories...]",
		Short: "Run Minder rule tests",
		Long:  `Run Starlark-based tests for Minder rules. If no directories are provided, tests the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"."}
			}

			runner := ruletest.NewRunner()
			hasFailures := false

			for _, dir := range args {
				fmt.Printf("Running tests in %s...\n", dir)
				results, err := runner.RunDir(dir)
				if err != nil {
					return fmt.Errorf("error running tests in %s: %w", dir, err)
				}

				if len(results) == 0 {
					fmt.Printf("No tests found in %s\n", dir)
					continue
				}

				for _, res := range results {
					if len(res.Failures) > 0 {
						hasFailures = true
						fmt.Printf("FAIL: %s\n", res.Name)
						for _, f := range res.Failures {
							fmt.Printf("  - %s\n", f)
						}
					} else {
						fmt.Printf("PASS: %s\n", res.Name)
					}
				}
			}

			if hasFailures {
				return fmt.Errorf("one or more tests failed")
			}

			return nil
		},
	}

	return cmd
}
