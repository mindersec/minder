// Copyright 2023 Stacklok, Inc
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

package controlplane

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/auth"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context,
	req *pb.GetAuthorizationURLRequest) (*pb.GetAuthorizationURLResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// get provider info
	provider, err := getProviderFromRequestOrDefault(ctx, s.store, req, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	// Configure tracing
	// trace call to AuthCodeURL
	span := trace.SpanFromContext(ctx)
	span.SetName("server.GetAuthorizationURL")
	span.SetAttributes(attribute.Key("provider").String(provider.Name))
	defer span.End()

	// Create a new OAuth2 config for the given provider
	oauthConfig, err := auth.NewOAuthConfig(provider.Name, req.Cli)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce based state
	state, err := mcrypto.GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Format the port number
	port := sql.NullInt32{
		Int32: req.Port,
		Valid: true,
	}

	// Delete any existing session state for the project
	err = s.store.DeleteSessionStateByProjectID(ctx, db.DeleteSessionStateByProjectIDParams{
		Provider:  provider.Name,
		ProjectID: projectID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "error deleting session state: %s", err)
	}

	var owner sql.NullString
	if req.Owner == nil {
		owner = sql.NullString{Valid: false}
	} else {
		owner = sql.NullString{Valid: true, String: *req.Owner}
	}

	var redirectUrl sql.NullString
	if req.RedirectUrl == nil {
		redirectUrl = sql.NullString{Valid: false}
	} else {
		encryptedRedirectUrl, err := s.cryptoEngine.EncryptString(*req.RedirectUrl)
		if err != nil {
			return nil, status.Errorf(codes.Unknown, "error encrypting redirect URL: %s", err)
		}
		redirectUrl = sql.NullString{Valid: true, String: encryptedRedirectUrl}
	}

	// Insert the new session state into the database along with the user's project ID
	// retrieved from the JWT token
	_, err = s.store.CreateSessionState(ctx, db.CreateSessionStateParams{
		Provider:     provider.Name,
		ProjectID:    projectID,
		Port:         port,
		SessionState: state,
		OwnerFilter:  owner,
		RedirectUrl:  redirectUrl,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error inserting session state: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	// Return the authorization URL and state
	return &pb.GetAuthorizationURLResponse{
		Url: oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline),
	}, nil
}

// HandleProviderCallback handles the OAuth 2.0 authorization code callback from the enrolled
// provider. This function gathers the state from the database and compares it to the state
// passed in. If they match, the provider code is exchanged for a provider token.
// note: this is an HTTP only (not RPC) handler
func (s *Server) HandleProviderCallback() runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := r.Context()

		if err := s.processCallback(ctx, w, r, pathParams); err != nil {
			log.Printf("error handling provider callback: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) processCallback(ctx context.Context, w http.ResponseWriter, r *http.Request,
	pathParams map[string]string) error {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Configure tracing
	span := trace.SpanFromContext(ctx)
	span.SetName("server.HandleProviderCallback")
	span.SetAttributes(attribute.Key("code").String(code))
	defer span.End()

	provider := pathParams["provider"]
	if !app.IsProviderSupported(provider) {
		return fmt.Errorf("provider %s is not supported yet", provider)
	}

	// Check the nonce to make sure it's valid
	valid, err := mcrypto.IsNonceValid(state, s.cfg.Auth.NoncePeriod)
	if err != nil {
		return fmt.Errorf("error checking nonce: %w", err)
	}
	if !valid {
		return errors.New("invalid nonce")
	}

	// get projectID from session along with state nonce from the database
	stateData, err := s.store.GetProjectIDPortBySessionState(ctx, state)
	if err != nil {
		return fmt.Errorf("error getting project ID by session state: %w", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider
	logger.BusinessRecord(ctx).Project = stateData.ProjectID

	err = s.generateOAuthToken(ctx, provider, code, stateData)
	if err != nil {
		return err
	}

	if stateData.RedirectUrl.Valid {
		redirectUrl, err := s.cryptoEngine.DecryptString(stateData.RedirectUrl.String)
		if err != nil {
			return fmt.Errorf("error decrypting redirect URL: %w", err)
		}
		parsedURL, err := url.Parse(redirectUrl)
		if err != nil {
			return fmt.Errorf("error parsing redirect URL: %w", err)
		}
		http.Redirect(w, r, parsedURL.String(), http.StatusTemporaryRedirect)
		return nil
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write(auth.OAuthSuccessHtml)
	if err != nil {
		return fmt.Errorf("error writing OAuth success page: %w", err)
	}

	return nil
}

func (s *Server) generateOAuthToken(ctx context.Context, provider string, code string,
	stateData db.GetProjectIDPortBySessionStateRow) error {
	// generate a new OAuth2 config for the given provider
	oauthConfig, err := auth.NewOAuthConfig(provider, true)
	if err != nil {
		return fmt.Errorf("error creating OAuth config: %w", err)
	}
	if oauthConfig == nil {
		return errors.New("oauth2.Config is nil")
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("error exchanging code for token: %w", err)
	}

	ftoken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: "",
	}

	// Convert token to JSON
	jsonData, err := json.Marshal(ftoken)
	if err != nil {
		return fmt.Errorf("error marshaling token: %w", err)
	}

	// encode token
	encryptedToken, err := s.cryptoEngine.EncryptOAuthToken(jsonData)
	if err != nil {
		return fmt.Errorf("error encoding token: %w", err)
	}

	encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	var owner sql.NullString
	if stateData.OwnerFilter.Valid {
		owner = sql.NullString{Valid: true, String: stateData.OwnerFilter.String}
	} else {
		owner = sql.NullString{Valid: false}
	}

	_, err = s.store.UpsertAccessToken(ctx, db.UpsertAccessTokenParams{
		ProjectID:      stateData.ProjectID,
		Provider:       provider,
		EncryptedToken: encodedToken,
		OwnerFilter:    owner,
	})
	if err != nil {
		return fmt.Errorf("error inserting access token: %w", err)
	}
	return nil
}

// getProviderAccessToken returns the access token for providers
func (s *Server) getProviderAccessToken(ctx context.Context, provider string,
	projectID uuid.UUID) (oauth2.Token, string, error) {

	encToken, err := s.store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: provider, ProjectID: projectID})
	if err != nil {
		return oauth2.Token{}, "", err
	}

	decryptedToken, err := s.cryptoEngine.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return oauth2.Token{}, "", err
	}

	// base64 decode the token
	decryptedToken.Expiry = encToken.ExpirationTime
	return decryptedToken, encToken.OwnerFilter.String, nil
}

// StoreProviderToken stores the provider token for a project
func (s *Server) StoreProviderToken(ctx context.Context,
	in *pb.StoreProviderTokenRequest) (*pb.StoreProviderTokenResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	// validate token
	err = auth.ValidateProviderToken(ctx, provider.Name, in.AccessToken)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid token provided")
	}

	ftoken := &oauth2.Token{
		AccessToken:  in.AccessToken,
		RefreshToken: "",
	}

	// Convert token to JSON
	jsonData, err := json.Marshal(ftoken)
	if err != nil {
		return nil, err
	}

	// encode token
	encryptedToken, err := s.cryptoEngine.EncryptOAuthToken(jsonData)
	if err != nil {
		return nil, err
	}
	encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	// additionally, add an owner
	var owner sql.NullString
	if in.Owner == nil {
		owner = sql.NullString{Valid: false}
	} else {
		owner = sql.NullString{String: *in.Owner, Valid: true}
	}

	_, err = s.store.UpsertAccessToken(ctx, db.UpsertAccessTokenParams{
		ProjectID:      projectID,
		Provider:       provider.Name,
		EncryptedToken: encodedToken,
		OwnerFilter:    owner,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing access token: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	return &pb.StoreProviderTokenResponse{}, nil
}

// VerifyProviderTokenFrom verifies the provider token since a timestamp
func (s *Server) VerifyProviderTokenFrom(ctx context.Context,
	in *pb.VerifyProviderTokenFromRequest) (*pb.VerifyProviderTokenFromResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	// check if a token has been created since timestamp
	_, err = s.store.GetAccessTokenSinceDate(ctx,
		db.GetAccessTokenSinceDateParams{Provider: provider.Name, ProjectID: projectID, UpdatedAt: in.Timestamp.AsTime()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &pb.VerifyProviderTokenFromResponse{Status: "KO"}, nil
		}
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	return &pb.VerifyProviderTokenFromResponse{Status: "OK"}, nil
}
