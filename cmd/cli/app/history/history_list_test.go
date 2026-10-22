// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCursorFromOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cursorStr string
		size      uint32
		want      *minderv1.Cursor
	}{
		{
			name: "no options returns nil",
			want: nil,
		},
		{
			name:      "cursor only",
			cursorStr: "next-page",
			want:      &minderv1.Cursor{Cursor: "next-page"},
		},
		{
			name: "size only",
			size: 10,
			want: &minderv1.Cursor{Size: 10},
		},
		{
			name:      "cursor and size",
			cursorStr: "next-page",
			size:      10,
			want:      &minderv1.Cursor{Cursor: "next-page", Size: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, cursorFromOptions(tt.cursorStr, tt.size))
		})
	}
}

func TestGetNext(t *testing.T) {
	t.Parallel()

	cursor := &minderv1.Cursor{Cursor: "next"}

	tests := []struct {
		name string
		resp *minderv1.ListEvaluationHistoryResponse
		want *minderv1.Cursor
	}{
		// nil resp is not tested: getNext is an internal helper whose only
		// caller checks the gRPC error before invoking it, so resp is always
		// non-nil at this point.  Passing nil would panic: getNext dereferences resp.Page.
		{
			name: "nil page returns nil",
			resp: &minderv1.ListEvaluationHistoryResponse{},
			want: nil,
		},
		{
			name: "nil next cursor returns nil",
			resp: &minderv1.ListEvaluationHistoryResponse{
				Page: &minderv1.CursorPage{},
			},
			want: nil,
		},
		{
			name: "next cursor is returned",
			resp: &minderv1.ListEvaluationHistoryResponse{
				Page: &minderv1.CursorPage{Next: cursor},
			},
			want: cursor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, getNext(tt.resp))
		})
	}
}

func TestGetPrev(t *testing.T) {
	t.Parallel()

	cursor := &minderv1.Cursor{Cursor: "previous"}

	tests := []struct {
		name string
		resp *minderv1.ListEvaluationHistoryResponse
		want *minderv1.Cursor
	}{
		// nil resp is not tested for the same reason as TestGetNext: passing nil would panic.
		{
			name: "nil page returns nil",
			resp: &minderv1.ListEvaluationHistoryResponse{},
			want: nil,
		},
		{
			name: "nil prev cursor returns nil",
			resp: &minderv1.ListEvaluationHistoryResponse{
				Page: &minderv1.CursorPage{},
			},
			want: nil,
		},
		{
			name: "prev cursor is returned",
			resp: &minderv1.ListEvaluationHistoryResponse{
				Page: &minderv1.CursorPage{Prev: cursor},
			},
			want: cursor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, getPrev(tt.resp))
		})
	}
}

func TestValidatedFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filters  []string
		accepted []string
		wantErr  bool
	}{
		{
			name:     "nil filters is always valid",
			filters:  nil,
			accepted: []string{"success", "failure"},
			wantErr:  false,
		},
		{
			name:     "empty filters is always valid",
			filters:  []string{},
			accepted: []string{"success", "failure"},
			wantErr:  false,
		},
		{
			name:     "all filters in accepted set",
			filters:  []string{"success", "failure"},
			accepted: []string{"success", "failure"},
			wantErr:  false,
		},
		{
			name:     "unknown filter value is rejected",
			filters:  []string{"unknown"},
			accepted: []string{"success", "failure"},
			wantErr:  true,
		},
		{
			name:     "one invalid value among valid values is rejected",
			filters:  []string{"success", "unknown"},
			accepted: []string{"success", "failure"},
			wantErr:  true,
		},
		{
			name:     "non-empty filters with empty accepted set is rejected",
			filters:  []string{"success"},
			accepted: []string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatedFilter(tt.filters, tt.accepted)
			if tt.wantErr {
				require.ErrorContains(t, err, "unexpected filter value")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
