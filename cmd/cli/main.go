// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides the entrypoint for the minder cli
package main

import (
	"github.com/mindersec/minder/cmd/cli/app"
	_ "github.com/mindersec/minder/cmd/cli/app/artifact"
	_ "github.com/mindersec/minder/cmd/cli/app/auth"
	_ "github.com/mindersec/minder/cmd/cli/app/auth/invite"
	_ "github.com/mindersec/minder/cmd/cli/app/auth/offline_token"
	_ "github.com/mindersec/minder/cmd/cli/app/docs"
	_ "github.com/mindersec/minder/cmd/cli/app/history"
	_ "github.com/mindersec/minder/cmd/cli/app/profile"
	_ "github.com/mindersec/minder/cmd/cli/app/profile/status"
	_ "github.com/mindersec/minder/cmd/cli/app/project"
	_ "github.com/mindersec/minder/cmd/cli/app/project/role"
	_ "github.com/mindersec/minder/cmd/cli/app/provider"
	_ "github.com/mindersec/minder/cmd/cli/app/quickstart"
	_ "github.com/mindersec/minder/cmd/cli/app/repo"
	_ "github.com/mindersec/minder/cmd/cli/app/ruletype"
	_ "github.com/mindersec/minder/cmd/cli/app/set_project"
	_ "github.com/mindersec/minder/cmd/cli/app/version"
)

func main() {
	app.Execute()
}
