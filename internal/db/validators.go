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

package db

func validateActionType(r string, dfl NullActionType) NullActionType {
	switch r {
	case "on":
		return NullActionType{ActionType: ActionTypeOn, Valid: true}
	case "off":
		return NullActionType{ActionType: ActionTypeOff, Valid: true}
	case "dry_run":
		return NullActionType{ActionType: ActionTypeDryRun, Valid: true}
	}

	return dfl
}

// ValidateRemediateType validates the remediate type, defaulting to "off" if invalid
func ValidateRemediateType(r string) NullActionType {
	return validateActionType(r, NullActionType{ActionType: ActionTypeOff, Valid: true})
}

// ValidateAlertType validates the alert type, defaulting to "on" if invalid
func ValidateAlertType(r string) NullActionType {
	return validateActionType(r, NullActionType{ActionType: ActionTypeOn, Valid: true})
}
