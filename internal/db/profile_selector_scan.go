// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	cr.LazyQuotes = true // Enable LazyQuotes to allow for uneven number of quotes
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
