// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
