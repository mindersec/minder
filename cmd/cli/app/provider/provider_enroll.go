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

	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Response is the response from the OAuth callback server.
type Response struct {
	Status string `json:"status"`
}

const (
	// MAX_WAIT is the maximum number of calls to the gRPC server before stopping.
	MAX_WAIT = time.Duration(5 * time.Minute)

	// legacyGitHubProvider is the legacy GitHub OAuth provider class
	legacyGitHubProvider = minderv1.ProviderClass_PROVIDER_CLASS_GITHUB
)

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
	oauthClient := minderv1.NewOAuthServiceClient(conn)
	providerClient := minderv1.NewProvidersServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	token := viper.GetString("token")
	owner := viper.GetString("owner")
	yesFlag := viper.GetBool("yes")
	skipBrowser := viper.GetBool("skip-browser")

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

	if token != "" {
		return enrollUsingToken(ctx, cmd, oauthClient, provider, project, token, owner)
	}

	// This will have a different timeout
	enrollemntCtx := cmd.Context()

	return enrollUsingOAuth2Flow(enrollemntCtx, cmd, oauthClient, providerClient, provider, project, owner, skipBrowser)
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
	oauthClient minderv1.OAuthServiceClient,
	providerClient minderv1.ProvidersServiceClient,
	provider string,
	project string,
	owner string,
	skipBrowser bool,
) error {
	oAuthCallbackCtx, oAuthCancel := context.WithTimeout(ctx, MAX_WAIT+5*time.Second)
	defer oAuthCancel()
	legacyProviderEnrolled, err := hasLegacyProvider(ctx, providerClient, project)
	if err != nil {
		return cli.MessageAndError("Error getting existing providers", err)
	}

	// If the user is using the legacy GitHub provider, don't let them enroll a new provider.
	// However, they may update the credentials for the existing provider.
	if legacyProviderEnrolled && provider != legacyGitHubProvider.ToString() {
		return fmt.Errorf("it seems you are using the legacy github provider. " +
			"If you would like to enroll a new provider, please delete your existing provider by " +
			"running \"minder provider delete --name github\"")
	}

	// Get random port
	port, err := rand.GetRandomPort()
	if err != nil {
		return cli.MessageAndError("Error getting random port", err)
	}

	resp, err := oauthClient.GetAuthorizationURL(ctx, &minderv1.GetAuthorizationURLRequest{
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

	if !skipBrowser {
		if err := browser.OpenURL(resp.GetUrl()); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
			cmd.Println("Please copy and paste the URL into a browser.")
		}
	}
	openTime := time.Now()

	done := make(chan bool)

	go callBackServer(oAuthCallbackCtx, cmd, project, port, done, oauthClient, openTime, resp.GetState())

	success := <-done

	if success {
		cmd.Println("Provider enrolled successfully")
	} else {
		cmd.Println("Failed to enroll provider")
	}
	return nil
}

func hasLegacyProvider(ctx context.Context, providerClient minderv1.ProvidersServiceClient, project string) (bool, error) {
	cursor := ""

	for {
		resp, err := providerClient.ListProviders(ctx, &minderv1.ListProvidersRequest{
			Context: &minderv1.Context{
				Project: &project,
			},
			Cursor: cursor,
		})
		if err != nil {
			return false, err
		}

		// check if a legacy GitHub provider is already enrolled
		for _, p := range resp.Providers {
			if p.GetClass() == legacyGitHubProvider.ToString() &&
				p.GetCredentialsState() == minderv1.CredentialsState_CREDENTIALS_STATE_SET.ToString() {
				return true, nil
			}
		}

		if resp.Cursor == "" {
			break
		}

		cursor = resp.Cursor
	}
	return false, nil
}

// callBackServer starts a server and handler to listen for the OAuth callback.
// It will wait for either a success or failure response from the server.
func callBackServer(ctx context.Context, cmd *cobra.Command, project string, port int, done chan bool,
	client minderv1.OAuthServiceClient, openTime time.Time, state string) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: time.Second * 10, // Set an appropriate timeout value
	}

	// Start the server in a goroutine
	cmd.Println("Listening for enrollment flow to complete on port", port)
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

		res, err := client.VerifyProviderCredential(clientCtx, &minderv1.VerifyProviderCredentialRequest{
			// do not set the provider because it may not be created yet
			Context:         &minderv1.Context{Project: &project},
			EnrollmentNonce: state,
		})
		if err == nil && res.Created {
			done <- true
			return
		}
		if err != nil || res.Created {
			cmd.Printf("Error calling server: %s\n", err)
			done <- false
			break
		}
		if time.Now().After(openTime.Add(MAX_WAIT)) {
			cmd.Printf("Timeout waiting for enrollment flow to complete...\n")
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

	// hidden flags
	enrollCmd.Flags().BoolP("skip-browser", "", false, "Skip opening the browser for OAuth flow")
}
