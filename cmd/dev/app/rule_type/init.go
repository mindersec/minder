// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/spf13/cobra"
)

// CmdInit is the command for initializing a rule type definition
func CmdInit() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "initialize a rule type definition",
		Long: `The 'ruletype init' subcommand allows you to initialize a rule type definition

The first positional argument is the directory to initialize the rule type in.
The rule type will be initialized in the current directory if no directory is provided.
`,
		RunE:         initCmdRun,
		SilenceUsage: true,
	}

	initCmd.Flags().StringP("name", "n", "", "name of the rule type")
	initCmd.Flags().BoolP("skip-tests", "s", false, "skip creating test files")

	if err := initCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	return initCmd
}

func initCmdRun(cmd *cobra.Command, args []string) error {
	name := cmd.Flag("name").Value.String()
	skipTests := cmd.Flag("skip-tests").Value.String() == "true"
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	if err := validateRuleTypeName(name); err != nil {
		return err
	}

	ruleTypeFileName := filepath.Join(dir, name+".yaml")
	ruleTypeTestFileName := filepath.Join(dir, name+".test.yaml")
	ruleTypeTestDataDirName := filepath.Join(dir, name+".testdata")

	if err := assertFilesDontExist(
		ruleTypeFileName, ruleTypeTestFileName, ruleTypeTestDataDirName); err != nil {
		return err
	}

	// Create rule type file
	if err := createRuleTypeFile(ruleTypeFileName, name); err != nil {
		return err
	}
	cmd.Printf("Created rule type file: %s\n", ruleTypeFileName)

	if !skipTests {
		// Create rule type test file
		if err := createRuleTypeTestFile(ruleTypeTestFileName, name); err != nil {
			return err
		}
		cmd.Printf("Created rule type test file: %s\n", ruleTypeTestFileName)

		// Create rule type test data directory
		if err := createRuleTypeTestDataDir(ruleTypeTestDataDirName); err != nil {
			return err
		}
		cmd.Printf("Created rule type test data directory: %s\n", ruleTypeTestDataDirName)
	}

	return nil
}

func validateRuleTypeName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}

	validName := regexp.MustCompile(`^[A-Za-z][-/[:word:]]*$`)

	// regexp to validate name
	if !validName.MatchString(name) {
		return errors.New("name must only contain alphanumeric characters and underscores")
	}

	return nil
}

func assertFilesDontExist(files ...string) error {
	for _, file := range files {
		_, err := os.Stat(file)
		if err == nil {
			return fmt.Errorf("file %s already exists", file)
		}

		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("permission denied for file %s", file)
		}
	}

	return nil
}

func createRuleTypeFile(fileName, name string) error {
	return createFileWithContent(fileName, renderwithRuleTypeName(`---
version: v1
release_phase: alpha
type: rule-type
name: {{ .RuleName }}
display_name:  # Display name for the rule type
short_failure_message:   # Short message to display when the rule fails
severity:
  value: medium
context: {}
description: |  # Description of the rule type
guidance: |  # Guidance for the rule type. This helps users understand how to fix the issue.
def:
  in_entity: repository  # The entity type the rule applies to
  rule_schema: {}
  ingest:
    type: git
    git:
  eval:
    type: rego
    rego:
      type: deny-by-default
      def: |
        package minder

        import rego.v1

        default allow := false

        allow if {
            true
        }
        
        message := "This is a test message"
`, name))
}

func createRuleTypeTestFile(fileName, name string) error {
	return createFileWithContent(fileName, renderwithRuleTypeName(`---
tests:
  - name: "TEST NAME GOES HERE""
    def: {}
    params: {}
    expect: "pass"
    entity: &test-repo
      type: repository
      entity:
        owner: "coolhead"
        name: "haze-wave"
    # When testing a rule, additional content can be supplied
    # from files in the "{{ .RuleName }}.testdata" directory.
    # File paths below are relative to this directory.
    # http:
    #  # Input from the "http" ingest type.
    #  body_file: HTTP_BODY_FILE
    # git:
    #   # Input from the "git" ingest type.  Base paths contain
    #   # directory contents, but do not actually need to be a
    #   # git repository.
    #   repo_base: REPO_BASE_PATH
`, name))
}

func createRuleTypeTestDataDir(dirName string) error {
	if err := os.Mkdir(dirName, 0750); err != nil {
		return fmt.Errorf("error creating directory %s: %w", dirName, err)
	}

	return nil
}

func createFileWithContent(fileName, content string) error {
	return os.WriteFile(fileName, []byte(content), 0644)
}

func renderwithRuleTypeName(templ, name string) string {
	tmpl, err := template.New("ruletype").Parse(templ)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ RuleName string }{name}); err != nil {
		panic(err)
	}

	return buf.String()
}
