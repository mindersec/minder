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

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getUser(ctx context.Context, client pb.UserServiceClient, queryType string,
	id int32, username string, email string) *pb.UserRecord {
	var userRecord *pb.UserRecord

	if queryType == "id" {
		// get by id
		user, err := client.GetUserById(ctx, &pb.GetUserByIdRequest{
			Id: id,
		})
		if err == nil {
			userRecord = user.User
		}
	} else if queryType == "username" {
		// get by username
		user, err := client.GetUserByUserName(ctx, &pb.GetUserByUserNameRequest{
			Username: username,
		})
		if err == nil {
			userRecord = user.User
		}
	} else if queryType == "email" {
		user, err := client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{
			Email: email,
		})
		if err == nil {
			userRecord = user.User
		}
	}

	return userRecord
}

var user_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an user within a mediator control plane",
	Long: `The medctl user get subcommand lets you retrieve details for an user within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		conn, err := util.GetGrpcConnection(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		username := viper.GetString("username")
		email := viper.GetString("email")
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

		// check no option selected, or more than one
		if count == 0 {
			fmt.Fprintf(os.Stderr, "Error: must specify one of id, username, or email\n")
			os.Exit(1)
		}
		if count > 1 {
			fmt.Fprintf(os.Stderr, "Error: must specify only one of id, username, or email\n")
			os.Exit(1)
		}

		var user *pb.UserRecord
		// get by id
		if id > 0 {
			user = getUser(ctx, client, "id", id, "", "")
		} else if username != "" {
			// get by username
			user = getUser(ctx, client, "username", 0, username, "")
		} else if email != "" {
			// get by email
			user = getUser(ctx, client, "email", 0, "", email)
		}

		if user == nil {
			fmt.Fprintf(os.Stderr, "Error getting user\n")
			os.Exit(1)
		}
		json, err := json.Marshal(user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshalling user: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(string(json))

	},
}

func init() {
	UserCmd.AddCommand(user_getCmd)
	user_getCmd.Flags().Int32P("id", "i", -1, "ID for the user to query")
	user_getCmd.Flags().StringP("username", "u", "", "Username for the user to query")
	user_getCmd.Flags().StringP("email", "e", "", "Email for the user to query")
}
