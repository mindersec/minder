// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

// V1RequiredMessage explains how to convert a legacy policy to Rego V1.
const V1RequiredMessage = "policy must use Rego V1 syntax; run 'opa fmt --v0-v1' to upgrade " +
	"before creating or updating the rule type"
