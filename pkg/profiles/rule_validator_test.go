// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles"
)

func TestSetDefaultValuesOnValidation(t *testing.T) {
	t.Parallel()

	rtstr := `
---
version: v1
release_phase: alpha
type: rule-type
name: foo
display_name: Foo
short_failure_message: Foo failed
severity:
  value: medium
context:
  provider: github
description: Very important rule
guidance: |
  This is how you should do it.
def:
  in_entity: repository
  rule_schema:
    type: object
    properties:
      schedule_interval:
        type: string
        description: |
          Sets the schedule interval in cron format for the workflow. Only applicable for remediation.
      publish_results:
        type: boolean
        description: |
          Publish the results of the analysis.
        default: true
      retention_days:
        type: integer
        description: |
          Number of days to retain the SARIF file.
        default: 5
      sarif_file:
        type: string
        description: |
          Name of the SARIF file.
        default: "results.sarif"
    required:
      - schedule_interval
      - publish_results
`

	rt := &minderv1.RuleType{}
	require.NoError(t, minderv1.ParseResource(strings.NewReader(rtstr), rt), "failed to parse rule type")

	rval, err := profiles.NewRuleValidator(rt)
	require.NoError(t, err, "failed to create rule validator")

	obj := map[string]any{
		"schedule_interval": "0 0 * * *",
		"publish_results":   false,
		"retention_days":    10,
	}

	// Validation should pass
	require.NoError(t, rval.ValidateRuleDefAgainstSchema(obj), "failed to validate rule definition")

	// Value is left as is
	require.Equal(t, "0 0 * * *", obj["schedule_interval"])
	require.Equal(t, 10, obj["retention_days"])
	require.Equal(t, false, obj["publish_results"])

	// default is set
	require.Equal(t, "results.sarif", obj["sarif_file"])
}

func TestExampleRulesAreValidatedCorrectly(t *testing.T) {
	t.Parallel()

	t.Log("parsing example profile")
	pol, err := profiles.ReadProfileFromFile("../../examples/rules-and-profiles/profiles/github/profile.yaml")
	require.NoError(t, err, "failed to parse example profile, make sure to do - make init-examples")

	// open rules in example directory
	err = filepath.Walk("../../examples/rules-and-profiles/rule-types/github", func(path string, info os.FileInfo, _ error) error {
		// skip directories
		if info.IsDir() {
			return nil
		}

		// skip non-yaml files
		if filepath.Ext(path) != ".yaml" {
			return nil
		}

		// skip test files
		if strings.HasSuffix(path, ".test.yaml") || strings.HasSuffix(path, ".test.yml") || strings.Contains(filepath.Dir(path), ".testdata") {
			return nil
		}

		fname := filepath.Base(path)
		t.Run(fname, func(t *testing.T) {
			t.Parallel()

			// open file
			//nolint:gosec // this is a test
			f, err := os.Open(path)
			require.NoError(t, err, "failed to open file %s", path)
			defer f.Close()

			t.Log("parsing rule type", path)
			rt := &minderv1.RuleType{}
			require.NoError(t, minderv1.ParseResource(f, rt), "failed to parse rule type %s", path)
			require.NotNil(t, rt, "failed to parse rule type %s", path)

			require.NoError(t, rt.Validate(), "failed to validate rule type %s", path)

			t.Log("creating rule validator")
			rval, err := profiles.NewRuleValidator(rt)
			require.NoError(t, err, "failed to create rule validator for rule type %s", path)

			rules, err := profiles.GetRulesFromProfileOfType(pol, rt)
			require.NoError(t, err, "failed to get rules from profile for rule type %s", path)

			t.Log("validating rules")
			for _, ruleCall := range rules {
				err := rval.ValidateRuleDefAgainstSchema(ruleCall.Def.AsMap())
				require.NoError(t, err, "failed to validate rule definition for rule type %s", path)

				err = rval.ValidateParamsAgainstSchema(ruleCall.GetParams().AsMap())
				require.NoError(t, err, "failed to validate rule parameters for rule type %s", path)
			}

		})

		return nil
	})
	require.NoError(t, err, "failed to walk rule types directory")
}
