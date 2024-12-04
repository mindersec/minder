// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps implements a data source that extracts dependencies from
// a filesystem or file.
package deps

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateArgs(t *testing.T) {
	h := depsDataSourceHandler{}
	t.Parallel()
	for _, tc := range []struct {
		name    string
		args    any
		mustErr bool
	}{
		{name: "no-args", args: nil, mustErr: false},
		{name: "wrong-type", args: struct{}{}, mustErr: true},
		{name: "no-path", args: map[string]any{"ecosystems": []string{"npm"}}, mustErr: false},
		{name: "blank-path", args: map[string]any{"path": "", "ecosystems": []string{"npm"}}, mustErr: false},
		{name: "path-set", args: map[string]any{"path": "directory/", "ecosystems": []string{"npm"}}, mustErr: false},
		{name: "no-ecosystems", args: map[string]any{"path": "directory/"}, mustErr: false},
		{name: "ecosystems-empty", args: map[string]any{"path": "directory/", "ecosystems": []string{}}, mustErr: false},
		{name: "ecosystems-nil", args: map[string]any{"path": "directory/", "ecosystems": nil}, mustErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := h.ValidateArgs(tc.args)
			if tc.mustErr {
				require.Error(t, res)
				return
			}
			require.NoError(t, res)
		})
	}
}

func TestValidateEcosystems(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		list    any
		mustErr bool
	}{
		{"empty-list", nil, false},
		{"valid-list-0", []string{}, false},
		{"valid-list-1", []string{"npm"}, false},
		{"valid-list-1+", []string{"npm", "pypi", "cargo"}, false},
		{"invalid-type", []string{"npm", "Hello!", "cargo"}, true},
		{"other-something", []struct{}{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := validateEcosystems(tc.list)
			if tc.mustErr {
				require.Error(t, errors.Join(errs...))
				return
			}
			require.Len(t, errs, 0)
		})
	}
}
