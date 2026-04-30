// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego_test

import (
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/stretchr/testify/require"

	rego "github.com/mindersec/minder/internal/engine/eval/rego"
)

func TestDetectRegoVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		def     string
		want    ast.RegoVersion
		wantStr string
	}{
		{
			name: "v0 policy without imports",
			def: `
package minder

default allow = false

allow {
	file.exists("foo")
}`,
			want:    ast.RegoV0,
			wantStr: "v0",
		},
		{
			name: "v1 policy with import rego.v1",
			def: `
package minder

import rego.v1

default allow := false

allow if {
	file.exists("foo")
}`,
			want:    ast.RegoV1,
			wantStr: "v1",
		},
		{
			name: "v0 policy with future.keywords parses as v1",
			def: `
package minder

import future.keywords.if
import future.keywords.in

default allow = false

allow if {
	"admin" in input.profile.roles
}`,
			// future.keywords policies are accepted by OPA's V1 parser
			// for backward compatibility during migration.
			want:    ast.RegoV1,
			wantStr: "v1",
		},
		{
			name: "v1 constraints policy",
			def: `
package minder

import rego.v1

violations contains {"msg": msg} if {
	input.ingested.name == ""
	msg := "name is required"
}`,
			want:    ast.RegoV1,
			wantStr: "v1",
		},
		{
			name: "v0 constraints policy",
			def: `
package minder

violations[{"msg": msg}] {
	input.ingested.name == ""
	msg = "name is required"
}`,
			want:    ast.RegoV0,
			wantStr: "v0",
		},
		{
			name:    "empty string defaults to v0",
			def:     "",
			want:    ast.RegoV0,
			wantStr: "v0",
		},
		{
			name: "valid in both v0 and v1 returns v1",
			def: `
package minder

default allow := false`,
			want:    ast.RegoV1,
			wantStr: "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := rego.DetectRegoVersion(tt.def)
			require.Equal(t, tt.want, got, "DetectRegoVersion() returned unexpected version")
			require.Equal(t, tt.wantStr, rego.VersionToString(got))
		})
	}
}

func TestRegoVersionRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str     string
		version ast.RegoVersion
	}{
		{"v0", ast.RegoV0},
		{"v1", ast.RegoV1},
		{"", ast.RegoV0},        // unknown defaults to v0
		{"v2", ast.RegoV0},      // unknown defaults to v0
		{"invalid", ast.RegoV0}, // unknown defaults to v0
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			t.Parallel()
			got := rego.VersionFromString(tt.str)
			require.Equal(t, tt.version, got)
		})
	}
}
