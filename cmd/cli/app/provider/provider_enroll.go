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

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Response is the response from the OAuth callback server.
type Response struct {
	Status string `json:"status"`
}

// MAX_WAIT is the maximum number of calls to the gRPC server before stopping.
const MAX_WAIT = time.Duration(5 * time.Minute)

var enrollCmd = &cobra.Command{
	Use:   "enroll",
	Short: "Enroll a provider within the minder control plane",
	Long: `The minder provider enroll command allows a user to enroll a provider
such as GitHub into the minder control plane. Once enrolled, users can perform
actions such as adding repositories.`,
	RunE: cli.GRPCClientWrapRunE(EnrollProviderCommand),
}

// EnrollProviderCommand is the command for enrolling a provider
func EnrollProviderCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewOAuthServiceClient(conn)
	provcli := minderv1.NewProvidersServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	token := viper.GetString("token")
	owner := viper.GetString("owner")
	yesFlag := viper.GetBool("yes")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Ask for confirmation if an owner is set on purpose
	ownerPromptStr := "your personal account"
	if owner != "" {
		ownerPromptStr = fmt.Sprintf("the %s organisation", owner)
	}

	if !yesFlag {
		yes := cli.PrintYesNoPrompt(cmd,
			fmt.Sprintf("You are about to enroll repositories from %s.", ownerPromptStr),
			"Do you confirm?",
			"Enroll operation cancelled.",
			true)
		if !yes {
			return nil
		}
	}

	prov, err := provcli.GetProvider(ctx, &minderv1.GetProviderRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		Name:    provider,
	})
	if err != nil {
		return cli.MessageAndError("Error getting provider", err)
	}

	if token != "" {
		if !prov.Provider.SupportsAuthFlow(minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT) {
			return fmt.Errorf("provider %s does not support token enrollment", provider)
		}

		return enrollUsingToken(ctx, cmd, client, provider, project, token, owner)
	}

	if !prov.Provider.SupportsAuthFlow(
		minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW) {
		return fmt.Errorf("provider %s does not support OAuth2 enrollment", provider)
	}

	// This will have a different timeout
	enrollemntCtx := cmd.Context()

	return enrollUsingOAuth2Flow(enrollemntCtx, cmd, client, provider, project, owner)
}

func enrollUsingToken(
	ctx context.Context,
	cmd *cobra.Command,
	client minderv1.OAuthServiceClient,
	provider string,
	project string,
	token string,
	owner string,
) error {
	_, err := client.StoreProviderToken(ctx, &minderv1.StoreProviderTokenRequest{
		Context:     &minderv1.Context{Provider: &provider, Project: &project},
		AccessToken: token,
		Owner:       &owner,
	})
	if err != nil {
		return cli.MessageAndError("Error storing token", err)
	}

	cmd.Println("Provider enrolled successfully")
	return nil
}

func enrollUsingOAuth2Flow(
	ctx context.Context,
	cmd *cobra.Command,
	client minderv1.OAuthServiceClient,
	provider string,
	project string,
	owner string,
) error {
	oAuthCallbackCtx, oAuthCancel := context.WithTimeout(ctx, MAX_WAIT+5*time.Second)
	defer oAuthCancel()

	// Get random port
	port, err := rand.GetRandomPort()
	if err != nil {
		return cli.MessageAndError("Error getting random port", err)
	}

	resp, err := client.GetAuthorizationURL(ctx, &minderv1.GetAuthorizationURLRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		Cli:     true,
		Port:    int32(port),
		Owner:   &owner,
	})
	if err != nil {
		return cli.MessageAndError("error getting authorization URL", err)
	}

	cmd.Printf("Your browser will now be opened to: %s\n", resp.GetUrl())
	cmd.Println("Please follow the instructions on the page to complete the OAuth flow.")
	cmd.Println("Once the flow is complete, the CLI will close")
	cmd.Println("If this is a headless environment, please copy and paste the URL into a browser on a different machine.")

	if err := browser.OpenURL(resp.GetUrl()); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
		cmd.Println("Please copy and paste the URL into a browser.")
	}
	openTime := time.Now()

	done := make(chan bool)

	go callBackServer(oAuthCallbackCtx, cmd, provider, project, fmt.Sprintf("%d", port), done, client, openTime)

	success := <-done

	if success {
		cmd.Println("Provider enrolled successfully")
	} else {
		cmd.Println("Failed to enroll provider")
	}
	return nil
}

// callBackServer starts a server and handler to listen for the OAuth callback.
// It will wait for either a success or failure response from the server.
func callBackServer(ctx context.Context, cmd *cobra.Command, provider string, project string, port string,
	done chan bool, client minderv1.OAuthServiceClient, openTime time.Time) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: time.Second * 10, // Set an appropriate timeout value
	}

	// Start the server in a goroutine
	cmd.Println("Listening for OAuth Login flow to complete on port", port)
	go func() {
		_ = server.ListenAndServe()
	}()

	// Start a goroutine for periodic gRPC calls
	defer func() {
		if err := server.Close(); err != nil {
			// Handle the error appropriately, such as logging or returning an error message.
			cmd.Printf("Error closing server: %s", err)
		}
		close(done)
	}()

	for {
		time.Sleep(time.Second)

		// create a shorter lived context for any client calls
		clientCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// todo: check if token has been created. We need an endpoint to pass an state and check if token is created
		res, err := client.VerifyProviderTokenFrom(clientCtx, &minderv1.VerifyProviderTokenFromRequest{
			Context:   &minderv1.Context{Provider: &provider, Project: &project},
			Timestamp: timestamppb.New(openTime),
		})
		if err == nil && res.Status == "OK" {
			done <- true
			return
		}
		if err != nil || res.Status == "OK" {
			cmd.Printf("Error calling server: %s\n", err)
			done <- false
			break
		}
		if time.Now().After(openTime.Add(MAX_WAIT)) {
			cmd.Printf("Timeout waiting for OAuth flow to complete...\n")
			done <- false
			break
		}
	}
}

func init() {
	ProviderCmd.AddCommand(enrollCmd)
	// Flags
	enrollCmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment")
	enrollCmd.Flags().StringP("owner", "o", "", "Owner to filter on for provider resources")
	enrollCmd.Flags().BoolP("yes", "y", false, "Bypass yes/no prompt when enrolling new provider")
	// Bind flags
	if err := viper.BindPFlag("token", enrollCmd.Flags().Lookup("token")); err != nil {
		enrollCmd.Printf("Error binding flag: %s", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("owner", enrollCmd.Flags().Lookup("owner")); err != nil {
		enrollCmd.Printf("Error binding flag: %s", err)
		os.Exit(1)
	}
}
