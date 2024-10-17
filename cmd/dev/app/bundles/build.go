// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package bundles

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/build"
)

// CmdBuild is the build command
func CmdBuild() *cobra.Command {
	var buildCmd = &cobra.Command{
		Use:   "build bundle_id input output",
		Short: "build a mindpak bundle",
		Args:  cobra.ExactArgs(3),
		Long: `
The 'bundle build' subcommand allows you to build a mindpak bundle from the specified path.

Arguments:

build_id: an identifier of the form 'namespace/name@version'
input: Directory containing bundle profiles and rule types
output: Path to tar where bundle will be written
`,
		RunE:         buildCmdRun,
		SilenceUsage: true,
	}
	return buildCmd
}

func buildCmdRun(_ *cobra.Command, args []string) error {
	metadata, err := parseVersion(args[0])
	if err != nil {
		return err
	}
	packer := build.NewPacker()
	options := build.InitOptions{
		Metadata: metadata,
		Path:     args[1],
	}

	bundle, err := packer.InitBundle(&options)
	if err != nil {
		return err
	}

	err = packer.WriteToFile(bundle, args[2])
	if err != nil {
		return err
	}

	return nil
}

func parseVersion(id string) (*mindpak.Metadata, error) {
	firstSplit := strings.Split(id, "/")
	if len(firstSplit) != 2 {
		return nil, fmt.Errorf("invalid bundle id: %s", id)
	}
	secondSplit := strings.Split(firstSplit[1], "@")
	if len(secondSplit) != 2 {
		return nil, fmt.Errorf("invalid bundle id: %s", id)
	}

	// TODO: validate that names and versions meet requirements
	return &mindpak.Metadata{
		Namespace: firstSplit[0],
		Name:      secondSplit[0],
		Version:   secondSplit[1],
		// leave this as nil for now so that the builds are reproducible
		Date: nil,
	}, nil
}
