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

// ErrRemediationSkipped is an error code that indicates that the remediation was not performed at all because
// the evaluation passed and the remediation was not needed.
var ErrRemediationSkipped = errors.New("remediation not performed")

// IsRemediateInformativeError returns true if the error is an informative error that should not be reported to the user
func IsRemediateInformativeError(err error) bool {
	return errors.Is(err, ErrRemediationSkipped) || errors.Is(err, ErrRemediationNotAvailable)
}

// IsRemediateFatalError returns true if the error is a fatal error that should stop be reported to the user
func IsRemediateFatalError(err error) bool {
	return err != nil && !IsRemediateInformativeError(err)
}

// ErrRemediateFailed is an error code that indicates that the remediation was attempted but failed.
var ErrRemediateFailed = errors.New("remediation failed")

// NewErrRemediationFailed creates a new remediation error
func NewErrRemediationFailed(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrRemediateFailed, msg)
}

// ErrRemediationNotAvailable is an error code that indicates that the remediation was not available for this rule_type
var ErrRemediationNotAvailable = errors.New("remediation not available")
