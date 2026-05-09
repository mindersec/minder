// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"github.com/open-policy-agent/opa/v1/ast"
)

// DetectRegoVersion attempts to parse the Rego source with the V1 parser first.
// If V1 parsing succeeds, it returns ast.RegoV1. If V1 parsing fails, it falls
// back to ast.RegoV0. This allows the system to accept both V0 and V1 policies
// without requiring any user-facing changes.
func DetectRegoVersion(def string) ast.RegoVersion {
	_, err := ast.ParseModuleWithOpts(MinderRegoFile, def,
		ast.ParserOptions{RegoVersion: ast.RegoV1})
	if err == nil {
		return ast.RegoV1
	}

	return ast.RegoV0
}

// VersionToString converts an ast.RegoVersion to the string stored in the
// database.
func VersionToString(v ast.RegoVersion) string {
	if v == ast.RegoV1 {
		return "v1"
	}
	return "v0"
}

// VersionFromString converts a stored database string back to an
// ast.RegoVersion.
func VersionFromString(s string) ast.RegoVersion {
	if s == "v1" {
		return ast.RegoV1
	}
	return ast.RegoV0
}
