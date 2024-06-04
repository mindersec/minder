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

package profile

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ExecOnOneProfile is a helper function to execute a function on a single profile
func ExecOnOneProfile(ctx context.Context, t table.Table, f string, dashOpen io.Reader, project string,
	exec func(context.Context, string, *minderv1.Profile) (*minderv1.Profile, error),
) (*minderv1.Profile, error) {
	ctx, cancel := cli.GetAppContext(ctx, viper.GetViper())
	defer cancel()

	reader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return nil, fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	p, err := parseProfile(reader, project)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile: %w", err)
	}

	// create a rule
	profile, err := exec(ctx, f, p)
	if err != nil {
		return nil, err
	}

	RenderProfileTable(profile, t)
	return profile, nil
}

func parseProfile(r io.Reader, proj string) (*minderv1.Profile, error) {
	p, err := profiles.ParseYAML(r)
	if err != nil {
		return nil, fmt.Errorf("error reading profile from file: %w", err)
	}

	// Override the YAML specified project with the command line argument
	if proj != "" {
		if p.Context == nil {
			p.Context = &minderv1.Context{}
		}

		p.Context.Project = &proj
	}

	return p, nil
}
