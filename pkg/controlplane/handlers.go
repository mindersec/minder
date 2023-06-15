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

// Package controlplane contains the gRPC server implementation for the control plane
package controlplane

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/spf13/viper"
)

// Google OAuth2 provider
const Google = "google"

// Github OAuth2 provider
const Github = "github"

// PaginationLimit is the maximum number of items that can be returned in a single page
const PaginationLimit = 10

// CheckHealth is a simple health check for monitoring
// The lintcheck is disabled because the unused-receiver is required by
// the implementation. UnimplementedHealthServiceServer is initialized
// within the Server struct
//
//revive:disable:unused-receiver
func (s *Server) CheckHealth(_ context.Context, _ *pb.CheckHealthRequest) (*pb.CheckHealthResponse, error) {
	return &pb.CheckHealthResponse{Status: "OK"}, nil
}

//revive:enable:unused-receiver

// newOAuthConfig creates a new OAuth2 config for the given provider
// and whether the client is a CLI or web client
func newOAuthConfig(provider string, cli bool) (*oauth2.Config, error) {
	redirectURL := func(provider string, cli bool) string {
		if cli {
			return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/cli", provider)
		}
		return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/web", provider)
	}

	scopes := func(provider string) []string {
		if provider == Google {
			return []string{"profile", "email"}
		}
		return []string{"user:email", "repo"}
	}

	endpoint := func(provider string) oauth2.Endpoint {
		if provider == Google {
			return google.Endpoint
		}
		return github.Endpoint
	}

	if provider != Google && provider != Github {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}

	return &oauth2.Config{
		ClientID:     viper.GetString(fmt.Sprintf("%s.client_id", provider)),
		ClientSecret: viper.GetString(fmt.Sprintf("%s.client_secret", provider)),
		RedirectURL:  redirectURL(provider, cli),
		Scopes:       scopes(provider),
		Endpoint:     endpoint(provider),
	}, nil
}

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context,
	req *pb.GetAuthorizationURLRequest) (*pb.GetAuthorizationURLResponse, error) {
	// Configure tracing
	// trace call to AuthCodeURL
	span := trace.SpanFromContext(ctx)
	span.SetName("server.GetAuthorizationURL")
	span.SetAttributes(attribute.Key("provider").String(req.Provider))
	defer span.End()

	// Get the user claims from the JWT token
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)

	// Create a new OAuth2 config for the given provider
	oauthConfig, err := newOAuthConfig(req.Provider, req.Cli)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce based state
	state, err := mcrypto.GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Delete any existing session state for the user
	groupID := sql.NullInt32{
		Int32: claims.GroupId,
		Valid: true,
	}

	// Format the port number
	port := sql.NullInt32{
		Int32: req.Port,
		Valid: true,
	}

	// Delete any existing session state for the group
	err = s.store.DeleteSessionStateByGroupID(ctx, groupID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error deleting session state: %w", err)
	}

	// Insert the new session state into the database along with the user's group ID
	// retrieved from the JWT token
	_, err = s.store.CreateSessionState(ctx, db.CreateSessionStateParams{
		GrpID:        groupID,
		Port:         port,
		SessionState: state,
	})
	if err != nil {
		return nil, fmt.Errorf("error inserting session state: %w", err)
	}

	// Return the authorization URL and state
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	response := &pb.GetAuthorizationURLResponse{
		Url: url,
	}
	return response, nil
}

// ExchangeCodeForTokenCLI exchanges an OAuth2 code for a token
// This function gathers the state from the database and compares it to the state
// passed in. If they match, the code is exchanged for a token.
// This function is used by the CLI client.
func (s *Server) ExchangeCodeForTokenCLI(ctx context.Context,
	in *pb.ExchangeCodeForTokenCLIRequest) (*pb.ExchangeCodeForTokenCLIResponse, error) {

	// Configure tracing
	span := trace.SpanFromContext(ctx)
	span.SetName("server.ExchangeCodeForTokenCLI")
	span.SetAttributes(attribute.Key("code").String(in.Code))
	defer span.End()

	// Check the nonce to make sure it's valid
	valid, err := mcrypto.IsNonceValid(in.State)

	if err != nil {
		return nil, fmt.Errorf("error checking nonce: %w", err)
	}

	if !valid {
		return nil, fmt.Errorf("invalid nonce")
	}

	// get groupID from session along with state nonce from the database
	groupID, err := s.store.GetGroupIDPortBySessionState(ctx, in.State)
	if err != nil {
		return nil, fmt.Errorf("error getting groupID by session state: %w", err)
	}

	// generate a new OAuth2 config for the given provider
	oauthConfig, err := newOAuthConfig(in.Provider, true)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, fmt.Errorf("oauth2.Config is nil")
	}

	token, err := oauthConfig.Exchange(ctx, in.Code)
	if err != nil {
		return nil, err
	}

	var status string
	if token.Valid() {
		status = "success"
	} else {
		status = "failure"
	}

	ftoken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}

	// Convert token to JSON
	jsonData, err := json.Marshal(ftoken)
	if err != nil {
		return nil, err
	}

	// encode token
	encryptedToken, err := mcrypto.EncryptBytes(viper.GetString("auth.token_key"), jsonData)
	if err != nil {
		return nil, err
	}

	encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	// delete token if it exists
	err = s.store.DeleteAccessToken(ctx, groupID.GrpID.Int32)
	if err != nil {
		return nil, fmt.Errorf("error deleting access token: %w", err)
	}

	_, err = s.store.CreateAccessToken(ctx, db.CreateAccessTokenParams{
		GroupID:        groupID.GrpID.Int32,
		EncryptedToken: encodedToken,
	})
	if err != nil {
		return nil, fmt.Errorf("error inserting access token: %w", err)
	}

	cliAppURL := fmt.Sprintf("http://localhost:%d/shutdown", groupID.Port.Int32)

	// The following is tagged with //nosec as its a false positive. GOSEC alerts
	// for when URL is constructed from user or external input, but it's not.
	// the values for 'status' are set above in the server code. For someone
	// to exploit this, they would need to be able to modify the code of the
	// and recompile their own version of the application.
	resp, err := http.Post(cliAppURL, "application/json", bytes.NewBuffer([]byte(`{"status": "`+status+`"}`))) // #nosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send status to CLI application, status code: %d", resp.StatusCode)
	}

	return &pb.ExchangeCodeForTokenCLIResponse{
		Html: "You can now close this window.",
	}, nil
}

// ExchangeCodeForTokenWEB exchanges an OAuth2 code for a token and returns
// a JWT token as a session cookie. This handler is specific for web clients.
// The lint check for this function is disabled because it's a false positive.
// It will complain about am unsused receiver (s *Server), however this receiver
// will be used later when we implement the the database store.
//
//revive:disable:unused-receiver
func (s *Server) ExchangeCodeForTokenWEB(ctx context.Context,
	in *pb.ExchangeCodeForTokenWEBRequest) (*pb.ExchangeCodeForTokenWEBResponse, error) {
	oauthConfig, err := newOAuthConfig(in.Provider, false)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, fmt.Errorf("oauth2.Config is nil")
	}

	// get the token
	_, err = oauthConfig.Exchange(ctx, in.Code)
	if err != nil {
		return nil, err
	}

	// TODO: The below response needs to return as a session cookie
	return &pb.ExchangeCodeForTokenWEBResponse{
		AccessToken: "access_token",
	}, nil
}

//revive:enable:unused-receiver
