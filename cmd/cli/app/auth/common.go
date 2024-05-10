// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/cmd/cli/app"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	"github.com/stacklok/minder/internal/util/rand"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func userRegistered(ctx context.Context, client minderv1.UserServiceClient) (bool, *minderv1.GetUserResponse, error) {
	res, err := client.GetUser(ctx, &minderv1.GetUserRequest{})
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

func renderNewUser(conn string, newUser *minderv1.CreateUserResponse) {
	t := table.New(table.Simple, layouts.KeyValue, nil)
	t.AddRow("Subject", newUser.GetIdentitySubject())
	t.AddRow("Project ID", newUser.ProjectId)
	t.AddRow("Project Name", newUser.ProjectName)
	t.AddRow("Minder Server", conn)
	t.Render()
}

func renderUserInfo(conn string, user *minderv1.GetUserResponse) {
	t := table.New(table.Simple, layouts.KeyValue, nil)
	t.AddRow("Minder Server", conn)
	t.AddRow("Subject", user.GetUser().GetIdentitySubject())
	for _, project := range getProjectTableRows(user.Projects) {
		t.AddRow(project...)
	}
	t.Render()
}

func renderUserInfoWhoami(conn string, outWriter io.Writer, format string, user *minderv1.GetUserResponse) {
	switch format {
	case app.Table:
		fmt.Fprintln(outWriter, cli.Header.Render("Here are your details:"))
		t := table.New(table.Simple, layouts.KeyValue, nil)
		t.AddRow("Subject", user.GetUser().GetIdentitySubject())
		t.AddRow("Created At", user.GetUser().GetCreatedAt().AsTime().String())
		t.AddRow("Updated At", user.GetUser().GetUpdatedAt().AsTime().String())
		t.AddRow("Minder Server", conn)
		for _, project := range getProjectTableRows(user.Projects) {
			t.AddRow(project...)
		}
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to JSON: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	case app.YAML:
		out, err := util.GetYamlFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to YAML: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	}
}

func getProjectTableRows(projects []*minderv1.Project) [][]string {
	var rows [][]string
	projectKey := "Project"
	for idx, project := range projects {
		if len(projects) > 1 {
			projectKey = fmt.Sprintf("Project #%d", idx+1)
		}
		projectVal := fmt.Sprintf("%s / %s", project.GetName(), project.GetProjectId())
		rows = append(rows, []string{projectKey, projectVal})
	}
	return rows
}

type loginError struct {
	ErrorType   string
	Description string
}

func (e loginError) Error() string {
	return fmt.Sprintf("Error: %s\nDescription: %s\n", e.ErrorType, e.Description)
}

func (e loginError) isAccessDenied() bool {
	return e.ErrorType == "access_denied"
}

func writeError(w http.ResponseWriter, loginerr loginError) (string, error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	htmlPage := genericAuthFailure
	msg := "Access Denied."

	if loginerr.isAccessDenied() {
		htmlPage = accessDeniedHtml
		msg = "Access Denied. Please accept the terms and conditions"
	}

	_, err := w.Write(htmlPage)
	if err != nil {
		return msg, err
	}
	return "", nil
}

// login is a helper function to handle the login process
// and return the access token
func login(
	ctx context.Context,
	cmd *cobra.Command,
	cfg *clientconfig.Config,
	extraScopes []string,
	skipBroswer bool,
) (*oidc.Tokens[*oidc.IDTokenClaims], error) {
	issuerUrlStr := cfg.Identity.CLI.IssuerUrl
	clientID := cfg.Identity.CLI.ClientId

	parsedURL, err := url.Parse(issuerUrlStr)
	if err != nil {
		return nil, cli.MessageAndError("Error parsing issuer URL", err)
	}

	issuerUrl := parsedURL.JoinPath("realms/stacklok")
	scopes := []string{"openid", "minder-audience"}

	if len(extraScopes) > 0 {
		scopes = append(scopes, extraScopes...)
	}

	callbackPath := "/auth/callback"

	errChan := make(chan loginError)

	errorHandler := func(w http.ResponseWriter, _ *http.Request, errorType string, errorDesc string, _ string) {
		loginerr := loginError{
			ErrorType:   errorType,
			Description: errorDesc,
		}

		msg, writeErr := writeError(w, loginerr)
		if writeErr != nil {
			// if we cannot display the access denied page, just print an error message
			cmd.Println(msg)
		}
		errChan <- loginerr
	}

	// create encrypted cookie handler to mitigate CSRF attacks
	hashKey := securecookie.GenerateRandomKey(32)
	encryptKey := securecookie.GenerateRandomKey(32)
	cookieHandler := httphelper.NewCookieHandler(hashKey, encryptKey, httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode))
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithPKCE(cookieHandler),
		rp.WithErrorHandler(errorHandler),
	}

	// Get random port
	port, err := rand.GetRandomPort()
	if err != nil {
		return nil, cli.MessageAndError("Error getting random port", err)
	}

	parsedURL, err = url.Parse(fmt.Sprintf("http://localhost:%v", port))
	if err != nil {
		return nil, cli.MessageAndError("Error parsing callback URL", err)
	}
	redirectURI := parsedURL.JoinPath(callbackPath)

	provider, err := rp.NewRelyingPartyOIDC(issuerUrl.String(), clientID, "", redirectURI.String(), scopes, options...)
	if err != nil {
		return nil, cli.MessageAndError("Error creating relying party", err)
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
		err := server.ListenAndServe()
		// ignore error if it's just a graceful shutdown
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			cmd.Printf("Error starting server: %v\n", err)
		}
	}()

	defer server.Shutdown(ctx)

	// get the OAuth authorization URL
	loginUrl := fmt.Sprintf("http://localhost:%v/login", port)

	if !skipBroswer {
		// Redirect user to provider to log in
		cmd.Printf("Your browser will now be opened to: %s\n", loginUrl)

		// open user's browser to login page
		if err := browser.OpenURL(loginUrl); err != nil {
			cmd.Printf("You may login by pasting this URL into your browser: %s\n", loginUrl)
		}
	} else {
		cmd.Printf("Skipping browser login. You may login by pasting this URL into your browser: %s\n", loginUrl)
	}

	cmd.Println("Please follow the instructions on the page to log in.")

	cmd.Println("Waiting for token...")

	// wait for the token to be received
	var token *oidc.Tokens[*oidc.IDTokenClaims]
	var loginErr error

	select {
	case token = <-tokenChan:
		break
	case loginErr = <-errChan:
		break
	}

	return token, loginErr
}
