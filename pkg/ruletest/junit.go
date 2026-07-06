// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"encoding/xml"
	"maps"
	"slices"
	"strings"
)

// JUnitTestSuites is the root element of a JUnit XML report.
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single test suite in a JUnit XML report.
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case in a JUnit XML report.
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

// JUnitFailure represents a failure within a test case.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Body    string `xml:",chardata"`
}

// AsJUnit converts test results into a JUnitTestSuites structure.
func AsJUnit(results []TestResult) JUnitTestSuites {
	suitesMap := make(map[string]*JUnitTestSuite)

	for _, res := range results {
		suiteName := res.Filename
		if _, ok := suitesMap[suiteName]; !ok {
			suitesMap[suiteName] = &JUnitTestSuite{
				Name: suiteName,
			}
		}

		suite := suitesMap[suiteName]
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

		suite.TestCases = append(suite.TestCases, tc)
	}

	var suitesList []JUnitTestSuite
	for _, suite := range slices.Collect(maps.Values(suitesMap)) {
		suitesList = append(suitesList, *suite)
	}

	return JUnitTestSuites{
		TestSuites: suitesList,
	}
}
