// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ruletypes

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stacklok/minder/internal/db"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleDefFromDB converts a rule type definition from the database to a protobuf
// rule type definition
func RuleDefFromDB(r *db.RuleType) (*minderv1.RuleType_Definition, error) {
	def := &minderv1.RuleType_Definition{}

	if err := protojson.Unmarshal(r.Definition, def); err != nil {
		return nil, fmt.Errorf("cannot unmarshal rule type definition: %w", err)
	}
	return def, nil
}

// RuleTypePBFromDB converts a rule type from the database to a protobuf
// rule type
func RuleTypePBFromDB(rt *db.RuleType) (*minderv1.RuleType, error) {
	def, err := RuleDefFromDB(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot get rule type definition: %w", err)
	}

	id := rt.ID.String()
	project := rt.ProjectID.String()

	var seval minderv1.Severity_Value
	if err := seval.FromString(string(rt.SeverityValue)); err != nil {
		seval = minderv1.Severity_VALUE_UNKNOWN
	}

	displayName := rt.DisplayName
	if displayName == "" {
		displayName = rt.Name
	}

	var releasePhase minderv1.RuleTypeReleasePhase
	if err := releasePhase.FromString(string(rt.ReleasePhase)); err != nil {
		releasePhase = minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED
	}

	// TODO: (2024/03/28) this is for compatibility with old CLI versions that expect provider, remove this eventually
	noProvider := ""
	return &minderv1.RuleType{
		Id:          &id,
		Name:        rt.Name,
		DisplayName: displayName,
		Context: &minderv1.Context{
			Provider: &noProvider,
			Project:  &project,
		},
		Description: rt.Description,
		Guidance:    rt.Guidance,
		Def:         def,
		Severity: &minderv1.Severity{
			Value: seval,
		},
		ReleasePhase: releasePhase,
	}, nil
}
