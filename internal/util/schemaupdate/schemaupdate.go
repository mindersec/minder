// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	oldProperties, hasOldProperties := oldSchemaMap["properties"]
	newProperties, hasNewProperties := newSchemaMap["properties"]

	if !hasNewProperties || !hasOldProperties {
		return fmt.Errorf("cannot remove properties from object type rule schema")
	}

	oldPropertiesMap, ok := oldProperties.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid old properties field")
	}
	newPropertiesMap, ok := newProperties.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid new properties field")
	}

	// copy new schema to avoid modifying the original
	mergedSchema, err := copySchema(newPropertiesMap)
	if err != nil {
		return fmt.Errorf("failed to copy new schema: %v", err)
	}

	// Merge the old schema into the new schema.
	// The merged schema should equal the new schema if the old schema
	// is a subset of the new schema
	err = mergo.Merge(&mergedSchema, &oldPropertiesMap, mergo.WithOverride, mergo.WithSliceDeepCopy)
	if err != nil {
		return fmt.Errorf("failed to merge old and new schema: %v", err)
	}

	// The new schema should be a superset of the old schema
	// if it's not, we may break profiles using this rule type
	// The mergedSchema is the new schema with the old schema merged in
	// so we can compare it to the new schema directly
	if !schemasAreEqual(mergedSchema, newPropertiesMap) {
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

	// If we don't have required fields in either schema, we're good
	if !hasNewRequired && !hasOldRequired {
		// If we don't have required fields in either schema, we're good
		// profiles using this rule type won't break
		return nil
	}

	// If the new schema doesn't have required fields, but the old schema does,
	// we're good
	if !hasNewRequired && hasOldRequired {
		// If we don't have required fields in the new schema but do
		// in the old schema, we're good.
		// profiles using this rule type won't break
		return nil
	}

	// If the new schema has required fields, but the old schema doesn't,
	// we may break profiles using this rule type
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
	if !requiredIsSuperset(oldRequiredSlice, newRequiredSlice) {
		return fmt.Errorf("cannot add required fields to rule schema")
	}

	return nil
}

func requiredIsSuperset(oldRequired, newRequired []interface{}) bool {
	oldSet := sets.New(oldRequired...)
	newSet := sets.New(newRequired...)

	return oldSet.IsSuperset(newSet)
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
func copySchema(s map[string]any) (map[string]any, error) {
	dst, err := deepcopy.Anything(s)
	if err != nil {
		return nil, fmt.Errorf("failed to deepcopy: %v", err)
	}

	castedDst, ok := dst.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to cast schema to map")
	}
	return castedDst, nil
}

func schemasAreEqual(a, b map[string]any) bool {
	// We need to ignore the description field when comparing the old and new schema to allow
	// to update the ruletype text. We also need to ignore changing defaults as they are advisory
	// for the UI at the moment
	return cmp.Equal(a, b,
		cmp.FilterPath(isScalarDescription, cmp.Ignore()),
		cmp.FilterPath(isDefaultValue, cmp.Ignore()),
	)
}
