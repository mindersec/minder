// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package history

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var foo = "foo"

func TestListEvaluationCursor(t *testing.T) {
	t.Parallel()

	epoch := time.UnixMicro(0).UTC()
	future := time.UnixMicro(999999999999999999).UTC()

	tests := []struct {
		name   string
		cursor func(*testing.T) string
		check  func(*testing.T, *ListEvaluationCursor)
		err    bool
	}{
		{
			name: "implicit next",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("0")
				return base64.StdEncoding.EncodeToString(payload)
			},
			check: func(t *testing.T, cursor *ListEvaluationCursor) {
				t.Helper()
				require.Equal(t, epoch, cursor.Time)
				require.Equal(t, Next, cursor.Direction)
			},
		},
		{
			name: "explicit next",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("+0")
				return base64.StdEncoding.EncodeToString(payload)
			},
			check: func(t *testing.T, cursor *ListEvaluationCursor) {
				t.Helper()
				require.Equal(t, epoch, cursor.Time)
				require.Equal(t, Next, cursor.Direction)
			},
		},
		{
			name: "explicit prev",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("-0")
				return base64.StdEncoding.EncodeToString(payload)
			},
			check: func(t *testing.T, cursor *ListEvaluationCursor) {
				t.Helper()
				require.Equal(t, epoch, cursor.Time)
				require.Equal(t, Prev, cursor.Direction)
			},
		},
		{
			name: "wrong uuid",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("malformed")
				return base64.StdEncoding.EncodeToString(payload)
			},
			err: true,
		},
		{
			name: "wrong uuid next",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("+malformed")
				return base64.StdEncoding.EncodeToString(payload)
			},
			err: true,
		},
		{
			name: "wrong uuid prev",
			cursor: func(t *testing.T) string {
				t.Helper()
				payload := []byte("-malformed")
				return base64.StdEncoding.EncodeToString(payload)
			},
			err: true,
		},
		{
			name: "empty",
			cursor: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			check: func(t *testing.T, cursor *ListEvaluationCursor) {
				t.Helper()
				require.Equal(t, future, cursor.Time)
				require.Equal(t, Next, cursor.Direction)
			},
		},
		{
			name: "not base64 encoded",
			cursor: func(t *testing.T) string {
				t.Helper()
				return "not base64 encoded"
			},
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded := tt.cursor(t)
			res, err := ParseListEvaluationCursor(encoded)

			if tt.err {
				require.Error(t, err)
				require.Nil(t, res)
				return
			}

			require.NoError(t, err)
			tt.check(t, res)
		})
	}
}

