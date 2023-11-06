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

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context,
	req *pb.GetAuthorizationURLRequest) (*pb.GetAuthorizationURLResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, req)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	// Configure tracing
	// trace call to AuthCodeURL
	span := trace.SpanFromContext(ctx)
	span.SetName("server.GetAuthorizationURL")
	span.SetAttributes(attribute.Key("provider").String(req.Provider))
	defer span.End()

	// get provider info
	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      req.Provider,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

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

	// Delete any existing session state for the group
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

	// Insert the new session state into the database along with the user's group ID
	// retrieved from the JWT token
	_, err = s.store.CreateSessionState(ctx, db.CreateSessionStateParams{
		Provider:     provider.Name,
		ProjectID:    projectID,
		Port:         port,
		SessionState: state,
		OwnerFilter:  owner,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error inserting session state: %s", err)
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
	in *pb.ExchangeCodeForTokenCLIRequest) (*httpbody.HttpBody, error) {

	// Configure tracing
	span := trace.SpanFromContext(ctx)
	span.SetName("server.ExchangeCodeForTokenCLI")
	span.SetAttributes(attribute.Key("code").String(in.Code))
	defer span.End()

	// Check the nonce to make sure it's valid
	valid, err := mcrypto.IsNonceValid(in.State)

	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error checking nonce: %s", err)
	}

	if !valid {
		return nil, status.Error(codes.InvalidArgument, "invalid nonce")
	}

	// get projectID from session along with state nonce from the database
	stateData, err := s.store.GetProjectIDPortBySessionState(ctx, in.State)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error getting group ID by session state: %s", err)
	}

	// get provider
	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: stateData.ProjectID,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// generate a new OAuth2 config for the given provider
	oauthConfig, err := auth.NewOAuthConfig(in.Provider, true)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, status.Error(codes.Unknown, "oauth2.Config is nil")
	}

	token, err := oauthConfig.Exchange(ctx, in.Code)
	if err != nil {
		return nil, err
	}

	ftoken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
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

	// delete token if it exists
	err = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{
		Provider: provider.Name, ProjectID: stateData.ProjectID})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error deleting access token: %s", err)
	}

	var owner sql.NullString
	if stateData.OwnerFilter.Valid {
		owner = sql.NullString{Valid: true, String: stateData.OwnerFilter.String}
	} else {
		owner = sql.NullString{Valid: false}
	}
	_, err = s.store.CreateAccessToken(ctx, db.CreateAccessTokenParams{
		ProjectID:      stateData.ProjectID,
		Provider:       provider.Name,
		EncryptedToken: encodedToken,
		OwnerFilter:    owner,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error inserting access token: %s", err)
	}

	return &httpbody.HttpBody{
		ContentType: "text/html",
		Data:        auth.OAuthSuccessHtml,
	}, nil
}

