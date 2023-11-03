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

package rule_type

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/exp/slices"

	"github.com/stacklok/mediator/internal/util"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

func execOnOneRuleType(
	table *tablewriter.Table,
	f string,
	dashOpen io.Reader,
	exec func(string, *minderv1.RuleType) (*minderv1.RuleType, error),
) error {
	preader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	r, err := minderv1.ParseRuleType(preader)
	if err != nil {
		return fmt.Errorf("error parsing rule type: %w", err)
	}

	// create a rule
	resprt, err := exec(f, r)
	if err != nil {
		return err
	}

	renderRuleTypeTable(resprt, table)
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
