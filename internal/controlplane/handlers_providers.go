// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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
	"errors"

	"github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetProvider gets a given provider available in a specific project.
func (s *Server) GetProvider(ctx context.Context, req *minderv1.GetProviderRequest) (*minderv1.GetProviderResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	prov, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name: req.Name,
		// Note that this does not take the hierarchy into account in purpose.
		// We want to get this call to be explicit for the given project.
		Projects: []uuid.UUID{projectID},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting provider: %v", err)
	}

	var cfg *structpb.Struct

	if len(prov.Definition) > 0 {
		cfg = &structpb.Struct{}
		if err := protojson.Unmarshal(prov.Definition, cfg); err != nil {
			return nil, status.Errorf(codes.Internal, "error unmarshalling provider definition: %v", err)
		}
	}

	return &minderv1.GetProviderResponse{
		Provider: &minderv1.Provider{
			Name:       prov.Name,
			Project:    projectID.String(),
			Version:    prov.Version,
			Implements: protobufProviderImplementsFromDB(ctx, prov),
			AuthFlows:  protobufProviderAuthFlowFromDB(ctx, prov),
			Config:     cfg,
		},
	}, nil
}

// ListProviders lists the providers available in a specific project.
func (s *Server) ListProviders(ctx context.Context, req *minderv1.ListProvidersRequest) (*minderv1.ListProvidersResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	params := db.ListProvidersByProjectIDPaginatedParams{
		ProjectID: projectID,
	}

	if req.Cursor != "" {
		cursor, err := cursorutil.NewProviderCursor(req.Cursor)
		if err != nil {
			return nil, err
		}

		params.CreatedAt = sql.NullTime{
			Valid: true,
			Time:  cursor.CreatedAt,
		}
	}

	if req.Limit == 0 {
		params.Limit = 10
	} else {
		params.Limit = req.Limit
	}

	list, err := s.store.ListProvidersByProjectIDPaginated(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &minderv1.ListProvidersResponse{
				Providers: []*minderv1.Provider{},
			}, nil
		}
		return nil, err
	}

	zerolog.Ctx(ctx).Debug().Int("count", len(list)).Msg("providers")

	provs := make([]*minderv1.Provider, 0, len(list))
	for _, p := range list {
		var cfg *structpb.Struct

		if len(p.Definition) > 0 {
			cfg = &structpb.Struct{}
			if err := protojson.Unmarshal(p.Definition, cfg); err != nil {
				return nil, status.Errorf(codes.Internal, "error unmarshalling provider definition: %v", err)
			}
		}

		provs = append(provs, &minderv1.Provider{
			Name:       p.Name,
			Project:    projectID.String(),
			Version:    p.Version,
			Implements: protobufProviderImplementsFromDB(ctx, p),
			AuthFlows:  protobufProviderAuthFlowFromDB(ctx, p),
			Config:     cfg,
		})
	}

	cursor := ""
	if len(list) > 0 {
		c := cursorutil.ProviderCursor{
			CreatedAt: list[len(list)-1].CreatedAt,
		}
		cursor = c.String()
	}

	return &minderv1.ListProvidersResponse{
		Providers: provs,
		Cursor:    cursor,
	}, nil
}

func protobufProviderImplementsFromDB(ctx context.Context, p db.Provider) []minderv1.ProviderType {
	impls := make([]minderv1.ProviderType, 0, len(p.Implements))
	for _, i := range p.Implements {
		impl, ok := providers.DBToPBType(i)
		if !ok {
			zerolog.Ctx(ctx).Error().Str("type", string(i)).Str("id", p.ID.String()).Msg("unknown provider type")
			// we won't return an error here, we'll just skip the provider implementation listing
			continue
		}
		impls = append(impls, impl)
	}

	return impls
}

func protobufProviderAuthFlowFromDB(ctx context.Context, p db.Provider) []minderv1.AuthorizationFlow {
	flows := make([]minderv1.AuthorizationFlow, 0, len(p.AuthFlows))
	for _, a := range p.AuthFlows {
		flow, ok := providers.DBToPBAuthFlow(a)
		if !ok {
			zerolog.Ctx(ctx).Error().Str("flow", string(a)).Msg("unknown authorization flow")
			continue
		}
		flows = append(flows, flow)
	}

	return flows
}

func (s *Server) GetUnclaimedProviders(ctx context.Context, req *minderv1.GetUnclaimedProvidersRequest) (*minderv1.GetUnclaimedProvidersResponse, error) {
	userID, _ := auth.GetUserClaimFromContext[string](ctx, "gh_id")
	if userID == "" {
		return nil, util.UserVisibleError(codes.PermissionDenied, "cannot determine GitHub user identity")
	}

	ret := &minderv1.GetUnclaimedProvidersResponse{
		Providers: []*minderv1.ProviderParameter{},
	}

	// TODO: get app installs from GitHub
	// It would be nice to use this endpoint, but we'd need a GitHub App user access token:
	// https://docs.github.com/en/rest/apps/installations?apiVersion=2022-11-28#list-repositories-accessible-to-the-user-access-token
	var ghClient github.Client

	opts := github.ListOptions{PerPage: 100}
	candidates := []*github.Installation{}
	for {
		// We want to use ListUserInstallations here when we figure out the auth token.
		installs, resp, err := ghClient.Apps.ListInstallations(ctx, nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error listing installations: %v", err)
		}
		candidates = append(candidates, installs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	for _, i := range candidates {
		// Get candidate admins and join with the user's GitHub ID
		ret.Providers = append(ret.Providers, &minderv1.ProviderParameter{
			Parameters: &minderv1.ProviderParameter_GithubApp{
				GithubApp: &minderv1.GitHubAppParams{
					InstallationId: i.GetID(),
					Organization:   i.GetAccount().GetLogin(),
					OrganizationId: i.GetAccount().GetID(),
				},
			},
		})
	}

	return ret, nil
}
