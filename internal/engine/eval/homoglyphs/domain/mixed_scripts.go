// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package domain

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed resources/scripts.txt
var scriptsContent embed.FS

// MixedScriptsProcessor is a processor for the mixed scripts rule type
type MixedScriptsProcessor struct {
	runeToScript map[rune]string
}

// NewMixedScriptsProcessor creates a new MixedScriptsProcessor
func NewMixedScriptsProcessor() (*MixedScriptsProcessor, error) {
	// 7th of Feb, 2024: https://www.unicode.org/Public/UCD/latest/ucd/Scripts.txt
	runeToScript, err := loadScriptData("resources/scripts.txt")
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
func (mse *MixedScriptsProcessor) FindMixedScripts(line string) []MixedScriptInfo {
	words := strings.Fields(line)
	mixedScripts := make([]MixedScriptInfo, 0)

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
			mixedScripts = append(mixedScripts, MixedScriptInfo{
				Text:         word,
				ScriptsFound: scripts,
			})
		}
	}

	return mixedScripts
}

// loadScriptData reads data from the specified file in Scripts.txt format and populates a runeToScript map.
// The function parses each line of the file, ignoring comments and empty lines.
// It expects lines in the format "<code>; <script> # <description>", where <code> can be a single character
// or a range. For each valid line, it updates runeToScript to map characters (or ranges of characters) to their
// respective scripts. Lines that do not conform to the expected format or contain no script information are skipped.
func loadScriptData(filePath string) (map[rune]string, error) {
	file, err := scriptsContent.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file fs.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
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
