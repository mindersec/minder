// Copyright 2023 Stacklok, Inc
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

// Package schemaupdate contains utility functions to compare two schemas
// for updates
package schemaupdate

import (
	"fmt"
	"maps"

	"dario.cat/mergo"
	"github.com/barkimedes/go-deepcopy"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ValidateSchemaUpdate validates that the new json schema doesn't break
// profiles using this rule type
func ValidateSchemaUpdate(oldRuleSchema *structpb.Struct, newRuleSchema *structpb.Struct) error {
	if len(newRuleSchema.GetFields()) == 0 {
		// If the new schema is empty (including nil), we're good
		// The rule type has removed the schema and profiles
		// won't break
		return nil
	}

	if schemaIsNilOrEmpty(oldRuleSchema) && !schemaIsNilOrEmpty(newRuleSchema) {
		// If old is nil and new is not, we need to verify that
		// the new definition is not introducing required fields
		newrs := newRuleSchema.AsMap()
		if _, ok := newrs["required"]; ok {
			return fmt.Errorf("cannot add required fields to rule schema")
		}

		// If no required fields are being added, we're good
		// profiles using this rule type won't break
		return nil
	}

	oldSchemaMap := oldRuleSchema.AsMap()
	newSchemaMap := newRuleSchema.AsMap()

	oldTypeCast, err := getOrInferType(oldSchemaMap)
	if err != nil {
		return err
	}

	newTypeCast, err := getOrInferType(newSchemaMap)
	if err != nil {
		return err
	}

	if oldTypeCast != newTypeCast {
		return fmt.Errorf("cannot change type of rule schema")
	}

	if oldTypeCast != "object" {
		// the change is fine
		return nil
	}

	// objects need further validation. We need to make sure that
	// the new schema is a superset of the old schema
	if err := validateObjectSchemaUpdate(oldSchemaMap, newSchemaMap); err != nil {
		return err
	}

	return nil
}

func getOrInferType(schemaMap map[string]any) (string, error) {
	typ, ok := schemaMap["type"]
	if !ok {
		return "object", nil
	}

	typCast, ok := typ.(string)
	if !ok {
		return "", fmt.Errorf("invalid type field")
	}

	return typCast, nil
}

func validateObjectSchemaUpdate(oldSchemaMap, newSchemaMap map[string]any) error {
	if err := validateRequired(oldSchemaMap, newSchemaMap); err != nil {
		return err
	}

	if err := validateProperties(oldSchemaMap, newSchemaMap); err != nil {
		return err
	}

	return nil
}

func validateProperties(oldSchemaMap, newSchemaMap map[string]any) error {
	dst, err := deepcopy.Anything(newSchemaMap)
	if err != nil {
		return fmt.Errorf("failed to deepcopy old schema: %v", err)
	}

	castedDst := dst.(map[string]any)

	err = mergo.Merge(&castedDst, &oldSchemaMap, mergo.WithOverride, mergo.WithSliceDeepCopy)
	if err != nil {
		return fmt.Errorf("failed to merge old and new schema: %v", err)
	}

	// The new schema should be a superset of the old schema
	// if it's not, we may break profiles using this rule type
	if !maps.Equal(newSchemaMap, castedDst) {
		return fmt.Errorf("cannot remove properties from rule schema")
	}

	return nil
}

func validateRequired(oldSchemaMap, newSchemaMap map[string]any) error {
	// If the new schema has required fields, we need to make sure
	// that the old schema has those fields as well
	oldRequired, hasOldRequired := oldSchemaMap["required"]
	newRequired, hasNewRequired := newSchemaMap["required"]

	if !hasNewRequired && !hasOldRequired {
		// If we don't have required fields in either schema, we're good
		// profiles using this rule type won't break
		return nil
	}

	if !hasNewRequired && hasOldRequired {
		// If we don't have required fields in the new schema but do
		// in the old schema, we're good.
		// profiles using this rule type won't break
		return nil
	}

	if hasNewRequired && !hasOldRequired {
		// If we have required fields in the new schema but not the old
		// schema, we may break profiles using this rule type
		return fmt.Errorf("cannot add required fields to rule schema")
	}

	oldRequiredSlice, ok := oldRequired.([]interface{})
	if !ok {
		return fmt.Errorf("invalid old required field")
	}

	newRequiredSlice, ok := newRequired.([]interface{})
	if !ok {
		return fmt.Errorf("invalid new required field")
	}

	// We need to make sure that the old required fields are
	// a superset of the new required fields
	oldSet := sets.New(oldRequiredSlice...)
	newSet := sets.New(newRequiredSlice...)
	if !oldSet.IsSuperset(newSet) {
		return fmt.Errorf("cannot add required fields to rule schema")
	}

	return nil
}

func schemaIsNilOrEmpty(schema *structpb.Struct) bool {
	if schema == nil {
		return true
	}

	return len(schema.AsMap()) == 0
}
