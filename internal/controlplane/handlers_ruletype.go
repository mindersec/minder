// Copyright 2023 Stacklok, Inc
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
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/engcontext"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/ruletypes"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	errInvalidRuleType = errors.New("invalid rule type")
)

// ListRuleTypes is a method to list all rule types for a given context
func (s *Server) ListRuleTypes(
	ctx context.Context,
	_ *minderv1.ListRuleTypesRequest,
) (*minderv1.ListRuleTypesResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)

	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	lrt, err := s.store.ListRuleTypesByProject(ctx, entityCtx.Project.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule types: %s", err)
	}

	resp := &minderv1.ListRuleTypesResponse{}

	for idx := range lrt {
		rt := lrt[idx]
		rtpb, err := ruletypes.RuleTypePBFromDB(&rt)
		if err != nil {
			return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rt.Name, err)
		}

		resp.RuleTypes = append(resp.RuleTypes, rtpb)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = entityCtx.Project.ID

	return resp, nil
}

// GetRuleTypeByName is a method to get a rule type by name
func (s *Server) GetRuleTypeByName(
	ctx context.Context,
	in *minderv1.GetRuleTypeByNameRequest,
) (*minderv1.GetRuleTypeByNameResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)

	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	resp := &minderv1.GetRuleTypeByNameResponse{}

	rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		// TODO: Add option to fetch rule types from parent projects too
		Projects: []uuid.UUID{entityCtx.Project.ID},
		Name:     in.GetName(),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, util.UserVisibleError(codes.NotFound, "rule type %s not found", in.GetName())
	} else if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	rt, err := ruletypes.RuleTypePBFromDB(&rtdb)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rtdb.Name, err)
	}

	resp.RuleType = rt

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = rtdb.ProjectID
	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: rtdb.Name, ID: rtdb.ID}

	return resp, nil
}

// GetRuleTypeById is a method to get a rule type by id
func (s *Server) GetRuleTypeById(
	ctx context.Context,
	in *minderv1.GetRuleTypeByIdRequest,
) (*minderv1.GetRuleTypeByIdResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)

	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	resp := &minderv1.GetRuleTypeByIdResponse{}

	parsedRuleTypeID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid rule type ID")
	}

	rtdb, err := s.store.GetRuleTypeByID(ctx, parsedRuleTypeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, util.UserVisibleError(codes.NotFound, "rule type %s not found", in.GetId())
	} else if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	rt, err := ruletypes.RuleTypePBFromDB(&rtdb)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rtdb.Name, err)
	}

	resp.RuleType = rt

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = rtdb.ProjectID
	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: rtdb.Name, ID: rtdb.ID}

	return resp, nil
}

// CreateRuleType is a method to create a rule type
func (s *Server) CreateRuleType(
	ctx context.Context,
	crt *minderv1.CreateRuleTypeRequest,
) (*minderv1.CreateRuleTypeResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "error in entity context: %v", err)
	}

	projectID := entityCtx.Project.ID
	if err := validateGuidance(crt.RuleType.Guidance); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err)
	}

	newRuleType, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minderv1.RuleType, error) {
		return s.ruleTypes.CreateRuleType(ctx, projectID, uuid.Nil, crt.GetRuleType(), qtx)
	})
	if err != nil {
		if errors.Is(err, ruletypes.ErrRuleTypeInvalid) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid rule type definition: %s", err)
		} else if errors.Is(err, ruletypes.ErrRuleAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "rule type %s already exists", crt.RuleType.GetName())
		}
		return nil, status.Errorf(codes.Unknown, "failed to create rule type: %s", err)
	}

	return &minderv1.CreateRuleTypeResponse{
		RuleType: newRuleType,
	}, nil
}

// UpdateRuleType is a method to update a rule type
func (s *Server) UpdateRuleType(
	ctx context.Context,
	urt *minderv1.UpdateRuleTypeRequest,
) (*minderv1.UpdateRuleTypeResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	err := entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	projectID := entityCtx.Project.ID
	if err := validateGuidance(urt.RuleType.Guidance); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err)
	}

	updatedRuleType, err := db.WithTransaction(s.store, func(qtx db.ExtendQuerier) (*minderv1.RuleType, error) {
		return s.ruleTypes.UpdateRuleType(ctx, projectID, uuid.Nil, urt.GetRuleType(), qtx)
	})
	if err != nil {
		if errors.Is(err, ruletypes.ErrRuleTypeInvalid) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid rule type definition: %s", err)
		} else if errors.Is(err, ruletypes.ErrRuleNotFound) {
			return nil, status.Errorf(codes.NotFound, "rule type %s not found", urt.RuleType.GetName())
		}
		return nil, status.Errorf(codes.Unknown, "failed to update rule type: %s", err)
	}

	return &minderv1.UpdateRuleTypeResponse{
		RuleType: updatedRuleType,
	}, nil
}

// DeleteRuleType is a method to delete a rule type
func (s *Server) DeleteRuleType(
	ctx context.Context,
	in *minderv1.DeleteRuleTypeRequest,
) (*minderv1.DeleteRuleTypeResponse, error) {
	parsedRuleTypeID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid rule type ID")
	}

	// first read rule type by id, so we can get provider
	rtdb, err := s.store.GetRuleTypeByID(ctx, parsedRuleTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "rule type %s not found", in.GetId())
		}
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	// TEMPORARY HACK: Since we do not need to support the deletion of bundle
	// rule types yet, reject them in the API
	// TODO: Move this deletion logic to RuleTypeService
	if rtdb.SubscriptionID.Valid {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete rule type from bundle")
	}

	entityCtx := engcontext.EntityFromContext(ctx)

	err = entityCtx.ValidateProject(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	profiles, err := s.store.ListProfilesInstantiatingRuleType(ctx, rtdb.ID)
	// We have profiles that use this rule type, so we can't delete it
	if err == nil {
		if len(profiles) > 0 {
			return nil, util.UserVisibleError(codes.FailedPrecondition,
				"cannot delete: rule type %s is used by profiles %s", in.GetId(), strings.Join(profiles, ", "))
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		// If we failed for another reason, return an error
		return nil, status.Errorf(codes.Unknown, "failed to get profiles: %s", err)
	}

	// If there are no profiles instantiating this rule type, we can delete it
	err = s.store.DeleteRuleType(ctx, parsedRuleTypeID)
	if err != nil {
		// The rule got deleted in parallel?
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "rule type %s not found", in.GetId())
		}
		return nil, status.Errorf(codes.Unknown, "failed to delete rule type: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = rtdb.ProjectID
	logger.BusinessRecord(ctx).RuleType = logger.RuleType{Name: rtdb.Name, ID: rtdb.ID}

	return &minderv1.DeleteRuleTypeResponse{}, nil
}

func validateGuidance(guidance string) error {
	// As of the time of this writing, Minder profiles and rules
	// have a guidance that's less than 10kB long.
	if len(guidance) > 10*1<<10 {
		return fmt.Errorf(
			"%w: guidance too long",
			errInvalidRuleType,
		)
	}

	// The following lines validate that guidance is valid,
	// parseable markdown. Be mindful that any UTF-8 string is
	// valid markdown, so this is redundant at the moment. Should
	// the definition of `guidance` change to bytes, this check
	// would become much more relevant.
	md := goldmark.New(
		// GitHub Flavored Markdown
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	if err := md.Convert([]byte(guidance), &bytes.Buffer{}); err != nil {
		return fmt.Errorf(
			"%w: %s",
			errInvalidRuleType,
			err,
		)
	}

	return nil
}
