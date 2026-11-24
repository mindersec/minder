// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletypes

import (
	"testing"

	"github.com/stretchr/testify/require"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func ruleTypeDefWith(evalType, ingestType string) *pb.RuleType {
	return &pb.RuleType{
		Def: &pb.RuleType_Definition{
			Eval:   &pb.RuleType_Definition_Eval{Type: evalType},
			Ingest: &pb.RuleType_Definition_Ingest{Type: ingestType},
		},
	}
}

func TestValidateEvalAndIngestTypes(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		RuleType      *pb.RuleType
		ExpectedError string
	}{
		{
			Name:     "rest ingest with jq eval passes",
			RuleType: ruleTypeDefWith("jq", pb.IngestTypeRest),
		},
		{
			Name:     "diff ingest with rego eval passes",
			RuleType: ruleTypeDefWith("rego", pb.IngestTypeDiff),
		},
		{
			Name:     "vulncheck eval passes",
			RuleType: ruleTypeDefWith("vulncheck", pb.IngestTypeRest),
		},
		{
			Name:     "trusty eval passes",
			RuleType: ruleTypeDefWith("trusty", pb.IngestTypeRest),
		},
		{
			Name:     "homoglyphs eval passes",
			RuleType: ruleTypeDefWith("homoglyphs", pb.IngestTypeRest),
		},
		{
			Name:     "builtin ingest passes",
			RuleType: ruleTypeDefWith("jq", "builtin"),
		},
		{
			Name:     "artifact ingest passes",
			RuleType: ruleTypeDefWith("jq", "artifact"),
		},
		{
			Name:     "git ingest passes",
			RuleType: ruleTypeDefWith("jq", "git"),
		},
		{
			Name:     "deps ingest passes",
			RuleType: ruleTypeDefWith("jq", "deps"),
		},
		{
			Name:          "unknown eval type fails",
			RuleType:      ruleTypeDefWith("nonexistent", pb.IngestTypeRest),
			ExpectedError: "unsupported eval type: nonexistent",
		},
		{
			Name:          "unknown ingest type fails",
			RuleType:      ruleTypeDefWith("jq", "nonexistent"),
			ExpectedError: "unsupported ingest type: nonexistent",
		},
		{
			Name:          "empty eval type fails",
			RuleType:      ruleTypeDefWith("", pb.IngestTypeRest),
			ExpectedError: "unsupported eval type",
		},
		{
			Name:          "empty ingest type fails",
			RuleType:      ruleTypeDefWith("jq", ""),
			ExpectedError: "unsupported ingest type",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			err := validateEvalAndIngestTypes(scenario.RuleType)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}
