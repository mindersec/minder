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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
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

	// githubAppProvider is the name used to identify the new GitHub App provider class
	githubAppProvider = minderv1.ProviderClass_PROVIDER_CLASS_GITHUB_APP
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
func EnrollProviderCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	oauthClient := minderv1.NewOAuthServiceClient(conn)
	providerClient := minderv1.NewProvidersServiceClient(conn)

	// TODO: get rid of provider flag, only use class
	provider := viper.GetString("provider")
	if provider == "" {
		provider = viper.GetString("class")
	}
	providerName := viper.GetString("name")
	if providerName == "" {
		providerName = provider
	}
	project := viper.GetString("project")
	token := viper.GetString("token")
	owner := viper.GetString("owner")
	yesFlag := viper.GetBool("yes")
	skipBrowser := viper.GetBool("skip-browser")
	cfgFlag := viper.GetString("provider-config")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Ask for confirmation if an owner is set on purpose
	ownerPromptStr := "your personal account"
	if owner != "" {
		ownerPromptStr = fmt.Sprintf("the %s organisation", owner)
	}

	// Only show this option for the legacy flow
	// TODO: split this into multiple subcommands so we do not need to have
	// checks like this per flag
	// The Github App flow will ask these questions in the browser
	if !yesFlag && provider == legacyGitHubProvider.ToString() {
		yes := cli.PrintYesNoPrompt(cmd,
			fmt.Sprintf("You are about to enroll repositories from %s.", ownerPromptStr),
			"Do you confirm?",
			"Enroll operation cancelled.",
			true)
		if !yes {
			return nil
		}
	}

	config, err := providerConfigFromArg(cfgFlag, cmd.InOrStdin())
	if err != nil {
		return cli.MessageAndError("Error reading provider configuration", err)
	}

	// the token only applies to the old flow
	// TODO: allow for token to be passed in if the provider allows it, don't hardcode
	userFlow, err := supportsToken(provider)
	if err != nil {
		return cli.MessageAndError("Error checking provider support", err)
	}

	if token != "" && userFlow {
		return enrollUsingToken(ctx, cmd, oauthClient, providerClient, providerName, provider, project, token, owner, config)
	}

	// This will have a different timeout
	enrollemntCtx := cmd.Context()

	return enrollUsingOAuth2Flow(
		enrollemntCtx, cmd, oauthClient, providerClient, providerName, provider, project, owner, skipBrowser, config)
}

func enrollUsingToken(
	ctx context.Context,
	cmd *cobra.Command,
	client minderv1.OAuthServiceClient,
	provClient minderv1.ProvidersServiceClient,
	providerName string,
	providerClass string,
	project string,
	token string,
	owner string,
	providerConfig *structpb.Struct,
) error {
	_, err := provClient.CreateProvider(ctx, &minderv1.CreateProviderRequest{
		Context: &minderv1.Context{Provider: &providerName, Project: &project},
		Provider: &minderv1.Provider{
			Name:   providerName,
			Class:  providerClass,
			Config: providerConfig,
		},
	})
	st, ok := status.FromError(err)
	if !ok {
		// We can't parse the error, so just return it
		return err
	}

	if st.Code() != codes.AlreadyExists {
		return err
	}

	// the provider already exists, turn this call into an update of the token
	_, err = client.StoreProviderToken(ctx, &minderv1.StoreProviderTokenRequest{
		Context:     &minderv1.Context{Provider: &providerName, Project: &project},
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
	providerName string,
	providerClass string,
	project string,
	owner string,
	skipBrowser bool,
	providerConfig *structpb.Struct,
) error {
	oAuthCallbackCtx, oAuthCancel := context.WithTimeout(ctx, MAX_WAIT+5*time.Second)
	defer oAuthCancel()
	legacyProviderEnrolled, err := hasLegacyProvider(ctx, providerClient, project)
	if err != nil {
		return cli.MessageAndError("Error getting existing providers", err)
	}

	// If the user is using the legacy GitHub provider, don't let them enroll a new provider.
	// However, they may update the credentials for the existing provider.
	if legacyProviderEnrolled && providerName != legacyGitHubProvider.ToString() {
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
		Context:       &minderv1.Context{Provider: &providerName, Project: &project},
		Cli:           true,
		Port:          port,
		Owner:         &owner,
		Config:        providerConfig,
		ProviderClass: providerClass,
	})
	if err != nil {
		return cli.MessageAndError("error getting authorization URL", err)
	}

	cmd.Printf("Your browser will now be opened to: %s\n", resp.GetUrl())
	cmd.Println("Please follow the instructions on the page to complete the enrollment flow.")
	cmd.Println("Once the flow is complete, the CLI will close")
	cmd.Println("If this is a headless environment, please copy and paste the URL into a browser on a different machine.")
	cmd.Printf("Enrollment will time out after %.1f minutes.\n", MAX_WAIT.Minutes())

	if !skipBrowser {
		if err := browser.OpenURL(resp.GetUrl()); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
			cmd.Println("Please copy and paste the URL into a browser.")
		}
	}
	openTime := time.Now()

	done := make(chan bool)

	go callBackServer(oAuthCallbackCtx, cmd, project, int(port), done, oauthClient, openTime, resp.GetState())

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

func providerConfigFromArg(configSource string, dashReader io.Reader) (*structpb.Struct, error) {
	if configSource == "" {
		return nil, nil
	}

	reader, closer, err := util.OpenFileArg(configSource, dashReader)
	if err != nil {
		return nil, fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	var config map[string]any

	// TODO: handle YAML and JSON
	err = json.NewDecoder(reader).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error parsing provider configuration: %w", err)
	}

	return structpb.NewStruct(config)
}

func supportsToken(providerClass string) (bool, error) {
	providerDef, err := providers.GetProviderClassDefinition(providerClass)
	if err != nil {
		return false, err
	}
	return slices.Contains(providerDef.AuthorizationFlows, db.AuthorizationFlowUserInput), nil
}

func init() {
	ProviderCmd.AddCommand(enrollCmd)
	// Flags
	enrollCmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment (Legacy GitHub only)")
	enrollCmd.Flags().StringP("owner", "o", "", "Owner to filter on for provider resources (Legacy GitHub only)")
	enrollCmd.Flags().BoolP("yes", "y", false, "Bypass any yes/no prompts when enrolling a new provider")
	enrollCmd.Flags().StringP("class", "c", githubAppProvider.ToString(), "Provider class, defaults to github-app")
	enrollCmd.Flags().StringP("provider-config", "f", "", "Path to the provider configuration (or - for stdin)")
	enrollCmd.Flags().StringP("name", "n", "", "Name of the new provider. (Only when using a token)")

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
