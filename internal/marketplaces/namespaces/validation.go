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

// Package namespaces contains logic relating to the namespacing of Rule Types
// and Profiles
package namespaces

import (
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// these functions are tested through the tests for RuleTypeService

// ValidateNamespacedNameRules takes a name for a new profile or rule type and
// asserts that:
// A) If the subscriptionID is empty, there name should not be namespaced
// B) If subscriptionID is not empty, the name must be namespaced
// This assumes the name has already been validated against the other
// validation rules for profile and rule type names.
func ValidateNamespacedNameRules(name string, subscriptionID uuid.UUID) error {
	hasNamespace := strings.Contains(name, "/")
	if hasNamespace && subscriptionID == uuid.Nil {
		return errors.New("cannot create a rule type or profile with a namespace through the API")
	} else if !hasNamespace && subscriptionID != uuid.Nil {
		return errors.New("rule types and profiles from subscriptions must have namespaced names")
	}

	// in future, we may want to check that the namespace in the profile/rule
	// name is the same as the one in the subscription bundle
	return nil
}

// DoesSubscriptionIDMatch takes a subscription ID from the database, and
// compares it with the subscriptionID parameter. It asserts that:
// A) If the subscription ID from the DB is not null, that it is equal to
//
//	subscription ID.
//
// B) If the subscription ID from the DB is null, the subscriptionID parameterß
//
//	must be equal to uuid.Nil
//
// This logic is intended to check if the subscription ID associated with a
// rule type or profile matches a given subscription ID.
func DoesSubscriptionIDMatch(subscriptionID uuid.UUID, dbSubscriptionID uuid.NullUUID) error {
	// In theory, we could include the subscription ID in the GetRuleType query
	// but this would mean that we would not be able to differentiate between
	// a row which does not exist, and this case where the subscription ID is
	// wrong. This distinction is useful for error reporting purposes.
	if dbSubscriptionID.Valid && dbSubscriptionID.UUID != subscriptionID {
		return errors.New("attempted to edit a rule type or profile which belongs to a bundle")
	} else if !dbSubscriptionID.Valid && subscriptionID != uuid.Nil {
		return errors.New("attempted to edit a customer rule type or profile with bundle operation")
	}
	return nil
}

// ValidateLabelsPresence makes sure that only profiles that belong to a subscription
// bundle have labels
func ValidateLabelsPresence(labels []string, subscriptionID uuid.UUID) error {
	if subscriptionID == uuid.Nil && len(labels) > 0 {
		return errors.New("labels can only be applied to profiles from a subscription bundle")
	}
	return nil
}

// ValidateLabelsUpdate ensures that labels cannot be updated
func ValidateLabelsUpdate(labels, dbLabels []string) error {
	isEqual := slices.Equal(labels, dbLabels)
	if !isEqual {
		return errors.New("labels cannot be updated")
	}
	return nil
}
