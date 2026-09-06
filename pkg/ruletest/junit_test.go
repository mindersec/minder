// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAsJUnit_PassingResultsMultipleSuites(t *testing.T) {
	t.Parallel()

	coverage100 := []Property{{
		Name:  "coverage.statements.pct",
		Value: "100",
	}}

	tests := []struct {
		name     string
		results  []TestRun
		expected JUnitTestSuites
	}{{
		name: "passing results multiple suites",
		results: []TestRun{
			{
				BaseDir:     "foo",
				LoadedRules: []string{"rule_a", "rule_b"},
				Results: []TestResult{
					{Filename: "file_a.star", Name: "test_one", EvaluatedRules: map[string]struct{}{"rule_a": {}}},
					{Filename: "file_a.star", Name: "test_two", EvaluatedRules: map[string]struct{}{"rule_a": {}}},
					{Filename: "file_b.star", Name: "test_three", EvaluatedRules: map[string]struct{}{"rule_b": {}}},
				},
			},
			{
				BaseDir:     "bar",
				LoadedRules: []string{"rule_c", "rule_d"},
				Results: []TestResult{
					{Filename: "file_b.star", Name: "test_one", EvaluatedRules: map[string]struct{}{"rule_c": {}}},
				},
			},
		},
		expected: JUnitTestSuites{
			Tests: 4,
			TestSuites: []JUnitTestSuite{
				{
					File:       "foo/file_a.star",
					Name:       "file_a.star",
					Tests:      2,
					Properties: &coverage100,
					TestCases: []JUnitTestCase{
						{Name: "test_one", ClassName: "file_a.star"},
						{Name: "test_two", ClassName: "file_a.star"},
					},
				},
				{
					File:       "foo/file_b.star",
					Name:       "file_b.star",
					Tests:      1,
					Properties: &coverage100,
					TestCases:  []JUnitTestCase{{Name: "test_three", ClassName: "file_b.star"}},
				},
				{
					File:       "bar/file_b.star",
					Name:       "file_b.star",
					Tests:      1,
					Properties: &coverage100,
					TestCases:  []JUnitTestCase{{Name: "test_one", ClassName: "file_b.star"}},
				},
				{
					File: "bar",
					Name: "bar",
					Properties: &[]Property{{
						Name:  "coverage.statements.pct",
						Value: "0",
					}},
					TestCases: []JUnitTestCase{
						{
							Name: "rule_d",
						},
					},
				},
			},
		},
	}, {
		name: "failure aggregation",
		results: []TestRun{
			{
				BaseDir:     "suite",
				LoadedRules: []string{},
				Results: []TestResult{
					{Filename: "suite.star", Name: "test_fail_one", Failures: []string{"err1", "err2"}},
					{Filename: "suite.star", Name: "test_fail_two", Failures: []string{"err3"}},
					{Filename: "suite.star", Name: "test_pass"},
					{Filename: "other.star", Name: "test_other", Errors: []string{"err4"}},
				},
			},
		},
		expected: JUnitTestSuites{
			Tests:    4,
			Failures: 2,
			Errors:   1,
			TestSuites: []JUnitTestSuite{
				{
					File:       "suite/suite.star",
					Name:       "suite.star",
					Tests:      3,
					Failures:   2,
					Properties: &coverage100,
					TestCases: []JUnitTestCase{
						{
							Name: "test_fail_one", ClassName: "suite.star",
							Failure: &JUnitFailure{Message: "Test failed", Body: "err1\nerr2"},
						},
						{
							Name: "test_fail_two", ClassName: "suite.star",
							Failure: &JUnitFailure{Message: "Test failed", Body: "err3"},
						},
						{Name: "test_pass", ClassName: "suite.star"},
					},
				},
				{
					File:       "suite/other.star",
					Name:       "other.star",
					Tests:      1,
					Errors:     1,
					Properties: &coverage100,
					TestCases: []JUnitTestCase{
						{
							Name: "test_other", ClassName: "other.star",
							Error: &JUnitFailure{Message: "Test error", Body: "err4"},
						},
					},
				},
			},
		},
	}, {
		name:     "empty input",
		results:  nil,
		expected: JUnitTestSuites{},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			suites := AsJUnit(tt.results)

			if diff := cmp.Diff(tt.expected, suites); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}
