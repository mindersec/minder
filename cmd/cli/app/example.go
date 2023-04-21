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

package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func callWebhookService(address string) {
	url := fmt.Sprintf("http://%s/api/v1/github/hook", address)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Fatalf("Error calling webhook service: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading webhook response: %v", err)
	}

	fmt.Printf("Webhook response: %s\n", string(body))
}

func callHealthService(address string) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error connecting to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewHealthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.CheckHealth(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Error calling health service: %v", err)
	}

	fmt.Printf("Health service response: %s\n", resp.GetStatus())
}

func callAuthURLService(address string) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error connecting to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.GetAuthorizationURL(ctx, &pb.AuthorizationURLRequest{})
	if err != nil {
		log.Fatalf("Error calling auth URL service: %v", err)
	}

	fmt.Printf("Auth URL service response: %s\n", resp.GetUrl())
}

// exampleCmd represents the example command
var exampleCmd = &cobra.Command{
	Use:   "example",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		http_host := viper.GetString("http_server.host")
		http_port := viper.GetInt("http_server.port")
		grpc_host := viper.GetString("grpc_server.host")
		grpc_port := viper.GetInt("grpc_server.port")

		if cmd.Flags().Changed("http-host") {
			http_host, _ = cmd.Flags().GetString("http-host")
		}
		if cmd.Flags().Changed("http-port") {
			http_port, _ = cmd.Flags().GetInt("http-port")
		}
		if cmd.Flags().Changed("grpc-host") {
			grpc_host, _ = cmd.Flags().GetString("grpc-host")
		}
		if cmd.Flags().Changed("grpc-port") {
			grpc_port, _ = cmd.Flags().GetInt("grpc-port")
		}

		callWebhookService(fmt.Sprintf("%s:%d", http_host, http_port))
		callHealthService(fmt.Sprintf("%s:%d", grpc_host, grpc_port))
		callAuthURLService(fmt.Sprintf("%s:%d", grpc_host, grpc_port))
	},
}

func init() {
	RootCmd.AddCommand(exampleCmd)
	exampleCmd.PersistentFlags().String("http-host", "", "Server host")
	exampleCmd.PersistentFlags().Int("http-port", 0, "Server port")
	exampleCmd.PersistentFlags().String("grpc-host", "", "Server host")
	exampleCmd.PersistentFlags().Int("grpc-port", 0, "Server port")
	if err := viper.BindPFlags(exampleCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
