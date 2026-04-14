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
	"time"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

const (
	maxDetailsMessageSize int64 = 1 << 10
)

// ErrInternal is an error that occurs when there is an internal error in the minder engine.
var ErrInternal = errors.New("internal minder error")

// RateLimitError is a custom error type for rate limit errors.
type RateLimitError struct {
	Base      error
	Limit     int64
	Remaining int64
	ResetTime time.Time
}

// Unwrap returns the base error
func (e *RateLimitError) Unwrap() error {
	return e.Base
}

// Error implements the error interface for RateLimitError.
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %v (limit: %d, remaining: %d, reset at: %v)",
		e.Base, e.Limit, e.Remaining, e.ResetTime)
}

// NewRateLimitError creates a new rate limit error.
func NewRateLimitError(base error, limit, remaining int64, resetTime time.Time) error {
	return &RateLimitError{
		Base:      base,
		Limit:     limit,
		Remaining: remaining,
		ResetTime: resetTime,
	}
}

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

// HTTPErrorCodeToErr returns an engine error corresponding to the given HTTP status code.
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
