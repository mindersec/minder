// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package test provides the test command for mindev
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
			results, err := runner.RunPaths(args)
			if err != nil {
				cmd.PrintErrf("Error(s) running tests:\n%v\n", err)
			}

			if len(results) == 0 {
				cmd.Printf("No tests found\n")
				return err
			}

			hasFailures := false
			for _, res := range results {
				if len(res.Failures) > 0 {
					hasFailures = true
					cmd.Printf("FAIL: %s\n", res.Name)
					for _, f := range res.Failures {
						cmd.Printf("  - %s\n", f)
					}
				} else {
					cmd.Printf("PASS: %s\n", res.Name)
				}
			}

			if hasFailures {
				return fmt.Errorf("one or more tests failed")
			}

			return err
		},
	}

	return cmd
}
