// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestDataSourceRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "datasource root command shows help",
			Args:           []string{"datasource"},
			GoldenFileName: "datasource_root.help",
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
