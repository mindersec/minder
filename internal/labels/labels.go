// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package labels contains logic for parsing label filters.
package labels

import (
	"strings"
)

// ParseLabelFilter parses a comma-separated label filter string into lists of
// labels to include and exclude. It resolves wildcards so that if any inclusion
// rule is `*`, the included labels list evaluates simply to `["*"]`.
func ParseLabelFilter(filter string) (include []string, exclude []string) {
	if filter == "" {
		return nil, nil
	}

	var starMatched bool
	for _, label := range strings.Split(filter, ",") {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}

		inc, exc := ParseLabel(label)

		if inc != "" {
			if inc == "*" {
				starMatched = true
			} else {
				include = append(include, inc)
			}
		}
		if exc != "" {
			exclude = append(exclude, exc)
		}
	}

	if starMatched {
		include = []string{"*"}
	}

	return include, exclude
}

// ParseLabel parses a single label (without commas) into an include or exclude string.
// Returns the include label (if any) and the exclude label (if any).
func ParseLabel(label string) (include string, exclude string) {
	if strings.HasPrefix(label, "!") {
		return "", strings.TrimPrefix(label, "!")
	}
	return label, ""
}
