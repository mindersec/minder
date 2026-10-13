// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestRuleTypeDescriptionMarkdown(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"get", "create"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			response := &minderv1.ListRuleTypesResponse{}
			cli.LoadFixture(t, "mock_ruletypes_response.json", response)
			rt := response.RuleTypes[0]
			rt.Description = "First line\nsecond line.\n\nNext paragraph.\n\n- First item\n- Second item"
			var out bytes.Buffer
			tbl := initializeTableForOne(&out)
			if mode == "get" {
				oneRuleTypeToRows(tbl, rt)
			} else {
				tbl = initializeTableForList(&out)
				err := execOnOneRuleType(&cobra.Command{}, tbl,
					util.ExpandedFile{Path: filepath.Join("fixture", "rule_type_sample.yaml")}, "",
					func(context.Context, string, *minderv1.RuleType) (*minderv1.RuleType, error) { return rt, nil })
				require.NoError(t, err)
			}
			tbl.Render()
			require.Contains(t, out.String(), "First line second line.")
			require.Regexp(t, `First line second line\.[^\n]*\n[^\n]*│\s*\n[^\n]*Next paragraph\.`, out.String())
			require.Regexp(t, `First item[^\n]*\n[^\n]*Second item`, out.String())
		})
	}
}

//nolint:paralleltest // Each case sets the terminal width through the environment.
func TestDescriptionWrapsAtTableWidth(t *testing.T) {
	for _, width := range []string{"100", "140"} {
		t.Run(width, func(t *testing.T) {
			t.Setenv("COLUMNS", width)
			var out bytes.Buffer
			tbl := initializeTableForList(&out)
			description := strings.Repeat("A paragraph with several different words\n", 6)
			tbl.AddRowWithFormatter(2, renderDescription, "example", "repository", description)
			tbl.Render()

			var lines []string
			cellWidth := 0
			for _, row := range strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")[2:] {
				cells := strings.Split(row, "│")
				require.Len(t, cells, 3)
				cellWidth = len([]rune(cells[2])) - 2 // Exclude cell padding.
				if line := strings.TrimSpace(cells[2]); line != "" {
					lines = append(lines, line)
				}
			}
			require.Equal(t, strings.Join(strings.Fields(description), " "), strings.Join(lines, " "))
			for i := 0; i < len(lines)-1; i++ {
				nextWord := strings.Fields(lines[i+1])[0]
				require.Greater(t, len(lines[i])+1+len(nextWord), cellWidth,
					"line %q wrapped before the available column width", lines[i])
			}
		})
	}
}
