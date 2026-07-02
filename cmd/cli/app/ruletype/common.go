// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

func execOnOneRuleType(
	cmd *cobra.Command,
	t table.Table,
	f util.ExpandedFile,
	proj string,
	exec func(context.Context, string, *minderv1.RuleType) (*minderv1.RuleType, error),
) error {
	resources, err := fileconvert.ReadFromPath(cmd.Printf, f)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		r, ok := resource.(*minderv1.RuleType)
		if !ok {
			return fmt.Errorf("file is a different type of resource: %T", resource)
		}

		// Override the YAML specified project with the command line argument
		if proj != "" {
			if r.Context == nil {
				r.Context = &minderv1.Context{}
			}

			r.Context.Project = &proj
		}

		// create a rule
		rt, err := exec(cmd.Context(), f.Path, r)
		if err != nil {
			return err
		}

		// add the rule type to the table rows
		name := appendRuleTypePropertiesToName(rt)

		t.AddRow(
			name,
			rt.Def.InEntity,
			rt.Description,
		)
	}
	return nil
}

func validateFilesArg(files []string) error {
	if files == nil {
		return fmt.Errorf("error: file must be set")
	}

	if slices.Contains(files, "") {
		return fmt.Errorf("error: file must be nonempty")
	}

	if slices.Contains(files, "-") && len(files) > 1 {
		return fmt.Errorf("error: cannot use stdin with other files")
	}

	return nil
}

func printWarnings(cmd *cobra.Command, warnings []string) {
	for _, warning := range warnings {
		cmd.PrintErrf("Warning: %s\n", warning)
	}
}

func shouldSkipFile(f string) bool {
	// if the file is not json or yaml, skip it
	// Get file extension
	ext := filepath.Ext(f)
	switch ext {
	case ".yaml", ".yml", ".json", ".rego":
		return false
	default:
		fmt.Fprintf(os.Stderr, "Skipping file %s: not a yaml, json or rego file\n", f)
		return true
	}
}

// initializeTableForList initializes the table for the rule type
func initializeTableForList(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Name", "Entity Type", "Description"}).
		SetAutoMerge(true).
		SeparateRows()
}

// initializeTableForOne initializes the table for the rule type
func initializeTableForOne(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Rule Type", "Details"})
}

func oneRuleTypeToRows(t table.Table, rt *minderv1.RuleType) {
	t.AddRow("Name", rt.Name)
	t.AddRow("ID", *rt.Id)
	t.AddRow("Applicable Entity", rt.GetDef().InEntity)
	releasePhaseString := ruleTypeReleasePhaseToString(rt.ReleasePhase)
	if releasePhaseString != "" {
		t.AddRow("Release phase", releasePhaseString)
	}
	t.AddRow("Description", rt.Description)
	t.AddRow("Ingest type", rt.Def.Ingest.Type)
	t.AddRow("Eval type", rt.Def.Eval.Type)
	t.AddRow("Guidance", cli.RenderMarkdown(rt.Guidance, cli.WidthFraction(50.0)))
	t.AddRow("Remediation", cmp.Or(rt.Def.GetRemediate().GetType(), "unsupported"))
	t.AddRow("Alert", cmp.Or(rt.Def.GetAlert().GetType(), "unsupported"))
}

func ruleTypeReleasePhaseToString(phase minderv1.RuleTypeReleasePhase) string {
	var phaseString string
	switch phase {
	case minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED:
		phaseString = ""
	case minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA:
		phaseString = "alpha"
	case minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA:
		phaseString = "beta"
	case minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_GA:
		phaseString = ""
	case minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_DEPRECATED:
		phaseString = "deprecated"
	}
	return phaseString
}

// appendRuleTypePropertiesToName appends the rule type properties to the name. The format is:
// <name> (<properties>)
// where <properties> is a comma separated list of properties.
func appendRuleTypePropertiesToName(rt *minderv1.RuleType) string {
	name := rt.Name
	properties := []string{}
	// add the release_phase property if it is present
	phase := ruleTypeReleasePhaseToString(rt.ReleasePhase)
	if phase != "" {
		properties = append(properties, fmt.Sprintf("release_phase: %s", phase))
	}

	// add the can_remediate: false property if remediation is not supported
	if rt.Def.GetRemediate() == nil {
		properties = append(properties, "can_remediate: false")
	}

	// return the name with the properties if any
	if len(properties) != 0 {
		return fmt.Sprintf("%s\n(%s)", name, strings.Join(properties, ", "))
	}

	// return only name otherwise
	return name
}
