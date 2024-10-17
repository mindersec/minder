// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package domain

// Violation contains the result of a homoglyph violation
type Violation struct {
	// InvisibleChar an invisible character found in a line.
	InvisibleChar rune

	// mixedScript is a mixed script found in a line.
	MixedScript *MixedScriptInfo
}
