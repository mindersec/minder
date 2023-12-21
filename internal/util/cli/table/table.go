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

// Package table contains utilities for rendering tables
package table

import (
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	"github.com/stacklok/minder/internal/util/cli/table/simple"
)

const (
	// Simple is a simple table
	Simple = "simple"
)
const (
	// ColorRed is the color red
	ColorRed = "red"
	// ColorGreen is the color green
	ColorGreen = "green"
	// ColorYellow is the color yellow
	ColorYellow = "yellow"
)

// Table is an interface for rendering tables
type Table interface {
	AddRow(row []string)
	AddRowWithColor(row []string, rowColors []string)
	Render()
}

// New creates a new table
func New(_ string, layout layouts.TableLayout, header []string) Table {
	return simple.New(layout, header)
}
