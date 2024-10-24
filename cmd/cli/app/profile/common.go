// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles"
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
