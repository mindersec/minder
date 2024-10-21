// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package mindpak

import "fmt"

// BundleID groups together the pieces of information needed to identify a
// bundle. This cleans up the interfaces which deal with bundles.
type BundleID struct {
	Namespace string
	Name      string
}

// ID is a convenience function for creating BundleID instances
func ID(namespace string, name string) BundleID {
	return BundleID{
		Namespace: namespace,
		Name:      name,
	}
}

func (b BundleID) String() string {
	return fmt.Sprintf("%s/%s", b.Namespace, b.Name)
}
