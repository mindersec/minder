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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func getUserById(ctx context.Context, client pb.UserServiceClient, id int32) (*pb.GetUserByIdResponse, error) {
	user, err := client.GetUserById(ctx, &pb.GetUserByIdRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

func getUserByUsername(ctx context.Context, client pb.UserServiceClient, username string) (*pb.GetUserByUserNameResponse, error) {
	user, err := client.GetUserByUserName(ctx, &pb.GetUserByUserNameRequest{
		Username: username,
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

func getUserByEmail(ctx context.Context, client pb.UserServiceClient, email string) (*pb.GetUserByEmailResponse, error) {
	user, err := client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{
		Email: email,
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

func getOwnUser(ctx context.Context, client pb.UserServiceClient) (*pb.GetUserResponse, error) {
	user, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		return nil, err
	}
	return user, err
}

func printUser(content []byte, format string) {
	if format == app.JSON {
		fmt.Println(string(content))
	} else if format == app.YAML {
		var rawMsg json.RawMessage
		err := json.Unmarshal(content, &rawMsg)
		util.ExitNicelyOnError(err, "Error unmarshalling json")
		yamlResult, err := util.ConvertJsonToYaml(rawMsg)
		util.ExitNicelyOnError(err, "Error converting json to yaml")
		fmt.Println(string(yamlResult))
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

		m := protojson.MarshalOptions{
			Indent: "  ",
		}
		// get by id
		if id > 0 {
			user, err := getUserById(ctx, client, id)
			util.ExitNicelyOnError(err, "Error getting user by id")
			out, err := m.Marshal(user)
			util.ExitNicelyOnError(err, "Error marshalling json")
			printUser(out, format)
		} else if username != "" {
			// get by username
			user, err := getUserByUsername(ctx, client, username)
			util.ExitNicelyOnError(err, "Error getting user by username")
			out, err := m.Marshal(user)
			util.ExitNicelyOnError(err, "Error marshalling json")
			printUser(out, format)
		} else if email != "" {
			// get by email
			user, err := getUserByEmail(ctx, client, email)
			util.ExitNicelyOnError(err, "Error getting user by email")
			out, err := m.Marshal(user)
			util.ExitNicelyOnError(err, "Error marshalling json")
			printUser(out, format)
		} else {
			// just get personal profile
			user, err := getOwnUser(ctx, client)
			util.ExitNicelyOnError(err, "Error getting personal user")
			out, err := m.Marshal(user)
			util.ExitNicelyOnError(err, "Error marshalling json")
			printUser(out, format)
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
