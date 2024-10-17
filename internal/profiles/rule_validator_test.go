// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/profiles"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

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

		fname := filepath.Base(path)
		t.Run(fname, func(t *testing.T) {
			t.Parallel()

			// open file
			//nolint:gosec // this is a test
			f, err := os.Open(path)
			require.NoError(t, err, "failed to open file %s", path)
			defer f.Close()

			t.Log("parsing rule type", path)
			rt, err := minderv1.ParseRuleType(f)
			require.NoError(t, err, "failed to parse rule type %s", path)
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
