// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package templates contains template strings for evaluation details.
package templates

import (
	// This comment makes the linter happy.
	_ "embed"
)

// RegoDenyByDefaultTemplate is the template for details of the `rego`
// evaluation engine of type `deny-by-default`.
//
// It expects a `message` scalar values to be set. It optionally
// accepts an `entityName` string.
//
//go:embed regoDenyByDefaultTemplate.tmpl
var RegoDenyByDefaultTemplate string

// RegoConstraints is the template for details of the `rego`
// evaluation engine of type `constraints`.
//
// It expects a list of strings named `violations` to be set.
//
//go:embed regoConstraints.tmpl
var RegoConstraints string

// VulncheckTemplate is the template for evaluation details of the
// `vulncheck` evaluation engine.
//
// It expects a list of strings value named `packages`.
//
//go:embed vulncheckTemplate.tmpl
var VulncheckTemplate string

// TrustyTemplate is the template for evaluation details of the
// `trusty` evaluation engine.
//
// This template accepts two parameters, `lowScoringPackages`
// and `maliciousPackages`, which must be list of strings.
//
//go:embed trustyTemplate.tmpl
var TrustyTemplate string

// MixedScriptsTemplate is the template for details of the `homoglyphs`
// evaluation engine of type `mixed_scripts`.
//
// This template expects a list of Violations named `violations`.
//
//go:embed mixedScriptsTemplate.tmpl
var MixedScriptsTemplate string

// InvisibleCharactersTemplate is the template for details of the `homoglyphs`
// evaluation engine of type `invisible_characters`.
//
// This template expects a list of Violations named `violations`.
//
//go:embed invisibleCharactersTemplate.tmpl
var InvisibleCharactersTemplate string

// JqTemplate is the template for details of the `jq` evaluation engine.
//
// This template expects three parameters, `path`, `expected`, and `actual`, which are strings.
//
//go:embed jq.tmpl
var JqTemplate string
