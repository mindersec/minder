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
	"reflect"

	"dario.cat/mergo"
	"github.com/barkimedes/go-deepcopy"
	"github.com/google/go-cmp/cmp"
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

	if oldTypeCast != "object" && oldTypeCast != "array" {
		// the change is fine
		return nil
	}

	if oldTypeCast == "array" {
		return validateArraySchemaUpdate(oldSchemaMap, newSchemaMap)
	}

	// objects need further validation. We need to make sure that
	// the new schema is a superset of the old schema
	return validateObjectSchemaUpdate(oldSchemaMap, newSchemaMap)
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
		return fmt.Errorf("failed to validate required fields: %v", err)
	}

	if err := validateProperties(oldSchemaMap, newSchemaMap); err != nil {
		return fmt.Errorf("failed to validate properties: %v", err)
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

	// We need to ignore the description field when comparing the old and new schema to allow
	// to update the ruletype text. We also need to ignore changing defaults as they are advisory
	// for the UI at the moment
	opts := []cmp.Option{
		cmp.FilterPath(isScalarDescription, cmp.Ignore()),
		cmp.FilterPath(isDefaultValue, cmp.Ignore()),
	}

	// The new schema should be a superset of the old schema
	// if it's not, we may break profiles using this rule type
	if !cmp.Equal(newSchemaMap, castedDst, opts...) {
		return fmt.Errorf("cannot remove properties from rule schema")
	}

	return nil
}

func isScalarDescription(p cmp.Path) bool {
	if mi, ok := p.Last().(cmp.MapIndex); ok {
		key := mi.Key()
		left, right := mi.Values()
		// we can ignore description if it was a string and is a string
		if key.String() == "description" && isValueString(left) && isValueString(right) {
			return true
		}
	}
	return false
}

func isDefaultValue(p cmp.Path) bool {
	if mi, ok := p.Last().(cmp.MapIndex); ok {
		key := mi.Key()
		// we ignore default if it has a type sibling, assuming that this is a type default, not an attribute
		// named default. Further down we also check that the type has a string value
		if key.String() == "default" {
			if hasTypeSibling(p.Index(len(p) - 2)) {
				return true
			}
		}
	}

	return false
}

func hasTypeSibling(p cmp.PathStep) bool {
	left, right := p.Values()
	return isMapWithKey(left, "type") && isMapWithKey(right, "type")
}

func isMapWithKey(value reflect.Value, key string) bool {
	if !value.IsValid() {
		return false
	}

	if !value.CanInterface() {
		return false
	}

	valIf := value.Interface()
	valMap, ok := valIf.(map[string]any)
	if !ok {
		return false
	}

	for k := range valMap {
		if k == key {
			return true
		}
	}

	return false
}

func isValueString(value reflect.Value) bool {
	if !value.IsValid() {
		return false
	}

	if !value.CanInterface() {
		return false
	}

	valIf := value.Interface()
	if _, ok := valIf.(string); ok {
		return true
	}
	return false
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

func validateArraySchemaUpdate(oldSchemaMap, newSchemaMap map[string]any) error {
	if err := validateItems(oldSchemaMap, newSchemaMap); err != nil {
		return fmt.Errorf("failed to validate items: %v", err)
	}

	return nil
}

func validateItems(oldSchemaMap, newSchemaMap map[string]any) error {
	oldItems, hasOldItems := oldSchemaMap["items"]
	newItems, hasNewItems := newSchemaMap["items"]

	if !hasNewItems || !hasOldItems {
		// If we don't have items in either schema, we're good
		// profiles using this rule type won't break
		return fmt.Errorf("cannot remove items from rule schema")
	}

	oldItemsMap, ok := oldItems.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid old items field")
	}

	newItemsMap, ok := newItems.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid new items field")
	}

	// The new schema should be a superset of the old schema
	// if it's not, we may break profiles using this rule type
	if !reflect.DeepEqual(newItemsMap, oldItemsMap) {
		return fmt.Errorf("cannot change items type of rule schema")
	}

	return nil
}
