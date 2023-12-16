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

// The OpenFGA install via "go install" would be named "cli" instead of "fga"
// See https://github.com/openfga/cli/?tab=readme-ov-file#go
// This stub binary to makes it easier to install and run consistently from tools/bootstrap

package main

import (
	"github.com/openfga/cli/cmd"
)

func main() {
	cmd.Execute()
}