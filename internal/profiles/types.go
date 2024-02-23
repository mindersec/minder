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

// Package profiles contains business logic relating to the Profile entity in Minder
package profiles

import (
	"github.com/google/uuid"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// EntityAndRuleTuple is a tuple that allows us track rule instantiations
// and the entity they're associated with
type EntityAndRuleTuple struct {
	Entity minderv1.Entity
	RuleID uuid.UUID
}

// RuleTypeAndNamePair is a tuple of a rule instance's name and rule type name
type RuleTypeAndNamePair struct {
	RuleType string
	RuleName string
}

// RuleMapping is a mapping of rule instance info (name + type)
// to entity info (rule ID + entity type)
type RuleMapping map[RuleTypeAndNamePair]EntityAndRuleTuple
