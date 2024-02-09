package domain

// Violation contains the result of a homoglyph violation
type Violation struct {
	// invisibleChar an invisible character found in a line.
	invisibleChar rune

	// mixedScript is a mixed script found in a line.
	mixedScript *MixedScriptInfo
}
