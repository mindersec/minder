//
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

import (
	"encoding/json"
	"fmt"
	"io"
)

// Write writes the bundle manifest to a file
func (m *Manifest) Write(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	if err := e.Encode(&m); err != nil {
		return fmt.Errorf("encoding bundle manifest: %w", err)
	}

	return nil
}

// Read loads the manifest data by parsing json data from reader r
func (m *Manifest) Read(r io.Reader) error {
	dec := json.NewDecoder(r)
	if err := dec.Decode(m); err != nil {
		return fmt.Errorf("decoding manifest: %w", err)
	}
	return nil
}
