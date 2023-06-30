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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package user

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/stacklok/mediator/cmd/cli/app"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getUser(ctx context.Context, client pb.UserServiceClient, queryType string,
	id int32, username string, email string) (*pb.UserRecord, []*pb.GroupRecord, []*pb.RoleRecord, error) {
	var userRecord *pb.UserRecord
	var groups []*pb.GroupRecord
	var roles []*pb.RoleRecord

	if queryType == "id" {
		// get by id
		user, err := client.GetUserById(ctx, &pb.GetUserByIdRequest{
			Id: id,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		userRecord = user.User
		groups = user.Groups
		roles = user.Roles

	} else if queryType == "username" {
		// get by username
		user, err := client.GetUserByUserName(ctx, &pb.GetUserByUserNameRequest{
			Username: username,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		userRecord = user.User
		groups = user.Groups
		roles = user.Roles

	} else if queryType == "email" {
		user, err := client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{
			Email: email,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		userRecord = user.User
		groups = user.Groups
		roles = user.Roles
	} else if queryType == "personal" {
		user, err := client.GetUser(ctx, &pb.GetUserRequest{})
		if err != nil {
			return nil, nil, nil, err
		}
		userRecord = user.User
		groups = user.Groups
		roles = user.Roles
	}

	return userRecord, groups, roles, nil
}

type output struct {
	User   *pb.UserRecord    `json:"user"`
	Groups []*pb.GroupRecord `json:"groups"`
	Roles  []*pb.RoleRecord  `json:"roles"`
}

func printUser(user *pb.UserRecord, groups []*pb.GroupRecord, roles []*pb.RoleRecord, format string) {
	output := output{
		User:   user,
		Groups: groups,
		Roles:  roles,
	}
	if format == app.JSON {
		output, err := json.MarshalIndent(output, "", "  ")
		util.ExitNicelyOnError(err, "Error marshalling json")
		fmt.Println(string(output))
	} else if format == app.YAML {
		yamlData, err := yaml.Marshal(output)
		util.ExitNicelyOnError(err, "Error marshalling yaml")
		fmt.Println(string(yamlData))

	}
}

var user_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an user within a mediator control plane",
	Long: `The medic user get subcommand lets you retrieve details for an user within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		username := viper.GetString("username")
		email := viper.GetString("email")
		format := util.GetConfigValue("output", "output", cmd, "").(string)
		if format == "" {
			format = app.JSON
		}
		if format != app.JSON && format != app.YAML && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		count := 0

		if id > 0 {
			count++
		}
		if username != "" {
			count++
		}
		if email != "" {
			count++
		}
		if count > 1 {
			fmt.Fprintf(os.Stderr, "Error: must specify only one of id, username, or email\n")
			os.Exit(1)
		}

		var user *pb.UserRecord
		var groups []*pb.GroupRecord
		var roles []*pb.RoleRecord
		// get by id
		if id > 0 {
			user, groups, roles, err = getUser(ctx, client, "id", id, "", "")
			util.ExitNicelyOnError(err, "Error getting user by id")
			printUser(user, groups, roles, format)
		} else if username != "" {
			// get by username
			user, groups, roles, err = getUser(ctx, client, "username", 0, username, "")
			util.ExitNicelyOnError(err, "Error getting user by username")
			printUser(user, groups, roles, format)
		} else if email != "" {
			// get by email
			user, groups, roles, err = getUser(ctx, client, "email", 0, "", email)
			util.ExitNicelyOnError(err, "Error getting user by email")
			printUser(user, groups, roles, format)
		} else {
			// just get personal profile
			user, groups, roles, err = getUser(ctx, client, "personal", 0, "", "")
			util.ExitNicelyOnError(err, "Error getting personal user")
			printUser(user, groups, roles, format)
		}
	},
}

func init() {
	UserCmd.AddCommand(user_getCmd)
	user_getCmd.Flags().Int32P("id", "i", -1, "ID for the user to query")
	user_getCmd.Flags().StringP("username", "u", "", "Username for the user to query")
	user_getCmd.Flags().StringP("email", "e", "", "Email for the user to query")
	user_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
}
