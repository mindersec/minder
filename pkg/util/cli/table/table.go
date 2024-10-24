// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package table contains utilities for rendering tables
package table

import (
	"github.com/mindersec/minder/pkg/util/cli/table/layouts"
	"github.com/mindersec/minder/pkg/util/cli/table/simple"
)

const (
	// Simple is a simple table
	Simple = "simple"
)

// Table is an interface for rendering tables
type Table interface {
	AddRow(row ...string)
	AddRowWithColor(row ...layouts.ColoredColumn)
	Render()
}

// New creates a new table
func New(_ string, layout layouts.TableLayout, header []string) Table {
	return simple.New(layout, header)
}
