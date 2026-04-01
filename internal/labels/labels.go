// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package labels contains logic for parsing label filters.
package labels

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidLabel represents an error when a label contains forbidden characters or patterns.
	ErrInvalidLabel = errors.New("invalid label identifier")
)

// ParseLabelFilter parses a comma-separated label filter string into lists of
// labels to include and exclude. It respects rules such as rejecting `!*` and
// mixing `*` with other inclusion labels.
func ParseLabelFilter(filter string) (include []string, exclude []string, err error) {
	if filter == "" {
		return nil, nil, nil
	}

	for _, label := range strings.Split(filter, ",") {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}

		inc, exc, err := ParseLabel(label)
		if err != nil {
			return nil, nil, err
		}

		if inc != "" {
			if inc == "*" && len(include) != 0 {
				return nil, nil, fmt.Errorf("%w: cannot mix * with other labels", ErrInvalidLabel)
			}
			if inc != "*" && len(include) == 1 && include[0] == "*" {
				return nil, nil, fmt.Errorf("%w: cannot mix * with other labels", ErrInvalidLabel)
			}
			include = append(include, inc)
		}
		if exc != "" {
			exclude = append(exclude, exc)
		}
	}

	return include, exclude, nil
}

// ParseLabel parses a single label (without commas) into an include or exclude string.
// Returns the include label (if any), the exclude label (if any), and an error if validation fails.
func ParseLabel(label string) (include string, exclude string, err error) {
	if label == "" {
		return "", "", nil
	}
	if label == "!*" {
		return "", "", fmt.Errorf("%w: !* is not allowed", ErrInvalidLabel)
	}
	if strings.HasPrefix(label, "!") {
		return "", strings.TrimPrefix(label, "!"), nil
	}
	return label, "", nil
}
