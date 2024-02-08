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

// Package domain contains the domain logic for the homoglyphs rule type
package domain

import (
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain/resources"
)

// InvisibleCharactersProcessor is a processor for the invisible characters rule type
type InvisibleCharactersProcessor struct {
	invisibleCharacters map[rune]struct{}
}

// NewInvisibleCharactersProcessor creates a new InvisibleCharactersProcessor
func NewInvisibleCharactersProcessor() *InvisibleCharactersProcessor {
	return &InvisibleCharactersProcessor{
		invisibleCharacters: resources.InvisibleCharacters,
	}
}

// FindInvisibleCharacters checks for invisible characters in the given line
// and returns a slice of runes representing the invisible characters found.
func (ice *InvisibleCharactersProcessor) FindInvisibleCharacters(line string) []rune {
	invisibleChars := make([]rune, 0)
	for _, r := range line {
		if _, exists := ice.invisibleCharacters[r]; exists {
			invisibleChars = append(invisibleChars, r)
		}
	}

	return invisibleChars
}
