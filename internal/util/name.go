// Copyright (C) 2024 The Dagu Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package util

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// https://github.com/sindresorhus/filename-reserved-regex/blob/master/index.js

var (
	reservedCharRegex = regexp.MustCompile(
		`[<>:"/\\|!?*.\x00-\x1F]`,
	)
	reservedNamesRegex = regexp.MustCompile(
		`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`,
	)
)

const (
	// MaxSafeNameLength is the maximum length of a safe filename
	MaxSafeNameLength = 100
	// HashLength is the length of the hash appended to the filename
	HashLength = 5
)

// SafeName converts a string to a safe filename
func SafeName(str string) string {
	// Convert to lowercase and remove non-allowed characters
	safe := strings.ToLower(str)

	// Replace reserved names
	safe = reservedCharRegex.ReplaceAllString(safe, "_")

	// Replace reserved Windows names
	safe = reservedNamesRegex.ReplaceAllStringFunc(safe, func(s string) string {
		return "_" + s + "_"
	})

	// Replace spaces and non-printable characters with underscores
	safe = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) && r != ' ' {
			return r
		}
		return '_'
	}, safe)

	// Generate a 5-character hash
	hash := generateHash(str)

	// Truncate to a safe length (100 is generally safe)
	// Use runes to safely truncate multi-byte characters
	runes := []rune(safe)
	if len(runes) > MaxSafeNameLength-HashLength-1 {
		safe = string(runes[:MaxSafeNameLength-HashLength-1])
	}

	// Append the hash
	safe = safe + "_" + hash

	// Ensure the last character is not a partial Unicode character
	return strings.TrimRight(safe, "\uFFFD")
}

// generateHash creates a 5-character hash from the input string using SHA-256
func generateHash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])[0:HashLength]
}
