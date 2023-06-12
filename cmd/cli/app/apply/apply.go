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

// Package apply provides the apply command for the medctl CLI
package apply

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/cmd/cli/app/group"
	"github.com/stacklok/mediator/cmd/cli/app/org"
	"github.com/stacklok/mediator/cmd/cli/app/role"
	"github.com/stacklok/mediator/cmd/cli/app/user"
	"github.com/stacklok/mediator/pkg/util"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type objectParameters struct {
	Object     string
	Parameters map[string]interface{}
}

func parseContent(data []byte) ([]objectParameters, error) {
	var objects []map[string]interface{}
	err := json.Unmarshal(data, &objects)
	if err != nil {
		// try with yaml
		err = yaml.Unmarshal(data, &objects)
		if err != nil {
			return nil, fmt.Errorf("failed to parse content: %w", err)
		}
	}

	var ret []objectParameters
	for _, object := range objects {
		for objectName, objectData := range object {
			ret = append(ret, objectParameters{
				Object:     objectName,
				Parameters: objectData.(map[string]interface{}),
			})
		}
	}
	return ret, nil
}

// ApplyCmd is the root command for the apply subcommands
var ApplyCmd = &cobra.Command{
	Use:   "apply (-f FILENAME)",
	Short: "Appy a configuration to a mediator control plane",
	Long:  `The medctl apply command applies a configuration to a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		f := util.GetConfigValue("file", "file", cmd, "").(string)

		var data []byte
		var err error

		if f == "-" {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %s\n", err)
				os.Exit(1)
			}
		} else {
			f = filepath.Clean(f)
			data, err = os.ReadFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %s\n", f, err)
				os.Exit(1)
			}
		}

		// try to unmarshal with json or yaml
		objects, err := parseContent(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing content: %s\n", err)
			os.Exit(1)
		}

		for _, object := range objects {
			// iterate over params and set viper values
			params := object.Parameters
			for k, v := range params {
				valueType := reflect.TypeOf(v)
				if valueType.Kind() == reflect.Int {
					v1 := v.(int)
					viper.Set(k, int32(v1))
				} else {
					viper.Set(k, v)
				}
			}

			if object.Object == "org" {
				org.Org_createCmd.Run(cmd, args)
			} else if object.Object == "role" {
				role.Role_createCmd.Run(cmd, args)
			} else if object.Object == "user" {
				user.User_createCmd.Run(cmd, args)
			} else if object.Object == "group" {
				group.Group_createCmd.Run(cmd, args)
			} else {
				fmt.Fprintf(os.Stderr, "Error: unknown object type %s\n", object.Object)
				os.Exit(1)
			}
		}
	},
}

func init() {
	app.RootCmd.AddCommand(ApplyCmd)
	ApplyCmd.Flags().StringP("file", "f", "", "Path to the configuration file to apply or - for stdin")
	if err := ApplyCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
