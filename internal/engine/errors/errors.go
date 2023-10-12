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
// Package rule provides the CLI subcommand for managing rules

// Package errors provides errors for the evaluator engine
package errors

import (
	"errors"
	"fmt"
)

// ErrEvaluationFailed is an error that occurs during evaluation of a rule.
var ErrEvaluationFailed = errors.New("evaluation failure")

// NewErrEvaluationFailed creates a new evaluation error
func NewErrEvaluationFailed(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrEvaluationFailed, msg)
}

// ErrEvaluationSkipped specifies that the rule was evaluated but skipped.
var ErrEvaluationSkipped = errors.New("evaluation skipped")

// NewErrEvaluationSkipped creates a new evaluation error
func NewErrEvaluationSkipped(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrEvaluationSkipped, msg)
}

// ErrEvaluationSkipSilently specifies that the rule was evaluated but skipped silently.
var ErrEvaluationSkipSilently = errors.New("evaluation skipped silently")

// NewErrEvaluationSkipSilently creates a new evaluation error
func NewErrEvaluationSkipSilently(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrEvaluationSkipSilently, msg)
}

// ErrActionSkipped is an error code that indicates that the action was not performed at all because
// the evaluation passed and the action was not needed.
var ErrActionSkipped = errors.New("action not performed")

// IsActionInformativeError returns true if the error is an informative error that should not be reported to the user
func IsActionInformativeError(err error) bool {
	return errors.Is(err, ErrActionSkipped) || errors.Is(err, ErrActionNotAvailable)
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

// ActionsError is the error wrapper for actions
type ActionsError struct {
	RemediateErr error
	AlertErr     error
}
