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

// Package resources contains resources used by the homoglyphs evaluators.
package resources

// InvisibleCharacters is a map of all unicode characters that are considered invisible.
// Reference: https://invisible-characters.com/
// Note, the following characters are not included in the invisibleCharacters:
// '\u0009' (Character Tabulation or Tab)
// '\u0020' (Space)
// These two characters are common ASCII characters and are not necessarily
// considered malicious or a security threat in most contexts.
var InvisibleCharacters = map[rune]struct{}{
	'\u00A0': {}, // No-break space
	'\u00AD': {}, // Soft hyphen
	'\u034F': {}, // Combining grapheme joiner
	'\u061C': {}, // Arabic letter mark
	'\u115F': {}, // Hangul choseong filler
	'\u1160': {}, // Hangul jungseong filler
	'\u17B4': {}, // Khmer vowel inherent aq
	'\u17B5': {}, // Khmer vowel inherent aa
	'\u180E': {}, // Mongolian vowel separator
	'\u2000': {}, // En quad
	'\u2001': {}, // Em quad
	'\u2002': {}, // En space
	'\u2003': {}, // Em space
	'\u2004': {}, // Three-per-em space
	'\u2005': {}, // Four-per-em space
	'\u2006': {}, // Six-per-em space
	'\u2007': {}, // Figure space
	'\u2008': {}, // Punctuation space
	'\u2009': {}, // Thin space
	'\u200A': {}, // Hair space
	'\u200B': {}, // Zero width space
	'\u200C': {}, // Zero width non-joiner
	'\u200D': {}, // Zero width joiner
	'\u200E': {}, // Left-to-right mark
	'\u200F': {}, // Right-to-left mark
	'\u202F': {}, // Narrow no-break space
	'\u205F': {}, // Medium mathematical space
	'\u2060': {}, // Text joiner
	'\u2061': {}, // Function application
	'\u2062': {}, // Invisible times
	'\u2063': {}, // Invisible separator
	'\u2064': {}, // Invisible plus
	'\u206A': {}, // Inhibit symmetric swapping
	'\u206B': {}, // Activate symmetric swapping
	'\u206C': {}, // Inhibit arabic form shaping
	'\u206D': {}, // Activate arabic form shaping
	'\u206E': {}, // National digit shapes
	'\u206F': {}, // Nominal digit shapes
	'\u3000': {}, // Ideographic space
	'\u2800': {}, // Braille pattern blank
	'\u3164': {}, // Hangul filler
	'\uFEFF': {}, // Zero width no-break space
	'\uFFA0': {}, // Halfwidth hangul filler

	// The format \UXXXXXXXX represents 32-bit values, where each X is a hexadecimal digit.
	// If a codepoint doesn't have exactly four or eight hexadecimal digits,
	// it should be padded with zeros on the left until it reaches the required length.
	'\U0001D159': {}, // Musical symbol null notehead
	'\U0001D173': {}, // Musical symbol begin beam
	'\U0001D174': {}, // Musical symbol end beam
	'\U0001D175': {}, // Musical symbol begin tie
	'\U0001D176': {}, // Musical symbol end tie
	'\U0001D177': {}, // Musical symbol begin slur
	'\U0001D178': {}, // Musical symbol end slur
	'\U0001D179': {}, // Musical symbol begin phrase
	'\U0001D17A': {}, // Musical symbol end phrase
}
