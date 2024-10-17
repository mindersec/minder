// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
