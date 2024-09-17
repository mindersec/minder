// Copyright 2024 Stacklok, Inc.
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

package db

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// Scan implements the sql.Scanner interface for the SelectorInfo struct
func (s *ProfileSelector) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	// Convert the value to a string
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SelectorInfo: %v", value)
	}
	str := string(bytes)

	// Remove the parentheses
	str = strings.TrimPrefix(str, "(")
	str = strings.TrimSuffix(str, ")")

	// Split the string by commas to get the individual field values
	cr := csv.NewReader(strings.NewReader(str))
	parts, err := cr.Read()
	if err != nil {
		return fmt.Errorf("failed to scan SelectorInfo: %v", err)
	}

	// Assign the values to the struct fields
	if len(parts) != 5 {
		return fmt.Errorf("failed to scan SelectorInfo: unexpected number of fields")
	}

	if err := s.ID.Scan(parts[0]); err != nil {
		return fmt.Errorf("failed to scan id: %v", err)
	}

	if err := s.ProfileID.Scan(parts[1]); err != nil {
		return fmt.Errorf("failed to scan profile_id: %v", err)
	}

	s.Entity = NullEntities{}
	if parts[2] != "" {
		if err := s.Entity.Scan(parts[2]); err != nil {
			return fmt.Errorf("failed to scan entity: %v", err)
		}
	}

	selector := strings.TrimPrefix(parts[3], "\"")
	selector = strings.TrimSuffix(selector, "\"")
	selector = strings.ReplaceAll(selector, `""`, `"`)
	s.Selector = selector

	comment := strings.TrimPrefix(parts[4], "\"")
	comment = strings.TrimSuffix(comment, "\"")
	s.Comment = comment

	return nil
}
