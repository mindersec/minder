//
// Copyright 2023 Stacklok, Inc.
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

package ruletype

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func initializeTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"Provider", "Project Name", "Id", "Name", "Description"})
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2, 3})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)

	return table
}

func renderRuleTypeTable(
	rt *minderv1.RuleType,
	table *tablewriter.Table,
) {
	row := []string{
		*rt.Context.Provider,
		*rt.Context.Project,
		*rt.Id,
		rt.Name,
		rt.Description,
	}
	table.Append(row)
}
