// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/util"
)

//go:embed resources/scripts.txt
var scriptsContent embed.FS

// MixedScriptsProcessor is a processor for the mixed scripts rule type
type MixedScriptsProcessor struct {
	runeToScript map[rune]string
}

// FindViolations finds mixed scripts in the given line
func (mse *MixedScriptsProcessor) FindViolations(line string) []*Violation {
	return mse.FindMixedScripts(line)
}

// GetSubCommentText returns the sub comment text for mixed scripts
func (_ *MixedScriptsProcessor) GetSubCommentText() string {
	return "**Mixed Scripts Found:**\n\n"
}

// GetLineCommentText returns the line comment text for mixed scripts
func (_ *MixedScriptsProcessor) GetLineCommentText(violation *Violation) string {
	if violation == nil {
		return ""
	}

	return fmt.Sprintf("- Text: `%s`, Scripts: %v\n", violation.MixedScript.Text, violation.MixedScript.ScriptsFound)
}

// GetPassedReviewText returns the passed review text for mixed scripts
func (_ *MixedScriptsProcessor) GetPassedReviewText() string {
	return util.NoMixedScriptsFoundText
}

// GetFailedReviewText returns the failed review text for mixed scripts
func (_ *MixedScriptsProcessor) GetFailedReviewText() string {
	return util.MixedScriptsFoundText
}

// NewMixedScriptsProcessor creates a new MixedScriptsProcessor
func NewMixedScriptsProcessor(ctx context.Context) (HomoglyphProcessor, error) {
	// 7th of Feb, 2024: https://www.unicode.org/Public/UCD/latest/ucd/Scripts.txt
	runeToScript, err := loadScriptData(ctx, "resources/scripts.txt")
	if err != nil {
		return nil, err
	}

	return &MixedScriptsProcessor{
		runeToScript: runeToScript,
	}, nil
}

// MixedScriptInfo contains information about a word that mixes multiple scripts
type MixedScriptInfo struct {
	Text         string
	ScriptsFound []string
}

// FindMixedScripts returns a slice of MixedScriptInfo for words in the input string that
// mix multiple scripts, ignoring common characters, detecting
// potential obfuscation in text. Words with only common script characters are not flagged.
// E.g. “B. C“ is not considered mixed-scripts by default: it contains characters
// from Latin and Common, but Common is excluded by default.
func (mse *MixedScriptsProcessor) FindMixedScripts(line string) []*Violation {
	words := strings.Fields(line)
	mixedScripts := make([]*Violation, 0)

	for _, word := range words {
		scriptsFound := make(map[string]struct{})
		for _, r := range word {
			script, exists := mse.runeToScript[r]
			if !exists || script == "Common" {
				continue
			}
			scriptsFound[script] = struct{}{}
		}

		if len(scriptsFound) > 1 {
			scripts := make([]string, 0, len(scriptsFound))
			for script := range scriptsFound {
				scripts = append(scripts, script)
			}

			msi := &MixedScriptInfo{
				Text:         word,
				ScriptsFound: scripts,
			}
			mixedScripts = append(mixedScripts, &Violation{MixedScript: msi})
		}
	}

	return mixedScripts
}

// loadScriptData reads data from the specified file in Scripts.txt format and populates a runeToScript map.
// The function parses each line of the file, ignoring comments and empty lines.
// It expects lines in the format "<code>; <script> # <description>", where <code> can be a single character
// or a range. For each valid line, it updates runeToScript to map characters (or ranges of characters) to their
// respective scripts. Lines that do not conform to the expected format or contain no script information are skipped.
func loadScriptData(ctx context.Context, filePath string) (map[rune]string, error) {
	file, err := scriptsContent.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).
				Str("file", filePath).
				Str("eval", "mixed_scripts_processor").
				Str("component", "eval").
				Msg("failed to close file")
		}
	}(file)

	runeToScript := make(map[rune]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			continue
		}
		code := strings.TrimSpace(parts[0])
		scriptParts := strings.Fields(parts[1])
		if len(scriptParts) == 0 {
			continue
		}
		script := scriptParts[0]

		if strings.Contains(code, "..") {
			rangeParts := strings.Split(code, "..")
			start, err := stringToRune(rangeParts[0])
			if err != nil {
				return nil, err
			}
			end, err := stringToRune(rangeParts[1])
			if err != nil {
				return nil, err
			}
			for r := start; r <= end; r++ {
				runeToScript[r] = script
			}
		} else {
			char, err := stringToRune(code)
			if err != nil {
				return nil, err
			}
			runeToScript[char] = script
		}
	}

	return runeToScript, nil
}

// stringToRune converts a string representing a hex value to a rune.
func stringToRune(hexStr string) (rune, error) {
	var r rune
	_, err := fmt.Sscanf(hexStr, "%X", &r)
	return r, err
}
