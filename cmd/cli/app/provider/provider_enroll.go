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

package provider

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

	ghclient "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Response is the response from the OAuth callback server.
type Response struct {
	Status string `json:"status"`
}

// MAX_CALLS is the maximum number of calls to the gRPC server before stopping.
const MAX_CALLS = 300

// callBackServer starts a server and handler to listen for the OAuth callback.
// It will wait for either a success or failure response from the server.
func callBackServer(ctx context.Context, provider string, project string, port string,
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

			// create a shorter lived context for any client calls
			clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// todo: check if token has been created. We need an endpoint to pass an state and check if token is created
			res, err := client.VerifyProviderTokenFrom(clientCtx,
				&pb.VerifyProviderTokenFromRequest{Provider: provider, ProjectId: project, Timestamp: timestamppb.New(t)})
			if err == nil && res.Status == "OK" {
				return
			}
			if err != nil || res.Status == "OK" || calls >= MAX_CALLS {
				stopServer = true
			}
		}
	}()

}

var enrollProviderCmd = &cobra.Command{
	Use:   "enroll",
	Short: "Enroll a provider within the minder control plane",
	Long: `The minder provider enroll command allows a user to enroll a provider
such as GitHub into the minder control plane. Once enrolled, users can perform
actions such as adding repositories.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		msg, err := EnrollProviderCmd(cmd, args)
		util.ExitNicelyOnError(err, msg)
	},
}

// EnrollProviderCmd is the command for enrolling a provider
func EnrollProviderCmd(cmd *cobra.Command, _ []string) (string, error) {
	provider := util.GetConfigValue(viper.GetViper(), "provider", "provider", cmd, "").(string)
	if provider != ghclient.Github {
		msg := fmt.Sprintf("Only %s is supported at this time", ghclient.Github)
		return "", fmt.Errorf(msg)
	}
	project := viper.GetString("project")
	pat := util.GetConfigValue(viper.GetViper(), "token", "token", cmd, "").(string)
	owner := util.GetConfigValue(viper.GetViper(), "owner", "owner", cmd, "").(string)

	// Ask for confirmation if an owner is set on purpose
	ownerPromptStr := "your personal account"
	if owner != "" {
		ownerPromptStr = fmt.Sprintf("the %s organisation", owner)
	}
	yes := cli.PrintYesNoPrompt(cmd,
		fmt.Sprintf("You are about to enroll repositories from %s.", ownerPromptStr),
		"Do you confirm?",
		"Enroll operation cancelled.")
	if !yes {
		return "", nil
	}

	conn, err := util.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		return "Error getting grpc connection", err
	}
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)
	ctx, cancel := util.GetAppContext()
	defer cancel()
	oAuthCallbackCtx, oAuthCancel := context.WithTimeout(context.Background(), MAX_CALLS*time.Second)
	defer oAuthCancel()

	if pat != "" {
		// use pat for enrollment
		_, err := client.StoreProviderToken(context.Background(),
			&pb.StoreProviderTokenRequest{Provider: provider, ProjectId: project, AccessToken: pat, Owner: &owner})
		if err != nil {
			return "Error storing token", err
		}

		cli.PrintCmd(cmd, "Provider enrolled successfully")
		return "", nil
	}

	// Get random port
	port, err := rand.GetRandomPort()
	if err != nil {
		return "Error getting random port", err
	}

	resp, err := client.GetAuthorizationURL(ctx, &pb.GetAuthorizationURLRequest{
		Provider:  provider,
		ProjectId: project,
		Cli:       true,
		Port:      int32(port),
		Owner:     &owner,
	})
	if err != nil {
		return "Error getting authorization URL", err
	}

	fmt.Printf("Your browser will now be opened to: %s\n", resp.GetUrl())
	fmt.Println("Please follow the instructions on the page to complete the OAuth flow.")
	fmt.Println("Once the flow is complete, the CLI will close")
	fmt.Println("If this is a headless environment, please copy and paste the URL into a browser on a different machine.")

	if err := browser.OpenURL(resp.GetUrl()); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
		fmt.Println("Please copy and paste the URL into a browser.")
	}
	openTime := time.Now().Unix()

	var wg sync.WaitGroup
	wg.Add(1)

	go callBackServer(oAuthCallbackCtx, provider, project, fmt.Sprintf("%d", port), &wg, client, openTime)
	wg.Wait()

	cli.PrintCmd(cmd, "Provider enrolled successfully")
	return "", nil
}

func init() {
	ProviderCmd.AddCommand(enrollProviderCmd)
	enrollProviderCmd.Flags().StringP("provider", "p", "", "Name for the provider to enroll")
	enrollProviderCmd.Flags().StringP("project", "r", "", "ID of the project for enrolling the provider")
	enrollProviderCmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment")
	enrollProviderCmd.Flags().StringP("owner", "o", "", "Owner to filter on for provider resources")
	if err := enrollProviderCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
