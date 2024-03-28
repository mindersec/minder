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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func execOnOneRuleType(
	ctx context.Context,
	t table.Table,
	f string,
	dashOpen io.Reader,
	proj string,
	exec func(context.Context, string, *minderv1.RuleType) (*minderv1.RuleType, error),
) error {
	ctx, cancel := cli.GetAppContext(ctx, viper.GetViper())
	defer cancel()

	reader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	r, err := minderv1.ParseRuleType(reader)
	if err != nil {
		return fmt.Errorf("error parsing rule type: %w", err)
	}

	// Override the YAML specified project with the command line argument
	if proj != "" {
		if r.Context == nil {
			r.Context = &minderv1.Context{}
		}

		r.Context.Project = &proj
	}

	// create a rule
	rt, err := exec(ctx, f, r)
	if err != nil {
		return err
	}

	// add the rule type to the table rows
	t.AddRow(
		*rt.Context.Project,
		*rt.Id,
		rt.Name,
		cli.ConcatenateAndWrap(rt.Description, 20),
	)

	return nil
}

func validateFilesArg(files []string) error {
	if files == nil {
		return fmt.Errorf("error: file must be set")
	}

	if slices.Contains(files, "") {
		return fmt.Errorf("error: file must be set")
	}

	if slices.Contains(files, "-") && len(files) > 1 {
		return fmt.Errorf("error: cannot use stdin with other files")
	}

	return nil
}

func shouldSkipFile(f string) bool {
	// if the file is not json or yaml, skip it
	// Get file extension
	ext := filepath.Ext(f)
	switch ext {
	case ".yaml", ".yml", ".json":
		return false
	default:
		fmt.Fprintf(os.Stderr, "Skipping file %s: not a yaml or json file\n", f)
		return true
	}
}

// initializeTableForList initializes the table for the rule type
func initializeTableForList() table.Table {
	return table.New(table.Simple, layouts.RuleTypeList, nil)
}

// initializeTableForList initializes the table for the rule type
func initializeTableForOne() table.Table {
	return table.New(table.Simple, layouts.RuleTypeOne, nil)
}

func oneRuleTypeToRows(t table.Table, rt *minderv1.RuleType) {
	t.AddRow("ID", *rt.Id)
	t.AddRow("Name", rt.Name)
	t.AddRow("Description", rt.Description)
	t.AddRow("Project", *rt.Context.Project)
	t.AddRow("Ingest type", rt.Def.Ingest.Type)
	t.AddRow("Eval type", rt.Def.Eval.Type)
	rem := "unsupported"
	if rt.Def.GetRemediate() != nil {
		rem = rt.Def.GetRemediate().Type
	}
	t.AddRow("Remediation", rem)

	alert := "unsupported"
	if rt.Def.GetAlert() != nil {
		alert = rt.Def.GetAlert().Type
	}
	t.AddRow("Alert", alert)
	t.AddRow("Guidance", rt.Guidance)
}