// ExchangeCodeForTokenWEB exchanges an OAuth2 code for a token and returns
// a JWT token as a session cookie. This handler is specific for web clients.
// The lint check for this function is disabled because it's a false positive.
// It will complain about am unsused receiver (s *Server), however this receiver
// will be used later when we implement the database store.
//
//revive:disable:unused-receiver
func (s *Server) ExchangeCodeForTokenWEB(ctx context.Context,
	in *pb.ExchangeCodeForTokenWEBRequest) (*pb.ExchangeCodeForTokenWEBResponse, error) {
	oauthConfig, err := auth.NewOAuthConfig(in.Provider, false)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, status.Error(codes.Unknown, "oauth2.Config is nil")
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

// getProviderAccessToken returns the access token for providers
func (s *Server) getProviderAccessToken(ctx context.Context, provider string,
	projectID uuid.UUID, checkAuthz bool) (oauth2.Token, string, error) {
	// check if user is authorized
	if checkAuthz {
		if err := AuthorizedOnProject(ctx, projectID); err != nil {
			return oauth2.Token{}, "", err
		}
	}

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

// RevokeOauthTokens revokes the all oauth tokens for a provider
// This is in case of a security breach, where we need to revoke all tokens
func (s *Server) RevokeOauthTokens(ctx context.Context, _ *pb.RevokeOauthTokensRequest) (*pb.RevokeOauthTokensResponse, error) {
	providers, err := s.store.GlobalListProviders(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unable to list providers: %v", err)
	}

	revoked_tokens := 0

	for idx := range providers {
		provider := providers[idx]
		// need to read all tokens from the provider and revoke them
		tokens, err := s.store.GetAccessTokenByProvider(ctx, provider.Name)
		if errors.Is(err, sql.ErrNoRows) {
			zerolog.Ctx(ctx).Info().Str("provider", provider.Name).Msgf("no tokens found, skipping")
			continue
		} else if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting access tokens: %v", err)
		}

		for _, token := range tokens {
			objToken, err := s.cryptoEngine.DecryptOAuthToken(token.EncryptedToken)
			if err != nil {
				// just log and continue
				log.Error().Msgf("error decrypting token: %v", err)
			} else {
				// remove token from db
				_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: provider.Name, ProjectID: token.ProjectID})

				// remove from provider
				err := auth.DeleteAccessToken(ctx, provider.Name, objToken.AccessToken)

				if err != nil {
					log.Error().Msgf("Error deleting access token: %v", err)
				}
				revoked_tokens++
			}
		}
	}
	return &pb.RevokeOauthTokensResponse{RevokedTokens: int32(revoked_tokens)}, nil
}

// RevokeOauthProjectToken revokes the oauth token for a group
func (s *Server) RevokeOauthProjectToken(ctx context.Context,
	in *pb.RevokeOauthProjectTokenRequest) (*pb.RevokeOauthProjectTokenResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name: in.Provider, ProjectID: projectID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// need to read the token for the provider and group
	token, err := s.store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: provider.Name, ProjectID: projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}

	objToken, err := s.cryptoEngine.DecryptOAuthToken(token.EncryptedToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error decrypting token: %v", err)
	}
	// remove token from db
	_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: provider.Name, ProjectID: token.ProjectID})

	// remove from provider
	err = auth.DeleteAccessToken(ctx, provider.Name, objToken.AccessToken)

	if err != nil {
		log.Error().Msgf("Error deleting access token: %v", err)
	}
	return &pb.RevokeOauthProjectTokenResponse{}, nil
}

// StoreProviderToken stores the provider token for a group
func (s *Server) StoreProviderToken(ctx context.Context,
	in *pb.StoreProviderTokenRequest) (*pb.StoreProviderTokenResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name: in.Provider, ProjectID: projectID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// validate token
	err = auth.ValidateProviderToken(ctx, in.Provider, in.AccessToken)
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

	// additionally add owner
	var owner sql.NullString
	if in.Owner == nil {
		owner = sql.NullString{Valid: false}
	} else {
		owner = sql.NullString{String: *in.Owner, Valid: true}
	}

	_, err = s.store.CreateAccessToken(ctx, db.CreateAccessTokenParams{ProjectID: projectID, Provider: provider.Name,
		EncryptedToken: encodedToken, OwnerFilter: owner})

	if db.ErrIsUniqueViolation(err) {
		return nil, util.UserVisibleError(codes.AlreadyExists, "token already exists")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing access token: %v", err)
	}
	return &pb.StoreProviderTokenResponse{}, nil
}

// VerifyProviderTokenFrom verifies the provider token since a timestamp
func (s *Server) VerifyProviderTokenFrom(ctx context.Context,
	in *pb.VerifyProviderTokenFromRequest) (*pb.VerifyProviderTokenFromResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name: in.Provider, ProjectID: projectID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// check if a token has been created since timestamp
	_, err = s.store.GetAccessTokenSinceDate(ctx,
		db.GetAccessTokenSinceDateParams{Provider: provider.Name, ProjectID: projectID, CreatedAt: in.Timestamp.AsTime()})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &pb.VerifyProviderTokenFromResponse{Status: "KO"}, nil
		}
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}
	return &pb.VerifyProviderTokenFromResponse{Status: "OK"}, nil
}
