//
// Copyright 2024 Stacklok, Inc.
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

package role

import (
	"github.com/spf13/cobra"
)

var mappingCmd = &cobra.Command{
	Use:     "mapping",
	Aliases: []string{"map"},
	Short:   "Map roles to a user using claims",
	Long: `The minder project role mapping functionality allows one to map roles
to a user using claims.`,
}

func init() {
	RoleCmd.AddCommand(mappingCmd)
}
