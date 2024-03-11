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

package cli

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// StringArgSetter is an interface for setting string arguments
type StringArgSetter interface {
	StringP(string, string, string, string) *string
	Lookup(string) *pflag.Flag
}

// UseProjectFlag adds a project flag to the provided flag set and binds it to the viper instance
func UseProjectFlag(s StringArgSetter, v *viper.Viper) {
	s.StringP("project", "j", "", "ID of the project")
	if err := v.BindPFlag("project", s.Lookup("project")); err != nil {
		panic(fmt.Sprintf("Error binding project flag: %s", err))
	}
}
