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

package auth

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"

	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

//go:embed html/login_success.html
var loginSuccessHtml []byte

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Minder",
	Long: `The login command allows for logging in to Minder. Upon successful login, credentials will be saved to
$XDG_CONFIG_HOME/minder/credentials.json`,
	RunE: LoginCommand,
}

// LoginCommand is the login subcommand
func LoginCommand(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return cli.MessageAndError("Unable to read config", err)
	}

	issuerUrlStr := clientConfig.Identity.CLI.IssuerUrl
	clientID := clientConfig.Identity.CLI.ClientId

	parsedURL, err := url.Parse(issuerUrlStr)
	if err != nil {
		return cli.MessageAndError("Error parsing issuer URL", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	issuerUrl := parsedURL.JoinPath("realms/stacklok")
	scopes := []string{"openid"}
	callbackPath := "/auth/callback"

	// create encrypted cookie handler to mitigate CSRF attacks
	hashKey := securecookie.GenerateRandomKey(32)
	encryptKey := securecookie.GenerateRandomKey(32)
	cookieHandler := httphelper.NewCookieHandler(hashKey, encryptKey, httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode))
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithPKCE(cookieHandler),
	}

	// Get random port
	port, err := rand.GetRandomPort()
	if err != nil {
		return cli.MessageAndError("Error getting random port", err)
	}

	parsedURL, err = url.Parse(fmt.Sprintf("http://localhost:%v", port))
	if err != nil {
		return cli.MessageAndError("Error parsing callback URL", err)
	}
	redirectURI := parsedURL.JoinPath(callbackPath)

	provider, err := rp.NewRelyingPartyOIDC(issuerUrl.String(), clientID, "", redirectURI.String(), scopes, options...)
	if err != nil {
		return cli.MessageAndError("Error creating relying party", err)
	}

	stateFn := func() string {
		state, err := mcrypto.GenerateNonce()
		if err != nil {
			cmd.PrintErrln("error generating state for login")
			os.Exit(1)
		}
		return state
	}

	tokenChan := make(chan *oidc.Tokens[*oidc.IDTokenClaims])

	callback := func(w http.ResponseWriter, _ *http.Request,
		tokens *oidc.Tokens[*oidc.IDTokenClaims], _ string, _ rp.RelyingParty) {

		tokenChan <- tokens
		// send a success message to the browser
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write(loginSuccessHtml)
		if err != nil {
			// if we cannot display the success page, just print a success message
			cmd.Println("Authentication Successful")
		}
	}
	http.Handle("/login", rp.AuthURLHandler(stateFn, provider))
	http.Handle(callbackPath, rp.CodeExchangeHandler(callback, provider))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		ReadHeaderTimeout: time.Second * 10,
	}
	// Start the server in a goroutine
	go func() {
		_ = server.ListenAndServe()
	}()
	// get the OAuth authorization URL
	loginUrl := fmt.Sprintf("http://localhost:%v/login", port)

	// Redirect user to provider to log in
	cmd.Printf("Your browser will now be opened to: %s\n", loginUrl)
	cmd.Println("Please follow the instructions on the page to log in.")

	// open user's browser to login page
	if err := browser.OpenURL(loginUrl); err != nil {
		cmd.Printf("You may login by pasting this URL into your browser: %s\n", loginUrl)
	}

	cmd.Println("Waiting for token...")

	// wait for the token to be received
	token := <-tokenChan

	// save credentials
	filePath, err := util.SaveCredentials(util.OpenIdCredentials{
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpiresAt: token.Expiry,
	})
	if err != nil {
		cmd.PrintErrf("couldn't save credentials: %s\n", err)
	}

	conn, err := cli.GrpcForCommand(viper.GetViper())
	if err != nil {
		return cli.MessageAndError("Error getting grpc connection", err)
	}
	defer conn.Close()
	client := minderv1.NewUserServiceClient(conn)

	// check if the user already exists in the local database
	registered, userInfo, err := userRegistered(ctx, client)
	if err != nil {
		return cli.MessageAndError("Error checking if user exists", err)
	}

	if !registered {
		cmd.Println("First login, registering user...")
		newUser, err := client.CreateUser(ctx, &minderv1.CreateUserRequest{})
		if err != nil {
			return cli.MessageAndError("Error registering user", err)
		}

		cmd.Println(cli.SuccessBanner.Render(
			"You have been successfully registered. Welcome!"))
		cmd.Println(cli.WarningBanner.Render(
			"Minder is currently under active development and considered experimental, " +
				" we therefore provide no data retention or service stability guarantees.",
		))
		cmd.Println(cli.Header.Render("Here are your details:"))

		renderNewUser(conn.Target(), newUser)
	} else {
		cmd.Println(cli.SuccessBanner.Render(
			"You have successfully logged in."))

		cmd.Println(cli.Header.Render("Here are your details:"))
		renderUserInfo(conn.Target(), userInfo)
	}

	cmd.Printf("Your access credentials have been saved to %s\n", filePath)

	// shut down the HTTP server
	// TODO: should this use the app context?
	return server.Shutdown(context.Background())
}

func init() {
	AuthCmd.AddCommand(loginCmd)
}
