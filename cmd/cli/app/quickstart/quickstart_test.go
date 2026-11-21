// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quickstart

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestSummarizeResources(t *testing.T) {
	t.Parallel()

	rt := &minderv1.RuleType{
		Name:        "secret_scanning",
		DisplayName: "Secret Scanning",
		Description: "Prevent secret leaks in commits",
	}
	p := &minderv1.Profile{
		Name:        "security-baseline",
		DisplayName: "Security Baseline",
		Remediate:   proto.String("on"),
		Alert:       proto.String("on"),
	}
	ds := &minderv1.DataSource{
		Name: "test_datasource",
	}

	resources := []minderv1.ResourceMeta{rt, p, ds}
	summary := summarizeResources(resources)

	assert.Len(t, summary.RuleTypes, 1)
	assert.Equal(t, "secret_scanning", summary.RuleTypes[0].Name)
	assert.Equal(t, "Secret Scanning", summary.RuleTypes[0].DisplayName)
	assert.Equal(t, "Prevent secret leaks in commits", summary.RuleTypes[0].Description)

	assert.Len(t, summary.Profiles, 1)
	assert.Equal(t, "security-baseline", summary.Profiles[0].Name)
	assert.Equal(t, "Security Baseline", summary.Profiles[0].DisplayName)
	assert.Equal(t, "on", summary.Profiles[0].Remediate)
	assert.Equal(t, "on", summary.Profiles[0].Alert)
}

func TestDescribePolicy(t *testing.T) {
	t.Parallel()

	summary := PolicySummary{
		Profiles: []ProfileDetail{
			{
				Name:        "security-baseline",
				DisplayName: "Security Baseline",
				Remediate:   "on",
				Alert:       "on",
			},
		},
		RuleTypes: []RuleTypeDetail{
			{
				Name:        "secret_scanning",
				DisplayName: "Secret Scanning",
				Description: "Prevent secret leaks",
			},
		},
	}

	desc := describePolicy("https://github.com/mindersec/minder-rules-and-profiles?profiles/github", summary)

	assert.Contains(t, desc, "Step 1 - Review Security Baseline Policy")
	assert.Contains(t, desc, "Source: https://github.com/mindersec/minder-rules-and-profiles?profiles/github")
	assert.Contains(t, desc, "Security Baseline (Remediate: on, Alert: on)")
	assert.Contains(t, desc, "Secret Scanning: Prevent secret leaks")
	assert.NotContains(t, desc, "Data Sources:")
}

func TestSummarizeResources_Empty(t *testing.T) {
	t.Parallel()

	summary := summarizeResources([]minderv1.ResourceMeta{})
	assert.Empty(t, summary.RuleTypes)
	assert.Empty(t, summary.Profiles)
}

func TestDescribePolicy_Empty(t *testing.T) {
	t.Parallel()

	summary := PolicySummary{}
	desc := describePolicy("https://example.com/repo", summary)

	assert.Contains(t, desc, "Source: https://example.com/repo")
	assert.NotContains(t, desc, "Profiles:")
	assert.NotContains(t, desc, "Rule Types:")
	assert.NotContains(t, desc, "Data Sources:")
}

func TestFetchPolicyResources_InvalidURL(t *testing.T) {
	t.Parallel()

	_, _, err := fetchPolicyResources(context.Background(), nil, "invalid-url-without-scheme")
	assert.Error(t, err)
}
