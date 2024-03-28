// Copyright 2024 Stacklok, Inc.
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

package bundles

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stacklok/minder/internal/util/ptr"
	"github.com/stacklok/minder/pkg/mindpak"
	"github.com/stacklok/minder/pkg/mindpak/build"
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
		Date:      ptr.Ptr(time.Now()),
	}, nil
}
