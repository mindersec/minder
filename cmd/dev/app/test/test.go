// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package test provides the test command for mindev
package test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/pkg/ruletest"
)

// CmdTest returns the test cobra command
func CmdTest() *cobra.Command {
	var outputFormat string
	var junitFile string

	cmd := &cobra.Command{
		Use:   "test [paths...]",
		Short: "Run Minder rule tests",
		Long: "Run Starlark-based tests for Minder rules. Each path may be a file or directory. " +
			"If no paths are provided, tests the current directory.",
		RunE: func(cmd *cobra.Command, args []string) (finalErr error) {
			if outputFormat != "text" && outputFormat != "junit" {
				return fmt.Errorf("unsupported output format %q: must be \"text\" or \"junit\"", outputFormat)
			}

			cmd.SilenceUsage = true

			if len(args) == 0 {
				args = []string{"."}
			}

			runner := ruletest.NewRunner()
			results, err := runner.RunPaths(args)
			if err != nil {
				cmd.PrintErrf("Error(s) running tests:\n%v\n", err)
				// Record this as the final error message if no tests failed.
				finalErr = errors.New("one or more test files failed to load")
			}

			switch outputFormat {
			case "text":
				formatFailuresHuman(cmd, results)
			case "junit":
				if err := writeJUnit(cmd.OutOrStdout(), results); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported output format %q: must be \"text\" or \"junit\"", outputFormat)
			}

			if junitFile != "" {
				//nolint:gosec // path is provided by the user via a flag
				f, err := os.Create(junitFile)
				if err != nil {
					return fmt.Errorf("failed to create junit file: %w", err)
				}
				defer f.Close()
				if err := writeJUnit(f, results); err != nil {
					return err
				}
			}

			for _, res := range results {
				if len(res.Failures) > 0 {
					finalErr = errors.New("one or more tests failed")
					break
				}
			}
			return finalErr
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format (text, junit)")
	cmd.Flags().StringVar(&junitFile, "junit-file", "", "File to write JUnit report to (in addition to standard output)")

	return cmd
}

func formatFailuresHuman(cmd *cobra.Command, results []ruletest.TestResult) {
	if len(results) == 0 {
		cmd.Printf("No tests found\n")
	}
	for _, res := range results {
		if len(res.Failures) > 0 {
			cmd.Printf("FAIL: %s/%s\n", res.Filename, res.Name)
			for _, f := range res.Failures {
				cmd.Printf("  - %s\n", f)
			}
		} else {
			cmd.Printf("PASS: %s/%s\n", res.Filename, res.Name)
		}
	}
}

func writeJUnit(w io.Writer, results []ruletest.TestResult) error {
	suites := ruletest.AsJUnit(results)
	if _, err := fmt.Fprint(w, xml.Header); err != nil {
		return fmt.Errorf("failed to write XML header: %w", err)
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(suites); err != nil {
		return fmt.Errorf("failed to encode JUnit XML: %w", err)
	}
	return nil
}
