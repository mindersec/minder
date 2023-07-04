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

	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/gh/queries"
	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context,
	req *pb.GetAuthorizationURLRequest) (*pb.GetAuthorizationURLResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, req.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// if we do not have a group, check if we can infer it
	if req.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		req.GroupId = group
	}

	// Configure tracing
	// trace call to AuthCodeURL
	span := trace.SpanFromContext(ctx)
	span.SetName("server.GetAuthorizationURL")
	span.SetAttributes(attribute.Key("provider").String(req.Provider))
	defer span.End()

	// Create a new OAuth2 config for the given provider
	oauthConfig, err := auth.NewOAuthConfig(req.Provider, req.Cli)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce based state
	state, err := mcrypto.GenerateNonce()
	if err != nil {
		return nil, err
	}

	groupID := sql.NullInt32{
		Int32: req.GroupId,
		Valid: true,
	}

	// Format the port number
	port := sql.NullInt32{
		Int32: req.Port,
		Valid: true,
	}

	// Delete any existing session state for the group
	err = s.store.DeleteSessionStateByGroupID(ctx, db.DeleteSessionStateByGroupIDParams{Provider: req.Provider, GrpID: groupID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "error deleting session state: %s", err)
	}

	// Insert the new session state into the database along with the user's group ID
	// retrieved from the JWT token
	_, err = s.store.CreateSessionState(ctx, db.CreateSessionStateParams{
		Provider:     req.Provider,
		GrpID:        groupID,
		Port:         port,
		SessionState: state,
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
	in *pb.ExchangeCodeForTokenCLIRequest) (*pb.ExchangeCodeForTokenCLIResponse, error) {

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

	// get groupID from session along with state nonce from the database
	groupId, err := s.store.GetGroupIDPortBySessionState(ctx, in.State)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error getting group ID by session state: %s", err)
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

	// Populate the database with the repositories using the GraphQL API
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: token.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	isOrg, err := client.CheckIfTokenIsForOrganization(ctx)
	if err != nil {
		return nil, err
	}

	repos, err := client.ListAllRepositories(ctx, isOrg)
	if err != nil {
		return nil, err
	}

	// // Insert the repositories into the database
	err = queries.SyncRepositoriesWithDB(ctx, s.store, repos, in.Provider, groupId.GrpID.Int32)
	if err != nil {
		return nil, err
	}

	// github does not provide refresh token or expiry, set a manual expiry time
	viper.SetDefault("github.access_token_expiry", 86400)
	expiryTime := time.Now().Add(time.Duration(viper.GetInt("github.access_token_expiry")) * time.Second)

	ftoken := &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: "",
		Expiry:       expiryTime,
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
	err = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: auth.Github, GroupID: groupId.GrpID.Int32})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error deleting access token: %s", err)
	}

	_, err = s.store.CreateAccessToken(ctx, db.CreateAccessTokenParams{
		GroupID:        groupId.GrpID.Int32,
		Provider:       auth.Github,
		EncryptedToken: encodedToken,
		ExpirationTime: expiryTime,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error inserting access token: %s", err)
	}

	return &pb.ExchangeCodeForTokenCLIResponse{
		Html: "The oauth flow has been completed successfully. You can now close this window.",
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

func decryptToken(encToken string) (oauth2.Token, error) {
	var decryptedToken oauth2.Token

	// base64 decode the token
	decodeToken, err := base64.StdEncoding.DecodeString(encToken)
	if err != nil {
		return decryptedToken, err
	}

	// decrypt the token
	token, err := mcrypto.DecryptBytes(viper.GetString("auth.token_key"), decodeToken)
	if err != nil {
		return decryptedToken, err
	}

	// serialise token *oauth.Token
	err = json.Unmarshal(token, &decryptedToken)
	if err != nil {
		return decryptedToken, err
	}
	return decryptedToken, nil
}

// GetProviderAccessToken returns the access token for providers
func GetProviderAccessToken(ctx context.Context, store db.Store, provider string, groupId int32) (oauth2.Token, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, groupId) {
		return oauth2.Token{}, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	encToken, err := store.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: provider, GroupID: groupId})
	if err != nil {
		return oauth2.Token{}, err
	}

	decryptedToken, err := decryptToken(encToken.EncryptedToken)
	if err != nil {
		return oauth2.Token{}, err
	}

	// base64 decode the token
	decryptedToken.Expiry = encToken.ExpirationTime
	return decryptedToken, nil
}

