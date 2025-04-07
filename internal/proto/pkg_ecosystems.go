// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package proto contains internal protocol buffer API definitions and helpers.
package proto

// AsString returns the string representation of the DepEcosystem
func (ecosystem DepEcosystem) AsString() string {
	switch ecosystem {
	case DepEcosystem_DEP_ECOSYSTEM_NPM:
		return "npm"
	case DepEcosystem_DEP_ECOSYSTEM_GO:
		return "Go"
	case DepEcosystem_DEP_ECOSYSTEM_PYPI:
		return "PyPI"
	case DepEcosystem_DEP_ECOSYSTEM_UNSPECIFIED:
		// this shouldn't happen
		return ""
	default:
		return ""
	}
}
