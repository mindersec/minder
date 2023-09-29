// Copyright 2023 Stacklok, Inc.
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

package _go

import (
	"errors"
	"fmt"
)

// Validator is an interface which allows for the validation of a struct.
type Validator interface {
	Validate() error
}

// ensure GitHubProviderConfig implements the Validator interface
var _ Validator = (*GitHubProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (_ *GitHubProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	return nil
}

// ensure RESTProviderConfig implements the Validator interface
var _ Validator = (*RESTProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (rpcfg *RESTProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	if rpcfg.GetBaseUrl() == "" {
		return fmt.Errorf("base_url is required")
	}

	return nil
}

// Ensure Entity implements the Validator interface
var _ Validator = (*Entity)(nil)

var (
	// ErrInvalidEntity is returned when an entity is invalid
	ErrInvalidEntity = errors.New("invalid entity")
)

// Validate ensures that an entity is valid
func (entity *Entity) Validate() error {
	if !entity.IsValid() {
		return fmt.Errorf("%w: invalid entity type: %s", ErrInvalidEntity, entity.String())
	}

	return nil
}

var (
	// ErrInvalidRuleType is returned when a rule type is invalid
	ErrInvalidRuleType = errors.New("invalid rule type")
	// ErrInvalidRuleTypeDefinition is returned when a rule type definition is invalid
	ErrInvalidRuleTypeDefinition = errors.New("invalid rule type definition")
)

// Ensure RuleType implements the Validator interface
var _ Validator = (*RuleType)(nil)

// Validate ensures that a rule type is valid
func (rt *RuleType) Validate() error {
	if rt == nil {
		return fmt.Errorf("%w: rule type is nil", ErrInvalidRuleType)
	}

	if rt.Def == nil {
		return fmt.Errorf("%w: rule type definition is nil", ErrInvalidRuleType)
	}

	if err := rt.Def.Validate(); err != nil {
		return errors.Join(ErrInvalidRuleType, err)
	}

	return nil
}

// Validate validates a rule type definition
func (def *RuleType_Definition) Validate() error {
	// if !entities.IsValidEntity(entities.FromString(def.InEntity)) {
	// 	return fmt.Errorf("%w: invalid entity type: %s", ErrInvalidRuleTypeDefinition, def.InEntity)
	// }

	if def.RuleSchema == nil {
		return fmt.Errorf("%w: rule schema is nil", ErrInvalidRuleTypeDefinition)
	}

	if def.Ingest == nil {
		return fmt.Errorf("%w: data ingest is nil", ErrInvalidRuleTypeDefinition)
	}

	if def.Eval == nil {
		return fmt.Errorf("%w: data eval is nil", ErrInvalidRuleTypeDefinition)
	}

	return nil
}
