// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"errors"
	"testing"

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

			v, found, err := dict.Get(starlark.String("status"))
			if err != nil || !found {
				t.Fatalf("failed to get status: %v", err)
			}
			if s, ok := v.(starlark.String); !ok || string(s) != tt.wantSt {
				t.Errorf("status = %v, want %q", v, tt.wantSt)
			}

			v, found, err = dict.Get(starlark.String("message"))
			if err != nil || !found {
				t.Fatalf("failed to get message: %v", err)
			}
			if s, ok := v.(starlark.String); !ok || string(s) != tt.wantMsg {
				t.Errorf("message = %v, want %q", v, tt.wantMsg)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := builtinEval(thread, starlark.NewBuiltin("eval", builtinEval), tt.args, tt.kwargs)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() == "" {
				t.Fatalf("error is empty")
			}
		})
	}
}
