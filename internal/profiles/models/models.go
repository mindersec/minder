// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package models contains domain models for profiles
package models

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ProfileAggregate represents a profile along with rule instances
type ProfileAggregate struct {
	ID           uuid.UUID
	Name         string
	ActionConfig ActionConfiguration
	Rules        []RuleInstance
}

// ActionConfiguration stores the configuration state for a profile
type ActionConfiguration struct {
	Remediate ActionOpt
	Alert     ActionOpt
}

// RuleInstance is a domain-level model of a rule instance
type RuleInstance struct {
	ID         uuid.UUID
	Name       string
	Def        map[string]any
	Params     map[string]any
	RuleTypeID uuid.UUID
}

// RuleFromPB converts a protobuf rule instance to the domain model
func RuleFromPB(
	ruleTypeID uuid.UUID,
	pbRule *minderv1.Profile_Rule,
) RuleInstance {
	return RuleInstance{
		ID:         uuid.Nil, // When converting from PB, we do not care about this value
		Name:       pbRule.Name,
		Def:        pbRule.Def.AsMap(),
		Params:     pbRule.Params.AsMap(),
		RuleTypeID: ruleTypeID,
	}
}

// RuleFromDB converts a DB schema rule instance to the domain model
func RuleFromDB(rule db.RuleInstance) (RuleInstance, error) {
	// deserialize the defs/params
	var def map[string]any
	if err := json.Unmarshal(rule.Def, &def); err != nil {
		return RuleInstance{}, fmt.Errorf("unable to deserialize rule def: %w", err)
	}

	var params map[string]any
	if err := json.Unmarshal(rule.Params, &params); err != nil {
		return RuleInstance{}, fmt.Errorf("unable to deserialize rule params: %w", err)
	}

	return RuleInstance{
		ID:         rule.ID,
		Name:       rule.Name,
		Def:        def,
		Params:     params,
		RuleTypeID: rule.RuleTypeID,
	}, nil
}

// ActionOpt is the type that defines what action to take when remediating
type ActionOpt int

const (
	// ActionOptOn means perform the remediation
	ActionOptOn ActionOpt = iota
	// ActionOptOff means do not perform the remediation
	ActionOptOff
	// ActionOptDryRun means perform a dry run of the remediation
	ActionOptDryRun
	// ActionOptUnknown means the action is unknown. This is a sentinel value.
	ActionOptUnknown
)

func (a ActionOpt) String() string {
	return [...]string{"on", "off", "dry_run", "unknown"}[a]
}

// ActionOptFromString returns the ActionOpt from a string representation
func ActionOptFromString(s *string, defAction ActionOpt) ActionOpt {
	var actionOptMap = map[string]ActionOpt{
		"on":      ActionOptOn,
		"off":     ActionOptOff,
		"dry_run": ActionOptDryRun,
	}

	if s == nil {
		return defAction
	}

	if v, ok := actionOptMap[*s]; ok {
		return v
	}

	return ActionOptUnknown
}

// ActionOptFromDB converts the db representation of action type to ActionOpt
func ActionOptFromDB(dbState db.NullActionType) ActionOpt {
	if !dbState.Valid {
		return ActionOptUnknown
	}

	switch dbState.ActionType {
	case db.ActionTypeOn:
		return ActionOptOn
	case db.ActionTypeOff:
		return ActionOptOff
	case db.ActionTypeDryRun:
		return ActionOptDryRun
	default:
		return ActionOptUnknown
	}
}
