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

package profile

import (
	"fmt"
	"io"

	"github.com/olekukonko/tablewriter"

	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func execOnOneProfile(
	table *tablewriter.Table,
	f string,
	dashOpen io.Reader,
	project string,
	exec func(string, *minderv1.Profile) (*minderv1.Profile, error),
) error {
	preader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	p, err := parseProfile(preader, project)
	if err != nil {
		return fmt.Errorf("error parsing profile: %w", err)
	}

	// create a rule
	respprof, err := exec(f, p)
	if err != nil {
		return err
	}

	RenderProfileTable(respprof, table)
	return nil
}

func parseProfile(r io.Reader, proj string) (*minderv1.Profile, error) {
	p, err := engine.ParseYAML(r)
	if err != nil {
		return nil, fmt.Errorf("error reading profile from file: %w", err)
	}

	if proj != "" {
		if p.Context == nil {
			p.Context = &minderv1.Context{}
		}

		p.Context.Project = &proj
	}

	return p, nil
}
