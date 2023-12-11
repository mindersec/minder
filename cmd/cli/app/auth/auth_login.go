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
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/gorilla/securecookie"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/constants"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

//go:embed html/login_success.html
var loginSuccessHtml []byte

func userRegistered(ctx context.Context, client pb.UserServiceClient) (bool, *pb.GetUserResponse, error) {
	res, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return false, nil, nil
			}
		}
		return false, nil, fmt.Errorf("error retrieving user %w", err)
	}
	return true, res, nil
}

// auth_loginCmd represents the login command
var auth_loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to a minder control plane.",
	Long: `Login to a minder control plane. Upon successful login, credentials
will be saved to $XDG_CONFIG_HOME/minder/credentials.json`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			cli.Print(cmd.ErrOrStderr(), "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		issuerUrlStr := util.GetConfigValue(viper.GetViper(), "identity.cli.issuer_url", "identity-url", cmd,
			constants.IdentitySeverURL).(string)
		clientID := util.GetConfigValue(viper.GetViper(), "identity.cli.client_id", "identity-client", cmd, "minder-cli").(string)

		parsedURL, err := url.Parse(issuerUrlStr)
		if err != nil {
			return cli.MessageAndError(cmd, "Error parsing issuer URL", err)
		}

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
			return cli.MessageAndError(cmd, "Error getting random port", err)
		}

		parsedURL, err = url.Parse(fmt.Sprintf("http://localhost:%v", port))
		if err != nil {
			return cli.MessageAndError(cmd, "Error parsing callback URL", err)
		}
		redirectURI := parsedURL.JoinPath(callbackPath)

		provider, err := rp.NewRelyingPartyOIDC(issuerUrl.String(), clientID, "", redirectURI.String(), scopes, options...)
		if err != nil {
			return cli.MessageAndError(cmd, "Error creating relying party", err)
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

		callback := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string,
			rp rp.RelyingParty) {

			tokenChan <- tokens
			// send a success message to the browser
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, err := w.Write(loginSuccessHtml)
			if err != nil {
				// if we cannot display the success page, just print a success message
				cli.PrintCmd(cmd, "Authentication Successful")
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
		cli.PrintCmd(cmd, "Your browser will now be opened to: %s", loginUrl)
		cli.PrintCmd(cmd, "Please follow the instructions on the page to log in.")

		// open user's browser to login page
		if err := browser.OpenURL(loginUrl); err != nil {
			cli.PrintCmd(cmd, "You may login by pasting this URL into your browser: %s", loginUrl)
		}

		cli.PrintCmd(cmd, "Waiting for token...\n")

		// wait for the token to be received
		token := <-tokenChan

		// save credentials
		filePath, err := util.SaveCredentials(util.OpenIdCredentials{
			AccessToken:          token.AccessToken,
			RefreshToken:         token.RefreshToken,
			AccessTokenExpiresAt: token.Expiry,
		})
		if err != nil {
			cli.Print(cmd.ErrOrStderr(), "couldn't save credentials: %s\n", err)
		}

		conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return cli.MessageAndError(cmd, "Error getting grpc connection", err)
		}
		defer conn.Close()
		client := pb.NewUserServiceClient(conn)

		// check if the user already exists in the local database
		registered, userInfo, err := userRegistered(ctx, client)
		if err != nil {
			return cli.MessageAndError(cmd, "Error checking if user exists", err)
		}

		if !registered {
			cli.PrintCmd(cmd, "First login, registering user...\n")
			newUser, err := client.CreateUser(ctx, &pb.CreateUserRequest{})
			if err != nil {
				return cli.MessageAndError(cmd, "Error registering user", err)
			}

			cli.PrintCmd(cmd, cli.SuccessBanner.Render(
				"You have been successfully registered. Welcome!"))
			cli.PrintCmd(cmd, cli.WarningBanner.Render(
				"Minder is currently under active development and considered experimental, "+
					" we therefore provide no data retention or service stability guarantees.",
			))
			cli.PrintCmd(cmd, cli.Header.Render("Here are your details:"))

			renderNewUser(cmd, conn, newUser)
		} else {
			cli.PrintCmd(cmd, cli.SuccessBanner.Render(
				"You have successfully logged in."))

			cli.PrintCmd(cmd, cli.Header.Render("Here are your details:"))
			renderUserInfo(cmd, conn, userInfo)
		}

		cli.PrintCmd(cmd, "Your access credentials have been saved to %s", filePath)

		// shut down the HTTP server
		// TODO: should this use the app context?
		return server.Shutdown(context.Background())
	},
}

func renderNewUser(cmd *cobra.Command, conn *grpc.ClientConn, newUser *pb.CreateUserResponse) {

	rows := []table.Row{
		{"Project ID", newUser.ProjectId},
		{"Project Name", newUser.ProjectName},
		{"Minder Server", conn.Target()},
	}

	renderUserToTable(cmd, rows)
}

func renderUserInfo(cmd *cobra.Command, conn *grpc.ClientConn, user *pb.GetUserResponse) {
	minderSrvKey := "Minder Server"
	rows := []table.Row{
		{
			minderSrvKey, conn.Target(),
		},
	}

	rows = append(rows, getProjectTableRows(user.Projects)...)

	renderUserToTable(cmd, rows)
}

func renderUserToTable(cmd *cobra.Command, rows []table.Row) {
	columns := []table.Column{
		{Title: "Key", Width: cli.KeyValTableWidths.Key},
		{Title: "Value", Width: cli.KeyValTableWidths.Value},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)),
		table.WithStyles(cli.TableHiddenSelectStyles),
	)

	cli.PrintCmd(cmd, cli.TableRender(t))
}

func init() {
	AuthCmd.AddCommand(auth_loginCmd)
}