func TestListEvaluationFilter(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name   string
		filter func(*testing.T) (ListEvaluationFilter, error)
		check  func(*testing.T, ListEvaluationFilter)
		err    bool
	}{
		{
			name: "happy path",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectIDStr("deadbeef-0000-0000-0000-000000000000"),
					WithEntityType("repository"),
				)
			},
			check: func(t *testing.T, filter ListEvaluationFilter) {
				t.Helper()
				require.Equal(t, []string{"repository"}, filter.IncludedEntityTypes())
			},
		},
		{
			name: "mandatory project id",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithEntityType("repository"),
				)
			},
			err: true,
		},
		{
			name: "non-empty project id",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectID(uuid.Nil),
				)
			},
			err: true,
		},
		{
			name: "bogus",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithEntityType(""),
				)
			},
			err: true,
		},
		{
			name: "valid time range",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectIDStr("deadbeef-0000-0000-0000-000000000000"),
					WithFrom(now),
					WithTo(now),
				)
			},
			check: func(t *testing.T, filter ListEvaluationFilter) {
				t.Helper()
				require.Equal(t, now, *filter.GetFrom())
				require.Equal(t, now, *filter.GetTo())
			},
		},
		{
			name: "no from",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectIDStr("deadbeef-0000-0000-0000-000000000000"),
					WithTo(now),
				)
			},
			err: true,
		},
		{
			name: "no to",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectIDStr("deadbeef-0000-0000-0000-000000000000"),
					WithFrom(now),
				)
			},
			err: true,
		},
		{
			name: "from after to",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProjectIDStr("deadbeef-0000-0000-0000-000000000000"),
					WithEntityType("repository"),
					WithFrom(now.Add(1*time.Millisecond)),
					WithTo(now),
				)
			},
			err: true,
		},

		// inclusion-exclusion errors
		{
			name: "inclusion exclusion entity type",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithEntityType("repository"),
					WithEntityType("!artifact"),
				)
			},
			err: true,
		},
		{
			name: "inclusion exclusion entity name",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithEntityName("foo"),
					WithEntityName("!bar"),
				)
			},
			err: true,
		},
		{
			name: "inclusion exclusion profile name",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithProfileName("foo"),
					WithProfileName("!bar"),
				)
			},
			err: true,
		},
		{
			name: "inclusion exclusion evaluation status",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithStatus("success"),
					WithStatus("!failure"),
				)
			},
			err: true,
		},
		{
			name: "inclusion exclusion remediation status",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithRemediation("success"),
					WithRemediation("!failure"),
				)
			},
			err: true,
		},
		{
			name: "inclusion exclusion alert status",
			filter: func(t *testing.T) (ListEvaluationFilter, error) {
				t.Helper()
				return NewListEvaluationFilter(
					WithAlert("on"),
					WithAlert("!off"),
				)
			},
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res, err := tt.filter(t)

			if tt.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.check(t, res)
		})
	}
}

