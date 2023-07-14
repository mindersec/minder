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

package enroll

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// Response is the response from the OAuth callback server.
type Response struct {
	Status string `json:"status"`
}

// MAX_CALLS is the maximum number of calls to the gRPC server before stopping.
const MAX_CALLS = 300

// callBackServer starts a server and handler to listen for the OAuth callback.
// It will wait for either a success or failure response from the server.
func callBackServer(ctx context.Context, provider string, group int32, port string,
	wg *sync.WaitGroup, client pb.OAuthServiceClient, since int64) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: time.Second * 10, // Set an appropriate timeout value
	}

	go func() {
		wg.Wait()
		err := server.Close()
		if err != nil {
			// Handle the error appropriately, such as logging or returning an error message.
			fmt.Printf("Error closing server: %s", err)
		}
	}()

	// Start the server in a goroutine
	fmt.Println("Listening for OAuth Login flow to complete on port", port)
	go func() {
		_ = server.ListenAndServe()
	}()

	var stopServer bool
	// Start a goroutine for periodic gRPC calls
	go func() {
		defer wg.Done()
		calls := 0

		for {
			// Perform periodic gRPC calls
			if stopServer {
				// Check the stop condition and break the loop if necessary
				break
			}

			time.Sleep(time.Second)
			t := time.Unix(since, 0)
			calls++

			// todo: check if token has been created. We need an endpoint to pass an state and check if token is created
			res, err := client.VerifyProviderTokenFrom(ctx,
				&pb.VerifyProviderTokenFromRequest{Provider: provider, GroupId: group, Timestamp: timestamppb.New(t)})
			if err != nil || res.Status == "OK" || calls >= MAX_CALLS {
				stopServer = true
			}
		}
	}()

}

var enrollProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Enroll a provider within the mediator control plane",
	Long: `The medic enroll provider command allows a user to enroll a provider
such as GitHub into the mediator control plane. Once enrolled, users can perform
actions such as adding repositories.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		if provider != ghclient.Github {
			fmt.Fprintf(os.Stderr, "Only %s is supported at this time\n", ghclient.Github)
			os.Exit(1)
		}
		group := util.GetConfigValue("group-id", "group-id", cmd, 0).(int)
		pat := util.GetConfigValue("token", "token", cmd, "").(string)

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewOAuthServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		if pat != "" {
			// use pat for enrollment
			_, err := client.StoreProviderToken(context.Background(),
				&pb.StoreProviderTokenRequest{Provider: provider, GroupId: int32(group), AccessToken: pat})
			util.ExitNicelyOnError(err, "Error storing token")
			fmt.Println("Provider enrolled successfully")
		} else {
			// Get random port
			port, err := util.GetRandomPort()
			util.ExitNicelyOnError(err, "Error getting random port")

			resp, err := client.GetAuthorizationURL(ctx, &pb.GetAuthorizationURLRequest{
				Provider: provider,
				GroupId:  int32(group),
				Cli:      true,
				Port:     int32(port),
			})
			util.ExitNicelyOnError(err, "Error getting authorization URL")

			fmt.Printf("Your browser will now be opened to: %s\n", resp.GetUrl())
			fmt.Println("Please follow the instructions on the page to complete the OAuth flow.")
			fmt.Println("Once the flow is complete, the CLI will close")
			fmt.Println("If this is a headless environment, please copy and paste the URL into a browser on a different machine.")

			util.ExitNicelyOnError(err, "Error opening browser")

			if err := browser.OpenURL(resp.GetUrl()); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
				os.Exit(1)
			}
			openTime := time.Now().Unix()

			var wg sync.WaitGroup
			wg.Add(1)

			go callBackServer(ctx, provider, int32(group), fmt.Sprintf("%d", port), &wg, client, openTime)
			wg.Wait()
		}
	},
}

func init() {
	EnrollCmd.AddCommand(enrollProviderCmd)
	enrollProviderCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	enrollProviderCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for enrolling the provider")
	enrollProviderCmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment")
	if err := enrollProviderCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
