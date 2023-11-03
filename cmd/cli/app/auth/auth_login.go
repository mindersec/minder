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
	"net/http"
	"net/url"
	"strings"
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

	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/rand"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func userRegistered(ctx context.Context, client pb.UserServiceClient) (bool, *pb.GetUserResponse, error) {
	res, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return false, nil, nil
			}
		}
		return false, nil, fmt.Errorf("error retrieving user %v", err)
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
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		issuerUrlStr := util.GetConfigValue(viper.GetViper(), "identity.cli.issuer_url", "identity-url", cmd,
			"https://auth.staging.stacklok.dev").(string)
		realm := util.GetConfigValue(viper.GetViper(), "identity.cli.realm", "identity-realm", cmd, "stacklok").(string)
		clientID := util.GetConfigValue(viper.GetViper(), "identity.cli.client_id", "identity-client", cmd, "minder-cli").(string)

		parsedURL, err := url.Parse(issuerUrlStr)
		util.ExitNicelyOnError(err, "Error parsing issuer URL")
		issuerUrl := parsedURL.JoinPath("realms", realm)
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
		util.ExitNicelyOnError(err, "Error getting random port")

		parsedURL, err = url.Parse(fmt.Sprintf("http://localhost:%v", port))
		util.ExitNicelyOnError(err, "Error creating callback server")
		redirectURI := parsedURL.JoinPath(callbackPath)

		provider, err := rp.NewRelyingPartyOIDC(issuerUrl.String(), clientID, "", redirectURI.String(), scopes, options...)
		util.ExitNicelyOnError(err, "error creating identity provider reference")

		stateFn := func() string {
			state, err := mcrypto.GenerateNonce()
			util.ExitNicelyOnError(err, "error generating state for login")
			return state
		}

		tokenChan := make(chan *oidc.Tokens[*oidc.IDTokenClaims])

		callback := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string,
			rp rp.RelyingParty) {

			tokenChan <- tokens
			msg := "<div><h2>Authentication successful</h2><div>You may now close this tab and return to your terminal.</div></div>"
			// send a success message to the browser
			fmt.Fprint(w, msg)
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
			fmt.Println(err)
		}

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()
		client := pb.NewUserServiceClient(conn)

		// check if the user already exists in the local database
		registered, userInfo, err := userRegistered(ctx, client)
		util.ExitNicelyOnError(err, "Error fetching user")

		if !registered {
			cli.PrintCmd(cmd, "First login, registering user...\n")
			newUser, err := client.CreateUser(ctx, &pb.CreateUserRequest{})
			util.ExitNicelyOnError(err, "Error registering user")

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
		err = server.Shutdown(context.Background())
		util.ExitNicelyOnError(err, "Failed to shut down server")
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
	projects := []string{}
	for _, project := range user.Projects {
		projects = append(projects, project.GetName())
	}

	minderSrvKey := "Minder Server"
	projectKey := "Project Name"
	if len(projects) > 1 {
		projectKey += "s"
	}

	rows := []table.Row{
		{
			projectKey, strings.Join(projects, ", "),
		},
		{
			minderSrvKey, conn.Target(),
		},
	}

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
