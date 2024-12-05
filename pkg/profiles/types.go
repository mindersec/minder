// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package profiles contains business logic relating to the Profile entity in Minder
package profiles

import (
	"github.com/google/uuid"
)

// RuleIdAndNamePair is a tuple of a rule's instance ID and the name derived from the rule's
// descriptive name and rule type name
type RuleIdAndNamePair struct {
	RuleID          uuid.UUID
	DerivedRuleName string
}

// RuleTypeAndNamePair is a tuple of a rule instance's name and rule type name
type RuleTypeAndNamePair struct {
	RuleType string
	RuleName string
}

// RuleMapping is a mapping of rule instance info (name + type)
// to entity info (rule ID + entity type)
type RuleMapping map[RuleTypeAndNamePair]RuleIdAndNamePair
