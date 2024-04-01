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
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
	mcrypto "github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context,
	req *pb.GetAuthorizationURLRequest) (*pb.GetAuthorizationURLResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	var provider string
	if req.GetContext().GetProvider() == "" {
		provider = defaultProvider
	} else {
		provider = req.GetContext().GetProvider()
	}

	// get provider info
	providerDef, err := providers.GetProviderClassDefinition(provider)
	if err != nil {
		return nil, providerError(err)
	}

	if !slices.Contains(providerDef.AuthorizationFlows, db.AuthorizationFlowOauth2AuthorizationCodeFlow) &&
		!slices.Contains(providerDef.AuthorizationFlows, db.AuthorizationFlowGithubAppFlow) {
		return nil, util.UserVisibleError(codes.InvalidArgument,
			"provider does not support authorization code flow")
	}

	// Configure tracing
	// trace call to AuthCodeURL
	span := trace.SpanFromContext(ctx)
	span.SetName("server.GetAuthorizationURL")
	span.SetAttributes(attribute.Key("provider").String(provider))
	defer span.End()

	user, _ := auth.GetUserClaimFromContext[string](ctx, "gh_id")
	// If the user's token doesn't have gh_id set yet, we'll pass it through for now.
	s.mt.AddTokenOpCount(ctx, "issued", user != "")

	// Generate a random nonce based state
	state, err := mcrypto.GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Delete any existing session state for the project
	err = s.store.DeleteSessionStateByProjectID(ctx, db.DeleteSessionStateByProjectIDParams{
		Provider:  provider,
		ProjectID: projectID})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "error deleting session state: %s", err)
	}

	owner := sql.NullString{
		Valid:  req.GetOwner() != "",
		String: req.GetOwner(),
	}

	var redirectUrl sql.NullString
	// Empty redirect URL means null string (default condition)
	if req.GetRedirectUrl() != "" {
		encryptedRedirectUrl, err := s.cryptoEngine.EncryptString(*req.RedirectUrl)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error encrypting redirect URL: %s", err)
		}
		redirectUrl = sql.NullString{Valid: true, String: encryptedRedirectUrl}
	}

	// Insert the new session state into the database along with the user's project ID
	// retrieved from the JWT token
	_, err = s.store.CreateSessionState(ctx, db.CreateSessionStateParams{
		Provider:     provider,
		ProjectID:    projectID,
		RemoteUser:   sql.NullString{Valid: user != "", String: user},
		SessionState: state,
		OwnerFilter:  owner,
		RedirectUrl:  redirectUrl,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "error inserting session state: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider
	logger.BusinessRecord(ctx).Project = projectID

	// Create a new OAuth2 config for the given provider
	oauthConfig, err := s.providerAuthFactory(provider, req.Cli)
	if err != nil {
		return nil, err
	}

	var authorizationURL string
	if slices.Contains(providerDef.AuthorizationFlows, db.AuthorizationFlowGithubAppFlow) {
		gitHubAppConfig := s.cfg.Provider.GitHubApp
		if gitHubAppConfig == nil {
			return nil, status.Errorf(codes.Internal, "error getting GitHub App config: %s", err)
		}
		authorizationURL = fmt.Sprintf("https://github.com/apps/%v/installations/new?state=%v", gitHubAppConfig.AppName, state)
	} else if slices.Contains(providerDef.AuthorizationFlows, db.AuthorizationFlowOauth2AuthorizationCodeFlow) {
		authorizationURL = oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	}

	// Return the authorization URL and state
	return &pb.GetAuthorizationURLResponse{
		Url:   authorizationURL,
		State: state,
	}, nil
}

// HandleOAuthCallback handles the OAuth 2.0 authorization code callback from the enrolled
// provider. This function gathers the state from the database and compares it to the state
// passed in. If they match, the provider code is exchanged for a provider token.
// note: this is an HTTP only (not RPC) handler
func (s *Server) HandleOAuthCallback() runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := r.Context()

		if err := s.processOAuthCallback(ctx, w, r, pathParams); err != nil {
			if httpErr, ok := err.(*httpResponseError); ok {
				httpErr.WriteError(w)
				return
			}
			log.Printf("error handling provider callback: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// HandleGitHubAppCallback handles the authorization callback from the GitHub App. This function validates the
// GitHub user has access to the installation. It also gathers the state from the database and compares it to
// the state passed in, if present. If they match a new GitHub App provider is created with the installation ID.
// note: this is an HTTP only (not RPC) handler
func (s *Server) HandleGitHubAppCallback() runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := r.Context()

		if err := s.processAppCallback(ctx, w, r, pathParams); err != nil {
			var httpErr *httpResponseError
			if errors.As(err, &httpErr) {
				httpErr.WriteError(w)
				return
			}
			log.Printf("error handling provider callback: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) processOAuthCallback(ctx context.Context, w http.ResponseWriter, r *http.Request,
	pathParams map[string]string) error {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Configure tracing
	span := trace.SpanFromContext(ctx)
	span.SetName("server.HandleOAuthCallback")
	span.SetAttributes(attribute.Key("code").String(code))
	defer span.End()

	provider := pathParams["provider"]

	// Check the nonce to make sure it's valid
	valid, err := mcrypto.IsNonceValid(state, s.cfg.Auth.NoncePeriod)
	if err != nil {
		return fmt.Errorf("error checking nonce: %w", err)
	}
	if !valid {
		return errors.New("invalid nonce")
	}

	// get projectID from session along with state nonce from the database
	stateData, err := s.store.GetProjectIDBySessionState(ctx, state)
	if err != nil {
		return fmt.Errorf("error getting project ID by session state: %w", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider
	logger.BusinessRecord(ctx).Project = stateData.ProjectID

	token, err := s.exchangeCodeForToken(ctx, provider, code)
	if err != nil {
		return fmt.Errorf("error exchanging code for token: %w", err)
	}

	_, err = s.providers.CreateGitHubOAuthProvider(ctx, provider, db.ProviderClassGithub, *token, stateData, state)
	if err != nil {
		if errors.Is(err, providers.ErrInvalidTokenIdentity) {
			return newHttpError(http.StatusForbidden, "User token mismatch").SetContents(
				"The provided login token was associated with a different GitHub user.")
		}
		return fmt.Errorf("error creating provider: %w", err)
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

func (s *Server) processAppCallback(ctx context.Context, w http.ResponseWriter, r *http.Request,
	pathParams map[string]string) error {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Configure tracing
	span := trace.SpanFromContext(ctx)
	span.SetName("server.HandleGitHubAppCallback")
	span.SetAttributes(attribute.Key("code").String(code))
	defer span.End()

	provider := pathParams["provider"]

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider

	token, err := s.exchangeCodeForToken(ctx, provider, code)
	if err != nil {
		return err
	}

	installationID, err := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse installation ID to integer: %v", err)
	}

	err = s.providers.ValidateGitHubInstallationId(ctx, token, installationID)
	if err != nil {
		return newHttpError(http.StatusForbidden, "User installation ID mismatch").SetContents(
			"The GitHub user does not have access to the requested installation.")
	}

	if state != "" {
		stateData, err := s.getValidSessionState(ctx, state)
		if err != nil {
			return fmt.Errorf("error validating session state: %w", err)
		}

		logger.BusinessRecord(ctx).Project = stateData.ProjectID

		_, err = s.providers.CreateGitHubAppProvider(ctx, *token, stateData, installationID, state)
		if err != nil {
			if errors.Is(err, providers.ErrInvalidTokenIdentity) {
				return newHttpError(http.StatusForbidden, "User token mismatch").SetContents(
					"The provided login token was associated with a different GitHub user.")
			}
			return fmt.Errorf("error creating GitHub App provider: %w", err)
		}

		// If we have a redirect URL, redirect the user, otherwise show a success page
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
	} else {
		_, err := s.providers.CreateUnclaimedGitHubAppInstallation(ctx, token, installationID)
		if err != nil {
			return fmt.Errorf("error saving installation ID: %w", err)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write(auth.OAuthSuccessHtml)
	if err != nil {
		return fmt.Errorf("error writing OAuth success page: %w", err)
	}

	return nil
}

func (s *Server) getValidSessionState(ctx context.Context, state string) (db.GetProjectIDBySessionStateRow, error) {
	// Check the nonce to make sure it's valid
	valid, err := mcrypto.IsNonceValid(state, s.cfg.Auth.NoncePeriod)
	if err != nil {
		return db.GetProjectIDBySessionStateRow{}, fmt.Errorf("error checking nonce: %w", err)
	}
	if !valid {
		return db.GetProjectIDBySessionStateRow{}, errors.New("invalid nonce")
	}

	// get projectID from session along with state nonce from the database
	stateData, err := s.store.GetProjectIDBySessionState(ctx, state)
	if err != nil {
		return db.GetProjectIDBySessionStateRow{}, fmt.Errorf("error getting project ID by session state: %w", err)
	}
	return stateData, nil
}

func (s *Server) exchangeCodeForToken(ctx context.Context, providerName string, code string) (*oauth2.Token, error) {
	// generate a new OAuth2 config for the given provider
	oauthConfig, err := s.providerAuthFactory(providerName, true)
	if err != nil {
		return nil, fmt.Errorf("error creating OAuth config: %w", err)
	}
	if oauthConfig == nil {
		return nil, errors.New("oauth2.Config is nil")
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("error exchanging code for token: %w", err)
	}
	return token, nil
}

// StoreProviderToken stores the provider token for a project
func (s *Server) StoreProviderToken(ctx context.Context,
	in *pb.StoreProviderTokenRequest) (*pb.StoreProviderTokenResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	if providerName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "provider name is required")
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		// We don't check parent projects here because subprojects should not update the credentials of their parent
		Projects: []uuid.UUID{projectID},
		Name:     providerName,
	})
	if err != nil {
		return nil, providerError(err)
	}

	if !slices.Contains(provider.AuthFlows, db.AuthorizationFlowUserInput) {
		return nil, util.UserVisibleError(codes.InvalidArgument,
			"provider does not support token enrollment")
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
// Deprecated: Use VerifyProviderCredential instead
func (s *Server) VerifyProviderTokenFrom(ctx context.Context,
	in *pb.VerifyProviderTokenFromRequest) (*pb.VerifyProviderTokenFromResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	if providerName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "provider name is required")
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		// We don't check parent projects here because subprojects should not check the credentials of their parent
		Projects: []uuid.UUID{projectID},
		Name:     providerName,
	})
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

// VerifyProviderCredential verifies the provider credential has been created for the matching enrollment nonce
func (s *Server) VerifyProviderCredential(ctx context.Context,
	in *pb.VerifyProviderCredentialRequest) (*pb.VerifyProviderCredentialResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	enrollmentNonce := in.EnrollmentNonce

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	// check if a token has been created matching the nonce
	accessToken, err := s.store.GetAccessTokenByEnrollmentNonce(ctx, db.GetAccessTokenByEnrollmentNonceParams{
		ProjectID:       projectID,
		EnrollmentNonce: sql.NullString{Valid: true, String: enrollmentNonce},
	})

	if err == nil {
		return &pb.VerifyProviderCredentialResponse{
			Created:      true,
			ProviderName: accessToken.Provider,
		}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "error getting access token: %v", err)
	}

	// if there's no token, check for an installation ID
	installation, err := s.store.GetInstallationIDByEnrollmentNonce(ctx,
		db.GetInstallationIDByEnrollmentNonceParams{
			ProjectID: uuid.NullUUID{
				Valid: true,
				UUID:  projectID,
			},
			EnrollmentNonce: sql.NullString{Valid: true, String: enrollmentNonce},
		},
	)

	if err == nil {
		provider, err := s.store.GetProviderByID(ctx, installation.ProviderID.UUID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting provider: %v", err)
		}
		return &pb.VerifyProviderCredentialResponse{
			Created:      true,
			ProviderName: provider.Name,
		}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "error getting installation ID: %v", err)
	}

	return &pb.VerifyProviderCredentialResponse{Created: false}, nil
}

type httpResponseError struct {
	statusCode   int
	short        string
	pageContents string
}

func newHttpError(statusCode int, short string) *httpResponseError {
	return &httpResponseError{
		statusCode:   statusCode,
		short:        short,
		pageContents: "An unknown error occurred",
	}
}

func (e *httpResponseError) SetContents(contents string, args ...any) *httpResponseError {
	e.pageContents = fmt.Sprintf(contents, args...)
	return e
}

// Error implements error
func (e *httpResponseError) Error() string {
	return fmt.Sprintf("HTTP error: %d %s", e.statusCode, e.short)
}

func (e *httpResponseError) WriteError(w http.ResponseWriter) {
	http.Error(w, e.pageContents, e.statusCode)
}
