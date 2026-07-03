// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.starlark.net/starlark"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestFormatEvalResult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantSt  string
		wantMsg string
	}{
		{
			name:    "pass on nil error",
			err:     nil,
			wantSt:  "pass",
			wantMsg: "",
		},
		{
			name:    "fail on ErrEvaluationFailed",
			err:     interfaces.ErrEvaluationFailed,
			wantSt:  "fail",
			wantMsg: interfaces.ErrEvaluationFailed.Error(),
		},
		{
			name:    "skip on ErrEvaluationSkipped",
			err:     interfaces.ErrEvaluationSkipped,
			wantSt:  "skip",
			wantMsg: interfaces.ErrEvaluationSkipped.Error(),
		},
		{
			name:    "error on unknown error",
			err:     errors.New("some unexpected error"),
			wantSt:  "error",
			wantMsg: "some unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dict := formatEvalResult(tt.err)

			result, err := dictToGoMap(dict)
			if err != nil {
				t.Fatalf("dictToGoMap failed: %v", err)
			}

			diff := cmp.Diff(result, map[string]any{"status": tt.wantSt, "message": tt.wantMsg})
			if diff != "" {
				t.Errorf("unexpected result: %s", diff)
			}
		})
	}
}

func TestBuiltinEval_InvalidArgs(t *testing.T) {
	t.Parallel()

	thread := &starlark.Thread{Name: "test"}

	tests := []struct {
		name    string
		args    starlark.Tuple
		kwargs  []starlark.Tuple
		wantErr string
	}{
		{
			name:    "missing rule param",
			args:    starlark.Tuple{},
			kwargs:  nil,
			wantErr: "missing argument for rule",
		},
		{
			name:    "rule not a string",
			args:    starlark.Tuple{starlark.MakeInt(1)},
			wantErr: "got int, want string",
		},
		{
			name:    "invalid entity type",
			args:    starlark.Tuple{starlark.String("rule.yaml")},
			kwargs:  []starlark.Tuple{{starlark.String("entity"), starlark.String("not a dict")}},
			wantErr: "got string, want dict",
		},
		{
			name: "invalid mock_fs type",
			args: starlark.Tuple{starlark.String("fs_check")},
			kwargs: []starlark.Tuple{
				{starlark.String("mock_fs"), starlark.String("not a dict")},
			},
			wantErr: "got string, want dict",
		},
		{
			name: "mock_fs non-string keys",
			args: starlark.Tuple{starlark.String("fs_check")},
			kwargs: []starlark.Tuple{
				{
					starlark.String("mock_fs"),
					func() *starlark.Dict {
						d := starlark.NewDict(1)
						_ = d.SetKey(starlark.MakeInt(1), starlark.String("content"))
						return d
					}(),
				},
			},
			wantErr: "mock_fs keys and values must be strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := &testCaseRunner{}
			_, err := tr.builtinEval(thread, starlark.NewBuiltin("eval", tr.builtinEval), tt.args, tt.kwargs)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain expected string %q", err.Error(), tt.wantErr)
			}
		})
	}
}
