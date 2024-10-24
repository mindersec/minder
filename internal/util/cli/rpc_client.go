// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cli contains utility for the cli
package cli

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/v3/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	mcrypto "github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli/useragent"
	"github.com/mindersec/minder/internal/util/rand"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

//go:embed html/login_success.html
var loginSuccessHtml []byte

//go:embed html/access_denied.html
var accessDeniedHtml []byte

//go:embed html/generic_failure.html
var genericAuthFailure []byte

func requestIDInterceptor(printer func(string, ...interface{})) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var header metadata.MD
		opts = append(opts, grpc.Header(&header))
		err := invoker(ctx, method, req, reply, cc, opts...)
		if len(header.Get("request-id")) != 0 {
			printer("Request ID: %s\n", header.Get("request-id")[0])
		}
		return err
	}
}

// GrpcForCommand is a helper for getting a testing connection from cobra flags
func GrpcForCommand(cmd *cobra.Command, v *viper.Viper) (*grpc.ClientConn, error) {
	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](v)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %w", err)
	}

	grpcHost := clientConfig.GRPCClientConfig.Host
	grpcPort := clientConfig.GRPCClientConfig.Port
	insecureDefault := grpcHost == "localhost" || grpcHost == "127.0.0.1" || grpcHost == "::1"
	allowInsecure := clientConfig.GRPCClientConfig.Insecure || insecureDefault

	issuerUrl := clientConfig.Identity.CLI.IssuerUrl
	clientId := clientConfig.Identity.CLI.ClientId

	opts := []grpc.DialOption{}
	opts = append(opts, grpc.WithUserAgent(useragent.GetUserAgent()))

	if viper.GetBool("verbose") {
		opts = append(opts, grpc.WithUnaryInterceptor(requestIDInterceptor(cmd.PrintErrf)))
	}

	return util.GetGrpcConnection(
		grpcHost,
		grpcPort,
		allowInsecure,
		issuerUrl,
		clientId,
		opts...,
	)
}

// EnsureCredentials is a PreRunE function to ensure that the user
// has valid credentials, opening a browser for login if needed.
func EnsureCredentials(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return MessageAndError("Unable to read config", err)
	}

	_, err = util.GetToken(clientConfig.Identity.CLI.IssuerUrl, clientConfig.Identity.CLI.ClientId)
	if err != nil { // or token is expired?
		tokenFile, err := LoginAndSaveCreds(ctx, cmd, clientConfig)
		if err != nil {
			return MessageAndError("Error fetching credentials from Minder", err)
		}
		cmd.Printf("Your access credentials have been saved to %s\n", tokenFile)
	}
	return nil
}

// LoginAndSaveCreds runs a login flow for the user, opening a browser if needed.
// If the credentials need to be refreshed, the new credentials will be saved for future use.
func LoginAndSaveCreds(ctx context.Context, cmd *cobra.Command, clientConfig *clientconfig.Config) (string, error) {
	skipBrowser := viper.GetBool("login.skip-browser")

	// wait for the token to be received
	var loginErr loginError
	token, err := Login(ctx, cmd, clientConfig, nil, skipBrowser)
	if errors.As(err, &loginErr) && loginErr.isAccessDenied() {
		return "", errors.New("Access denied. Please run the command again and accept the terms and conditions.")
	}
	if err != nil {
		return "", err
	}

	// save credentials
	filePath, err := util.SaveCredentials(util.OpenIdCredentials{
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpiresAt: token.Expiry,
	})
	if err != nil {
		cmd.PrintErrf("couldn't save credentials: %s\n", err)
		return "", err
	}

	return filePath, err
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

// Login is a helper function to handle the login process
// and return the access token
func Login(
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
		return nil, MessageAndError("Error parsing issuer URL", err)
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
		return nil, MessageAndError("Error getting random port", err)
	}

	parsedURL, err = url.Parse(fmt.Sprintf("http://localhost:%v", port))
	if err != nil {
		return nil, MessageAndError("Error parsing callback URL", err)
	}
	redirectURI := parsedURL.JoinPath(callbackPath)

	provider, err := rp.NewRelyingPartyOIDC(ctx, issuerUrl.String(), clientID, "", redirectURI.String(), scopes, options...)
	if err != nil {
		return nil, MessageAndError("Error creating relying party", err)
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
