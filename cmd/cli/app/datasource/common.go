// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// executeOnOneDataSource executes a function on a single data source
func executeOnOneDataSource(
	ctx context.Context,
	t table.Table,
	f string,
	dashOpen io.Reader,
	proj string,
	exec func(context.Context, string, *minderv1.DataSource) (*minderv1.DataSource, error),
) error {
	ctx, cancel := cli.GetAppContext(ctx, viper.GetViper())
	defer cancel()

	reader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	ds := &minderv1.DataSource{}
	if err := minderv1.ParseResourceProto(reader, ds); err != nil {
		return fmt.Errorf("error parsing data source: %w", err)
	}

	// Override the YAML specified project with the command line argument
	if proj != "" {
		if ds.Context == nil {
			ds.Context = &minderv1.ContextV2{}
		}
		ds.Context.ProjectId = proj
	}

	if err := ds.Validate(); err != nil {
		return fmt.Errorf("error validating data source: %w", err)
	}

	// create or update the data source
	createdDS, err := exec(ctx, f, ds)
	if err != nil {
		return err
	}

	// add the data source to the table rows
	name := appendDataSourcePropertiesToName(createdDS)
	t.AddRow(
		createdDS.Context.ProjectId,
		createdDS.Id,
		name,
	)

	return nil
}

// validateFilesArg validates the file arguments
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

// shouldSkipFile determines if a file should be skipped based on its extension
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

// appendDataSourcePropertiesToName appends the data source properties to the name.
func appendDataSourcePropertiesToName(ds *minderv1.DataSource) string {
	name := ds.Name
	var properties []string

	if dType := ds.GetDriverType(); dType != "" {
		properties = append(properties, fmt.Sprintf("type: %s", dType))
	}

	if len(properties) != 0 {
		return fmt.Sprintf("%s\n(%s)", name, strings.Join(properties, ", "))
	}

	return name
}

// initializeTableForList initializes the table for listing data sources
func initializeTableForList(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Project ID", "ID", "Name", "Description"}).
		SetAutoMerge(true)
}
