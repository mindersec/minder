// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"cmp"
	"encoding/xml"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// JUnitTestSuites is the root element of a JUnit XML report.
type JUnitTestSuites struct {
	XMLName  xml.Name `xml:"testsuites"`
	Tests    int      `xml:"tests,attr,omitempty"`
	Errors   int      `xml:"errors,attr,omitempty"`
	Failures int      `xml:"failures,attr,omitempty"`
	Skipped  int      `xml:"skipped,attr,omitempty"`
	Disabled int      `xml:"disabled,attr,omitempty"`

	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single test suite in a JUnit XML report.
type JUnitTestSuite struct {
	Name     string `xml:"name,attr"`
	Tests    int    `xml:"tests,attr"`
	Errors   int    `xml:"errors,attr"`
	Failures int    `xml:"failures,attr"`
	Skipped  int    `xml:"skipped,attr,omitempty"`
	Disabled int    `xml:"disabled,attr,omitempty"`

	File string `xml:"file,attr,omitempty"`

	Properties *[]Property `xml:"properties>property,omitempty"`

	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case in a JUnit XML report.
type JUnitTestCase struct {
	Name      string `xml:"name,attr"`
	ClassName string `xml:"classname,attr"`

	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitFailure `xml:"error,omitempty"`
	Skipped   *JUnitFailure `xml:"skipped,omitempty"`
	SystemOut *Output       `xml:"system-out,omitempty"`
	SystemErr *Output       `xml:"system-err,omitempty"`
}

// JUnitFailure represents a failure within a test case.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Body    string `xml:",chardata"`
}

// Property represents a key-value pair in a JUnit XML report (used for coverage).
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Output represents the text output from a test case (currently not captured).
type Output struct {
	Body string `xml:",cdata"`
}

// AsJUnit converts test results into a JUnitTestSuites structure.
func AsJUnit(results []TestRun) JUnitTestSuites {
	suitesMap := make(map[string]*JUnitTestSuite)

	for _, run := range results {
		for _, res := range run.Results {
			suiteName := res.Filename
			filePath := filepath.Join(run.BaseDir, suiteName)
			if _, ok := suitesMap[filePath]; !ok {
				suitesMap[filePath] = &JUnitTestSuite{
					Name: suiteName,
					File: filePath,
					Properties: &[]Property{
						{
							Name:  "coverage.statements.pct",
							Value: "100", // Claim 100% coverage if tests cover a rule at all
						},
					},
				}
			}

			suite := suitesMap[filePath]
			suite.Tests++

			tc := JUnitTestCase{
				Name:      res.Name,
				ClassName: suiteName,
			}

			if len(res.Failures) > 0 {
				suite.Failures++
				tc.Failure = &JUnitFailure{
					Message: "Test failed",
					Body:    strings.Join(res.Failures, "\n"),
				}
			}
			if len(res.Errors) > 0 {
				suite.Errors++
				tc.Error = &JUnitFailure{
					Message: "Test error",
					Body:    strings.Join(res.Errors, "\n"),
				}
			}

			suite.TestCases = append(suite.TestCases, tc)
		}
		// See discussion in #6749: we want to record _something_ for ruletypes that don't
		// have a corresponding test, but Tests > 0 is misleading.  The two options are:
		// 1. Make up a suite and put each ruletype in as a testcase
		// 2. Put each ruletype in its own suite
		// We have chosen (2), as it makes it easier to understand the summary statistics like Test:0
		for _, rule := range run.UncoveredRules() {
			suite := &JUnitTestSuite{
				// We use a different format from test suites to indicate ruletypes that don't have any tests.
				Name: fmt.Sprintf("no-tests:%s", rule),
				File: run.BaseDir + string(filepath.Separator), // TODO: do we want to keep and pass along a ruletype -> file map?
				Properties: &[]Property{
					{
						Name:  "coverage.statements.pct",
						Value: "0",
					},
				},
			}
			suitesMap[suite.Name] = suite
		}
	}

	res := JUnitTestSuites{}

	for _, suite := range slices.SortedFunc(
		maps.Values(suitesMap),
		func(a, b *JUnitTestSuite) int {
			return cmp.Or(cmp.Compare(a.File, b.File), cmp.Compare(a.Name, b.Name))
		},
	) {
		res.TestSuites = append(res.TestSuites, *suite)
		res.Tests += suite.Tests
		res.Failures += suite.Failures
		res.Errors += suite.Errors
		res.Skipped += suite.Skipped
		res.Disabled += suite.Disabled
	}

	return res
}
