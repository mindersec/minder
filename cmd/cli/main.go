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

// Package main provides the entrypoint for the minder cli
package main

import (
	"github.com/stacklok/minder/cmd/cli/app"
	_ "github.com/stacklok/minder/cmd/cli/app/artifact"
	_ "github.com/stacklok/minder/cmd/cli/app/auth"
	_ "github.com/stacklok/minder/cmd/cli/app/auth/invite"
	_ "github.com/stacklok/minder/cmd/cli/app/auth/offline_token"
	_ "github.com/stacklok/minder/cmd/cli/app/docs"
	_ "github.com/stacklok/minder/cmd/cli/app/profile"
	_ "github.com/stacklok/minder/cmd/cli/app/profile/status"
	_ "github.com/stacklok/minder/cmd/cli/app/project"
	_ "github.com/stacklok/minder/cmd/cli/app/project/role"
	_ "github.com/stacklok/minder/cmd/cli/app/provider"
	_ "github.com/stacklok/minder/cmd/cli/app/quickstart"
	_ "github.com/stacklok/minder/cmd/cli/app/repo"
	_ "github.com/stacklok/minder/cmd/cli/app/ruletype"
	_ "github.com/stacklok/minder/cmd/cli/app/set_project"
	_ "github.com/stacklok/minder/cmd/cli/app/version"
)

func main() {
	app.Execute()
}
