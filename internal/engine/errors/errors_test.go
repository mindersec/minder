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

package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/db"
)

func TestAsRemediationStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		err            ActionsError
		shouldErr      bool
		expectedStatus db.RemediationStatusTypes
	}{
		{
			name: "set-pending",
			err: ActionsError{
				RemediateErr:  nil,
				RemediateMeta: []byte(`{"status":"pending"}`),
			},
			expectedStatus: db.RemediationStatusTypesPending,
		},
		{
			name: "error-trumps-status",
			err: ActionsError{
				RemediateErr:  ErrActionFailed,
				RemediateMeta: []byte(`{"status":"pending"}`),
			},
			expectedStatus: db.RemediationStatusTypesFailure,
		},
		{
			name: "invalid-meta",
			err: ActionsError{
				RemediateMeta: []byte(`{"status":1}`),
			},
			expectedStatus: db.RemediationStatusTypesUnknown,
		},
		{
			name:           "no-meta",
			err:            ActionsError{},
			expectedStatus: db.RemediationStatusTypesSuccess,
		},
		{
			name: "custom-status",
			err: ActionsError{
				RemediateMeta: []byte(`{"status":"hello"}`),
			},
			expectedStatus: db.RemediationStatusTypes("hello"),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := tc.err.AsRemediationStatus()
			require.Equal(t, tc.expectedStatus, s)
		})
	}
}

func TestErrorAsRemediationStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		err            error
		expectedStatus db.RemediationStatusTypes
	}{
		{"no-error", nil, db.RemediationStatusTypesSuccess},
		{"action-failed", ErrActionFailed, db.RemediationStatusTypesFailure},
		{"action-skipped", ErrActionSkipped, db.RemediationStatusTypesSkipped},
		{"action-not-available", ErrActionNotAvailable, db.RemediationStatusTypesNotAvailable},
		{"other-error", fmt.Errorf("custom error"), db.RemediationStatusTypesError},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := ErrorAsRemediationStatus(tc.err)
			require.Equal(t, tc.expectedStatus, s)
		})
	}
}
