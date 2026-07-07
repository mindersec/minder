// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package test provides the test command for mindev
package test

import (
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/pkg/ruletest"
)

// CmdTest returns the test cobra command
func CmdTest() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "test [paths...]",
		Short: "Run Minder rule tests",
		Long: "Run Starlark-based tests for Minder rules. Each path may be a file or directory. " +
			"If no paths are provided, tests the current directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFormat != "text" && outputFormat != "junit" {
				return fmt.Errorf("unsupported output format %q: must be \"text\" or \"junit\"", outputFormat)
			}

			if len(args) == 0 {
				args = []string{"."}
			}

			runner := ruletest.NewRunner()
			results, err := runner.RunPaths(args)
			if err != nil {
				cmd.PrintErrf("Error(s) running tests:\n%v\n", err)
			}

			if len(results) == 0 {
				if outputFormat == "text" {
					cmd.Printf("No tests found\n")
				}
				return nil
			}

			if outputFormat == "junit" {
				suites := ruletest.AsJUnit(results)
				bytes, fmtErr := xml.MarshalIndent(suites, "", "  ")
				if fmtErr != nil {
					return fmtErr
				}
				cmd.Println(xml.Header + string(bytes))

				for _, res := range results {
					if len(res.Failures) > 0 {
						return errors.New("one or more tests failed")
					}
				}
				return nil
			}

			hasFailures := false
			for _, res := range results {
				if len(res.Failures) > 0 {
					hasFailures = true
					cmd.Printf("FAIL: %s/%s\n", res.Filename, res.Name)
					for _, f := range res.Failures {
						cmd.Printf("  - %s\n", f)
					}
				} else {
					cmd.Printf("PASS: %s/%s\n", res.Filename, res.Name)
				}
			}

			if hasFailures {
				return errors.New("one or more tests failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, junit)")

	return cmd
}