// RevokeOauthTokens revokes the all oauth tokens for a provider
func (s *Server) RevokeOauthTokens(ctx context.Context, in *pb.RevokeOauthTokensRequest) (*pb.RevokeOauthTokensResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// need to read all tokens from the provider and revoke them
	tokens, err := s.store.GetAccessTokenByProvider(ctx, auth.Github)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting access tokens: %v", err)
	}

	revoked_tokens := 0
	for _, token := range tokens {
		objToken, err := decryptToken(token.EncryptedToken)
		if err != nil {
			// just log and continue
			log.Error().Msgf("error decrypting token: %v", err)
		} else {
			// remove token from db
			_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: auth.Github, GroupID: token.GroupID})

			// remove from provider
			err := auth.DeleteAccessToken(ctx, token.Provider, objToken.AccessToken)

			if err != nil {
				log.Error().Msgf("Error deleting access token: %v", err)
			}
			revoked_tokens++
		}
	}
	return &pb.RevokeOauthTokensResponse{RevokedTokens: int32(revoked_tokens)}, nil
}

// RevokeOauthGroupToken revokes the oauth token for a group
func (s *Server) RevokeOauthGroupToken(ctx context.Context,
	in *pb.RevokeOauthGroupTokenRequest) (*pb.RevokeOauthGroupTokenResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// need to read the token for the provider and group
	token, err := s.store.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: auth.Github, GroupID: in.GroupId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}

	objToken, err := decryptToken(token.EncryptedToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error decrypting token: %v", err)
	}
	// remove token from db
	_ = s.store.DeleteAccessToken(ctx, db.DeleteAccessTokenParams{Provider: auth.Github, GroupID: token.GroupID})

	// remove from provider
	err = auth.DeleteAccessToken(ctx, token.Provider, objToken.AccessToken)

	if err != nil {
		log.Error().Msgf("Error deleting access token: %v", err)
	}
	return &pb.RevokeOauthGroupTokenResponse{}, nil
}

// StoreProviderToken stores the provider token for a group
func (s *Server) StoreProviderToken(ctx context.Context,
	in *pb.StoreProviderTokenRequest) (*pb.StoreProviderTokenResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// validate token
	err := auth.ValidateProviderToken(ctx, in.Provider, in.AccessToken)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid token provided")
	}

	// github does not provide refresh token or expiry, set a manual expiry time
	viper.SetDefault("github.access_token_expiry", 86400)
	expiryTime := time.Now().Add(time.Duration(viper.GetInt("github.access_token_expiry")) * time.Second)

	ftoken := &oauth2.Token{
		AccessToken:  in.AccessToken,
		RefreshToken: "",
		Expiry:       expiryTime,
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

	_, err = s.store.CreateAccessToken(ctx, db.CreateAccessTokenParams{GroupID: in.GroupId, Provider: in.Provider,
		EncryptedToken: encodedToken, ExpirationTime: expiryTime})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "error storing access token: %v", err)
	}
	return &pb.StoreProviderTokenResponse{}, nil
}

// VerifyProviderTokenFrom verifies the provider token since a timestamp
func (s *Server) VerifyProviderTokenFrom(ctx context.Context,
	in *pb.VerifyProviderTokenFromRequest) (*pb.VerifyProviderTokenFromResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// check if a token has been created since timestamp
	_, err := s.store.GetAccessTokenSinceDate(ctx,
		db.GetAccessTokenSinceDateParams{Provider: in.Provider, GroupID: in.GroupId, CreatedAt: in.Timestamp.AsTime()})
	if err != nil {
		if err == sql.ErrNoRows {
			return &pb.VerifyProviderTokenFromResponse{Status: "KO"}, nil
		}
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}
	return &pb.VerifyProviderTokenFromResponse{Status: "OK"}, nil
}
