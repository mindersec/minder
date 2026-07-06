// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"testing"
)

func TestAsJUnit_PassingResultsMultipleSuites(t *testing.T) {
	t.Parallel()
	results := []TestResult{
		{Filename: "file_a.star", Name: "test_one"},
		{Filename: "file_a.star", Name: "test_two"},
		{Filename: "file_b.star", Name: "test_three"},
	}

	suites := AsJUnit(results)

	if len(suites.TestSuites) != 2 {
		t.Fatalf("expected 2 suites, got %d", len(suites.TestSuites))
	}

	suiteMap := make(map[string]JUnitTestSuite)
	for _, s := range suites.TestSuites {
		suiteMap[s.Name] = s
	}

	a := suiteMap["file_a.star"]
	if a.Tests != 2 {
		t.Errorf("file_a.star: expected 2 tests, got %d", a.Tests)
	}
	if a.Failures != 0 {
		t.Errorf("file_a.star: expected 0 failures, got %d", a.Failures)
	}

	b := suiteMap["file_b.star"]
	if b.Tests != 1 {
		t.Errorf("file_b.star: expected 1 test, got %d", b.Tests)
	}
	if b.Failures != 0 {
		t.Errorf("file_b.star: expected 0 failures, got %d", b.Failures)
	}
}

func TestAsJUnit_FailuresAggregated(t *testing.T) {
	t.Parallel()
	results := []TestResult{
		{Filename: "suite.star", Name: "test_fail_one", Failures: []string{"err1", "err2"}},
		{Filename: "suite.star", Name: "test_fail_two", Failures: []string{"err3"}},
		{Filename: "suite.star", Name: "test_pass"},
	}

	suites := AsJUnit(results)

	if len(suites.TestSuites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.TestSuites))
	}

	suite := suites.TestSuites[0]
	if suite.Tests != 3 {
		t.Errorf("expected 3 tests, got %d", suite.Tests)
	}
	if suite.Failures != 2 {
		t.Errorf("expected 2 failures, got %d", suite.Failures)
	}

	tcMap := make(map[string]JUnitTestCase)
	for _, tc := range suite.TestCases {
		tcMap[tc.Name] = tc
	}

	failOne := tcMap["test_fail_one"]
	if failOne.Failure == nil {
		t.Fatal("test_fail_one: expected a failure")
	}
	if failOne.Failure.Body != "err1\nerr2" {
		t.Errorf("test_fail_one: expected joined failures, got %q", failOne.Failure.Body)
	}

	pass := tcMap["test_pass"]
	if pass.Failure != nil {
		t.Error("test_pass: expected no failure")
	}
}

func TestAsJUnit_EmptyInput(t *testing.T) {
	t.Parallel()
	suites := AsJUnit(nil)

	if len(suites.TestSuites) != 0 {
		t.Errorf("expected 0 suites, got %d", len(suites.TestSuites))
	}
}
