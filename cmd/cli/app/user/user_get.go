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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func getUserById(ctx context.Context, client pb.UserServiceClient, id int32) (*pb.GetUserByIdResponse, error) {
	user, err := client.GetUserById(ctx, &pb.GetUserByIdRequest{
		UserId: id,
	})
	if err != nil {
		return nil, err
	}
	return user, err
}

func getUserBySubject(ctx context.Context, client pb.UserServiceClient, subject string) (*pb.GetUserBySubjectResponse, error) {
	user, err := client.GetUserBySubject(ctx, &pb.GetUserBySubjectRequest{
		Subject: subject,
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

func printUser(user protoreflect.ProtoMessage, format string) {
	if format == app.JSON {
		out, err := util.GetJsonFromProto(user)
		util.ExitNicelyOnError(err, "Error getting json from proto")
		fmt.Println(out)
	} else if format == app.YAML {
		out, err := util.GetYamlFromProto(user)
		util.ExitNicelyOnError(err, "Error getting yaml from proto")
		fmt.Println(out)
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
		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		subject := viper.GetString("subject")
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
		if subject != "" {
			count++
		}
		if count > 1 {
			fmt.Fprintf(os.Stderr, "Error: must specify only one of id or subject\n")
			os.Exit(1)
		}

		// get by id
		var result protoreflect.ProtoMessage
		if id > 0 {
			user, err := getUserById(ctx, client, id)
			util.ExitNicelyOnError(err, "Error getting user by id")
			result = user
		} else if subject != "" {
			// get by subject
			user, err := getUserBySubject(ctx, client, subject)
			util.ExitNicelyOnError(err, "Error getting user by subject")
			result = user
		} else {
			// just get personal profile
			user, err := getOwnUser(ctx, client)
			util.ExitNicelyOnError(err, "Error getting personal user")
			result = user
		}
		printUser(result, format)
	},
}

func init() {
	UserCmd.AddCommand(user_getCmd)
	user_getCmd.Flags().Int32P("id", "i", -1, "ID for the user to query")
	user_getCmd.Flags().StringP("username", "u", "", "Username for the user to query")
	user_getCmd.Flags().StringP("email", "e", "", "Email for the user to query")
	user_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
}
