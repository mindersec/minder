// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package dbadapter provides database-specific error mappings between
// engine errors and database status types.
package dbadapter

import (
	"errors"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	engineerrors "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// ErrorAsEvalStatus returns the evaluation status for a given error
func ErrorAsEvalStatus(err error) db.EvalStatusTypes {
	switch {
	case errors.Is(err, interfaces.ErrEvaluationFailed):
		return db.EvalStatusTypesFailure
	case errors.Is(err, interfaces.ErrEvaluationSkipped):
		return db.EvalStatusTypesSkipped
	case err != nil:
		return db.EvalStatusTypesError
	default:
		return db.EvalStatusTypesSuccess
	}
}

// ErrorAsEvalDetails returns the evaluation details for a given error
func ErrorAsEvalDetails(err error) string {
	var evalErr *engineerrors.EvaluationError
	if errors.As(err, &evalErr) && evalErr.Template != "" {
		return evalErr.Details()
	}
	if errors.As(err, &evalErr) {
		return evalErr.Msg
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// ErrorAsRemediationStatus returns the remediation status for a given error
func ErrorAsRemediationStatus(err error) db.RemediationStatusTypes {
	switch {
	case err == nil:
		return db.RemediationStatusTypesSuccess
	case errors.Is(err, engineerrors.ErrActionFailed):
		return db.RemediationStatusTypesFailure
	case errors.Is(err, engineerrors.ErrActionSkipped):
		return db.RemediationStatusTypesSkipped
	case errors.Is(err, engineerrors.ErrActionNotAvailable):
		return db.RemediationStatusTypesNotAvailable
	case errors.Is(err, engineerrors.ErrActionPending):
		return db.RemediationStatusTypesPending
	default:
		return db.RemediationStatusTypesError
	}
}

// RemediationStatusAsError returns the remediation status for a given error
func RemediationStatusAsError(prevStatus *db.ListRuleEvaluationsByProfileIdRow) error {
	if prevStatus == nil {
		return engineerrors.ErrActionSkipped
	}

	s := prevStatus.RemStatus
	switch s {
	case db.RemediationStatusTypesSuccess:
		return nil
	case db.RemediationStatusTypesFailure:
		return engineerrors.ErrActionFailed
	case db.RemediationStatusTypesSkipped:
		return engineerrors.ErrActionSkipped
	case db.RemediationStatusTypesNotAvailable:
		return engineerrors.ErrActionNotAvailable
	case db.RemediationStatusTypesPending:
		return engineerrors.ErrActionPending
	case db.RemediationStatusTypesError:
		fallthrough
	default:
		return fmt.Errorf("generic remediation error status: %s", s)
	}
}

// ErrorAsAlertStatus returns the alert status for a given error
func ErrorAsAlertStatus(err error) db.AlertStatusTypes {
	switch {
	case err == nil:
		return db.AlertStatusTypesOn
	case errors.Is(err, engineerrors.ErrActionTurnedOff):
		return db.AlertStatusTypesOff
	case errors.Is(err, engineerrors.ErrActionFailed):
		return db.AlertStatusTypesError
	case errors.Is(err, engineerrors.ErrActionSkipped):
		return db.AlertStatusTypesSkipped
	case errors.Is(err, engineerrors.ErrActionNotAvailable):
		return db.AlertStatusTypesNotAvailable
	default:
		return db.AlertStatusTypesError
	}
}

// AlertStatusAsError returns the error for a given alert status
func AlertStatusAsError(prevStatus *db.ListRuleEvaluationsByProfileIdRow) error {
	if prevStatus == nil {
		return errors.New("no previous alert state")
	}

	s := prevStatus.AlertStatus
	switch s {
	case db.AlertStatusTypesOn:
		return nil
	case db.AlertStatusTypesOff:
		return engineerrors.ErrActionTurnedOff
	case db.AlertStatusTypesError:
		return engineerrors.ErrActionFailed
	case db.AlertStatusTypesSkipped:
		return engineerrors.ErrActionSkipped
	case db.AlertStatusTypesNotAvailable:
		return engineerrors.ErrActionNotAvailable
	default:
		return fmt.Errorf("unknown alert status: %s", s)
	}
}

// EvalErrorAsString returns the evaluation error as a string
func EvalErrorAsString(err error) string {
	return string(ErrorAsEvalStatus(err))
}

// RemediationErrorAsString returns the remediation error as a string
func RemediationErrorAsString(err error) string {
	return string(ErrorAsRemediationStatus(err))
}

// AlertErrorAsString returns the alert error as a string
func AlertErrorAsString(err error) string {
	return string(ErrorAsAlertStatus(err))
}
