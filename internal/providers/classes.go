// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import "sort"

// ListProviderClasses returns a list of provider classes.
func ListProviderClasses() []string {
	defs := ListProviderClassDefinitions()
	classes := make([]string, 0, len(defs))

	for class := range defs {
		classes = append(classes, class)
	}

	// Stable ordering keeps API output and tests deterministic.
	sort.Strings(classes)

	return classes
}
