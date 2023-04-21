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

package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func callAuthURLService(address string, provider string) (string, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("error connecting to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.GetAuthorizationURL(ctx, &pb.AuthorizationURLRequest{
		Provider: provider,
	})
	if err != nil {
		return "", fmt.Errorf("error calling auth URL service: %v", err)
	}

	return resp.GetUrl(), nil
}

// authCmd represents the auth command
var auth_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an account in mediator",
	Long: `This command allows a user to create an account within mediator
, should you require an oauth2 login, then pass in the --provider flag,
e.g. --provider=github. This will then initiate the OAuth2 flow and allow
mediator to access user account details via the provider / iDP.`,
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := viper.GetString("grpc_server.host")
		grpc_port := viper.GetInt("grpc_server.port")
		provider := viper.GetString("provider")

		if cmd.Flags().Changed("grpc-host") {
			grpc_host, _ = cmd.Flags().GetString("grpc-host")
		}
		if cmd.Flags().Changed("grpc-port") {
			grpc_port, _ = cmd.Flags().GetInt("grpc-port")
		}
		if cmd.Flags().Changed("provider") {
			provider, _ = cmd.Flags().GetString("provider")
		}

		url, err := callAuthURLService(fmt.Sprintf("%s:%d", grpc_host, grpc_port), provider)
		if err != nil {
			log.Fatal(err)
		}

		err = browser.OpenURL(url)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	AuthCmd.AddCommand(auth_createCmd)

	if err := viper.BindPFlags(auth_createCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