func TestFilterOptions(t *testing.T) {
	t.Parallel()

	now := time.Now()

	uuidstr := "deadbeef-0000-0000-0000-000000000000"
	uuidval := uuid.MustParse(uuidstr)

	tests := []struct {
		name   string
		option func(*testing.T) FilterOpt
		filter func(*testing.T) Filter
		check  func(*testing.T, Filter)
		err    bool
	}{
		// project id
		{
			name: "project id string",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectIDStr(uuidstr)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(ProjectFilter)
				require.Equal(t, uuidval, f.GetProjectID())
			},
		},
		{
			name: "project id uuid",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectID(uuidval)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(ProjectFilter)
				require.Equal(t, uuidval, f.GetProjectID())
			},
		},
		{
			name: "project id nil",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectID(uuid.Nil)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "project id malformed",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectIDStr("malformed")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong project filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectIDStr(uuidstr)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},
		{
			name: "wrong project filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProjectID(uuidval)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// entity type
		{
			name: "entity type in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityType("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(EntityTypeFilter)
				require.NotNil(t, f.IncludedEntityTypes())
				require.Equal(t, []string{"repository"}, f.IncludedEntityTypes())
				require.Nil(t, f.ExcludedEntityTypes())
			},
		},
		{
			name: "entity type not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityType("!repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(EntityTypeFilter)
				require.Nil(t, f.IncludedEntityTypes())
				require.NotNil(t, f.ExcludedEntityTypes())
				require.Equal(t, []string{"repository"}, f.ExcludedEntityTypes())
			},
		},
		{
			name: "empty entity type",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityType("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus entity type",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityType("foo")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong entity type filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityType("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// entity name
		{
			name: "entity name in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityName("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(EntityNameFilter)
				require.NotNil(t, f.IncludedEntityNames())
				require.Equal(t, []string{"repository"}, f.IncludedEntityNames())
				require.Nil(t, f.ExcludedEntityNames())
			},
		},
		{
			name: "entity name not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityName("!repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(EntityNameFilter)
				require.Nil(t, f.IncludedEntityNames())
				require.NotNil(t, f.ExcludedEntityNames())
				require.Equal(t, []string{"repository"}, f.ExcludedEntityNames())
			},
		},
		{
			name: "empty entity name",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityName("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus entity name",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityName("!")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong entity name filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithEntityName("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// profile name
		{
			name: "profile name in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProfileName("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(ProfileNameFilter)
				require.NotNil(t, f.IncludedProfileNames())
				require.Equal(t, []string{"repository"}, f.IncludedProfileNames())
				require.Nil(t, f.ExcludedProfileNames())
			},
		},
		{
			name: "profile name not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProfileName("!repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(ProfileNameFilter)
				require.Nil(t, f.IncludedProfileNames())
				require.NotNil(t, f.ExcludedProfileNames())
				require.Equal(t, []string{"repository"}, f.ExcludedProfileNames())
			},
		},
		{
			name: "empty profile name",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProfileName("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus profile name",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProfileName("!")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong profile name filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithProfileName("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// status
		{
			name: "status in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithStatus("success")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(StatusFilter)
				require.NotNil(t, f.IncludedStatuses())
				require.Equal(t, []string{"success"}, f.IncludedStatuses())
				require.Nil(t, f.ExcludedStatuses())
			},
		},
		{
			name: "status not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithStatus("!success")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(StatusFilter)
				require.Nil(t, f.IncludedStatuses())
				require.NotNil(t, f.ExcludedStatuses())
				require.Equal(t, []string{"success"}, f.ExcludedStatuses())
			},
		},
		{
			name: "empty status",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithStatus("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus status",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithStatus("foo")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong status filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithStatus("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// remediation
		{
			name: "remediation in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithRemediation("success")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(RemediationFilter)
				require.NotNil(t, f.IncludedRemediations())
				require.Equal(t, []string{"success"}, f.IncludedRemediations())
				require.Nil(t, f.ExcludedRemediations())
			},
		},
		{
			name: "remediation not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithRemediation("!success")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(RemediationFilter)
				require.Nil(t, f.IncludedRemediations())
				require.NotNil(t, f.ExcludedRemediations())
				require.Equal(t, []string{"success"}, f.ExcludedRemediations())
			},
		},
		{
			name: "empty remediation",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithRemediation("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus remediation",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithRemediation("foo")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong remediation filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithRemediation("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// alert
		{
			name: "alert in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithAlert("on")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(AlertFilter)
				require.NotNil(t, f.IncludedAlerts())
				require.Equal(t, []string{"on"}, f.IncludedAlerts())
				require.Nil(t, f.ExcludedAlerts())
			},
		},
		{
			name: "alert not in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithAlert("!on")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(AlertFilter)
				require.Nil(t, f.IncludedAlerts())
				require.NotNil(t, f.ExcludedAlerts())
				require.Equal(t, []string{"on"}, f.ExcludedAlerts())
			},
		},
		{
			name: "empty alert",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithAlert("")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "bogus alert",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithAlert("foo")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			err: true,
		},
		{
			name: "wrong alert filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithAlert("repository")
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},

		// from-to
		{
			name: "from in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithFrom(now)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(TimeRangeFilter)
				require.NotNil(t, f.GetFrom())
				require.Nil(t, f.GetTo())
				require.Equal(t, now, *f.GetFrom())
			},
		},
		{
			name: "to in filter",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithTo(now)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return &listEvaluationFilter{}
			},
			check: func(t *testing.T, filter Filter) {
				t.Helper()
				f := filter.(TimeRangeFilter)
				require.Nil(t, f.GetFrom())
				require.NotNil(t, f.GetTo())
				require.Equal(t, now, *f.GetTo())
			},
		},
		{
			name: "wrong timerange filter from",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithFrom(now)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},
		{
			name: "wrong timerange filter to",
			option: func(t *testing.T) FilterOpt {
				t.Helper()
				return WithTo(now)
			},
			filter: func(t *testing.T) Filter {
				t.Helper()
				return foo
			},
			err: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opt := tt.option(t)
			filter := tt.filter(t)
			err := opt(filter)
			if tt.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.check(t, filter)
		})
	}
}
