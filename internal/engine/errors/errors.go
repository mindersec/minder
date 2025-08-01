// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package errors provides errors for the evaluator engine
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

const (
	maxDetailsMessageSize int64 = 1 << 10
)

// ErrInternal is an error that occurs when there is an internal error in the minder engine.
var ErrInternal = errors.New("internal minder error")

type limitedWriter struct {
	w io.Writer
	n int64
}

var _ io.Writer = (*limitedWriter)(nil)

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n < 0 {
		return 0, io.ErrShortBuffer
	}
	if int64(len(p)) > l.n {
		return 0, io.ErrShortBuffer
	}
	n, err := l.w.Write(p)
	l.n -= int64(n)
	return n, err
}

// LimitedWriter returns a writer that allows up to `n` bytes being
// written. If more than `n` total bytes are written,
// `io.ErrShortBuffer` is returned.
func LimitedWriter(w io.Writer, n int64) io.Writer {
	return &limitedWriter{
		w: w,
		n: n,
	}
}

// EvaluationError is a custom error type for evaluation errors.
type EvaluationError struct {
	Base         error
	Msg          string
	Template     string
	TemplateArgs any
}

// Unwrap returns the base error, allowing errors.Is to work with wrapped errors.
func (e *EvaluationError) Unwrap() error {
	return e.Base
}

// Error implements the error interface for EvaluationError.
func (e *EvaluationError) Error() string {
	return fmt.Sprintf("%v: %s", e.Base, e.Msg)
}

// Details returns a pretty-printed message detailing the reason of
// the failure.
func (e *EvaluationError) Details() string {
	if e.Template == "" {
		return e.Msg
	}
	tmpl, err := template.New("error").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"stringsJoin": strings.Join,
		}).
		Parse(e.Template)
	if err != nil {
		return e.Error()
	}

	var buf strings.Builder
	w := LimitedWriter(&buf, maxDetailsMessageSize)
	if err := tmpl.Execute(w, e.TemplateArgs); err != nil {
		return e.Error()
	}
	return buf.String()
}

// NewDetailedErrEvaluationFailed creates a new evaluation error with
// a given error message and a templated detail message.
func NewDetailedErrEvaluationFailed(
	tmpl string,
	tmplArgs any,
	sfmt string,
	args ...any,
) error {
	formatted := fmt.Sprintf(sfmt, args...)
	return &EvaluationError{
		Base:         interfaces.ErrEvaluationFailed,
		Msg:          formatted,
		Template:     tmpl,
		TemplateArgs: tmplArgs,
	}
}

// NewErrEvaluationFailed creates a new evaluation error with a formatted message.
func NewErrEvaluationFailed(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return &EvaluationError{
		Base: interfaces.ErrEvaluationFailed,
		Msg:  msg,
	}
}

// NewErrEvaluationSkipped creates a new evaluation error
func NewErrEvaluationSkipped(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", interfaces.ErrEvaluationSkipped, msg)
}

// ErrEvaluationSkipSilently specifies that the rule was evaluated but skipped silently.
var ErrEvaluationSkipSilently = errors.New("evaluation skipped silently")

// NewErrEvaluationSkipSilently creates a new evaluation error
func NewErrEvaluationSkipSilently(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrEvaluationSkipSilently, msg)
}

// ErrActionSkipped is an error code that indicates that the action was not performed at all because
// the evaluation passed and the action was not needed
var ErrActionSkipped = errors.New("action skipped")

// ErrActionPending is an error code that indicates that the action was performed but is pending, i.e., opened a PR.
var ErrActionPending = errors.New("action pending")

// IsActionInformativeError returns true if the error is an informative error that should not be reported to the user
func IsActionInformativeError(err error) bool {
	return errors.Is(err, ErrActionSkipped) ||
		errors.Is(err, ErrActionNotAvailable) ||
		errors.Is(err, ErrActionTurnedOff) ||
		errors.Is(err, ErrActionPending)
}

// IsActionFatalError returns true if the error is a fatal error that should stop be reported to the user
func IsActionFatalError(err error) bool {
	return err != nil && !IsActionInformativeError(err)
}

// ErrActionFailed is an error code that indicates that the action was attempted but failed.
var ErrActionFailed = errors.New("action failed")

// NewErrActionFailed creates a new action error
func NewErrActionFailed(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrActionFailed, msg)
}

// ErrActionNotAvailable is an error code that indicates that the action was not available for this rule_type
var ErrActionNotAvailable = errors.New("action not available")

// ErrActionTurnedOff is an error code that indicates that the action is turned off for this rule_type
var ErrActionTurnedOff = errors.New("action turned off")

// ActionsError is the error wrapper for actions
type ActionsError struct {
	RemediateErr  error
	RemediateMeta json.RawMessage
	AlertErr      error
	AlertMeta     json.RawMessage
}

// ErrorAsEvalStatus returns the evaluation status for a given error
func ErrorAsEvalStatus(err error) db.EvalStatusTypes {
	if errors.Is(err, interfaces.ErrEvaluationFailed) {
		return db.EvalStatusTypesFailure
	} else if errors.Is(err, interfaces.ErrEvaluationSkipped) {
		return db.EvalStatusTypesSkipped
	} else if err != nil {
		return db.EvalStatusTypesError
	}
	return db.EvalStatusTypesSuccess
}

// ErrorAsEvalDetails returns the evaluation details for a given error
func ErrorAsEvalDetails(err error) string {
	var evalErr *EvaluationError
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
	if err == nil {
		return db.RemediationStatusTypesSuccess
	}

	switch err != nil {
	case errors.Is(err, ErrActionFailed):
		return db.RemediationStatusTypesFailure
	case errors.Is(err, ErrActionSkipped):
		return db.RemediationStatusTypesSkipped
	case errors.Is(err, ErrActionNotAvailable):
		return db.RemediationStatusTypesNotAvailable
	case errors.Is(err, ErrActionPending):
		return db.RemediationStatusTypesPending
	}
	return db.RemediationStatusTypesError
}

// RemediationStatusAsError returns the remediation status for a given error
func RemediationStatusAsError(prevStatus *db.ListRuleEvaluationsByProfileIdRow) error {
	if prevStatus == nil {
		return ErrActionSkipped
	}

	s := prevStatus.RemStatus
	switch s {
	case db.RemediationStatusTypesSuccess:
		return nil
	case db.RemediationStatusTypesFailure:
		return ErrActionFailed
	case db.RemediationStatusTypesSkipped:
		return ErrActionSkipped
	case db.RemediationStatusTypesNotAvailable:
		return ErrActionNotAvailable
	case db.RemediationStatusTypesPending:
		return ErrActionPending
	case db.RemediationStatusTypesError:
		return fmt.Errorf("generic remediation error status: %s", s)
	}
	return fmt.Errorf("generic remediation error status: %s", s)
}

// ErrorAsAlertStatus returns the alert status for a given error
func ErrorAsAlertStatus(err error) db.AlertStatusTypes {
	if err == nil {
		return db.AlertStatusTypesOn
	}

	switch err != nil {
	case errors.Is(err, ErrActionTurnedOff):
		return db.AlertStatusTypesOff
	case errors.Is(err, ErrActionFailed):
		return db.AlertStatusTypesError
	case errors.Is(err, ErrActionSkipped):
		return db.AlertStatusTypesSkipped
	case errors.Is(err, ErrActionNotAvailable):
		return db.AlertStatusTypesNotAvailable
	}
	return db.AlertStatusTypesError
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
		return ErrActionTurnedOff
	case db.AlertStatusTypesError:
		return ErrActionFailed
	case db.AlertStatusTypesSkipped:
		return ErrActionSkipped
	case db.AlertStatusTypesNotAvailable:
		return ErrActionNotAvailable
	}
	return fmt.Errorf("unknown alert status: %s", s)
}

var (
	// ErrUnauthorized is returned when a request is unauthorized
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned when a request is forbidden
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("not found")
	// ErrValidateOrSpammed is returned when a request is a validation or spammed error
	ErrValidateOrSpammed = errors.New("validation or spammed error")
	// ErrClientError is returned when a request is a client error
	ErrClientError = errors.New("client error")
	// ErrServerError is returned when a request is a server error
	ErrServerError = errors.New("server error")
	// ErrOther is returned when a request is another error
	ErrOther = errors.New("other error")
)

// HTTPErrorCodeToErr converts an HTTP error code to an error
func HTTPErrorCodeToErr(httpCode int) error {
	var err = ErrOther

	switch {
	case httpCode >= 200 && httpCode < 300:
		return nil
	case httpCode == 401:
		return ErrUnauthorized
	case httpCode == 403:
		return ErrForbidden
	case httpCode == 404:
		return ErrNotFound
	case httpCode == 422:
		return ErrValidateOrSpammed
	case httpCode >= 400 && httpCode < 500:
		return ErrClientError
	case httpCode >= 500:
		return ErrServerError
	}

	return err
}

// EvalErrorAsString returns the evaluation error as a string
func EvalErrorAsString(err error) string {
	dbEvalStatus := ErrorAsEvalStatus(err)
	return string(dbEvalStatus)
}

// RemediationErrorAsString returns the remediation error as a string
func RemediationErrorAsString(err error) string {
	dbRemediationStatus := ErrorAsRemediationStatus(err)
	return string(dbRemediationStatus)
}

// AlertErrorAsString returns the alert error as a string
func AlertErrorAsString(err error) string {
	dbAlertStatus := ErrorAsAlertStatus(err)
	return string(dbAlertStatus)
}
