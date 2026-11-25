// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	exceptionSvc "github.com/mindersec/minder/internal/exception/service"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ListExceptions lists active exceptions for the current project.
func (s *Server) ListExceptions(
	ctx context.Context,
	_ *pb.ListExceptionsRequest,
) (*pb.ListExceptionsResponse, error) {
	projectID := GetProjectID(ctx)

	exceptions, err := s.exceptions.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	resp := &pb.ListExceptionsResponse{
		Exceptions: make([]*pb.Exception, 0, len(exceptions)),
	}

	for _, exception := range exceptions {
		pbException, err := s.exceptionToProto(ctx, projectID, exception)
		if err != nil {
			return nil, err
		}
		resp.Exceptions = append(resp.Exceptions, pbException)
	}

	return resp, nil
}

// CreateException creates or updates an exception for the current project.
func (s *Server) CreateException(
	ctx context.Context,
	in *pb.CreateExceptionRequest,
) (*pb.CreateExceptionResponse, error) {
	entityID, err := uuid.Parse(in.GetEntityId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid entity ID")
	}

	expiresAt := in.GetExpiresAt()
	if expiresAt == nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "expires_at is required")
	}
	if err := expiresAt.CheckValid(); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid expires_at: %s", err)
	}

	exception, err := s.exceptions.Create(ctx, exceptionSvc.Exception{
		EntityID:       entityID,
		ProfileName:    in.GetProfileName(),
		RuleName:       in.GetRuleName(),
		Reason:         in.GetReason(),
		ReasonProvided: in.Reason != nil,
		ExpiresAt:      expiresAt.AsTime(),
	})
	if err != nil {
		return nil, err
	}

	pbException, err := s.exceptionToProto(ctx, GetProjectID(ctx), *exception)
	if err != nil {
		return nil, err
	}

	return &pb.CreateExceptionResponse{
		Exception: pbException,
	}, nil
}

// DeleteException deletes an exception from the current project.
func (s *Server) DeleteException(
	ctx context.Context,
	in *pb.DeleteExceptionRequest,
) (*pb.DeleteExceptionResponse, error) {
	exceptionID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid exception ID")
	}

	if err := s.exceptions.Delete(ctx, exceptionID, GetProjectID(ctx)); err != nil {
		return nil, err
	}

	return &pb.DeleteExceptionResponse{}, nil
}

// exceptionToProto converts a service exception into its protobuf representation.
func (s *Server) exceptionToProto(
	ctx context.Context,
	projectID uuid.UUID,
	exception exceptionSvc.Exception,
) (*pb.Exception, error) {
	entity, err := s.entityService.GetEntityByID(ctx, exception.EntityID, projectID)
	if err != nil {
		return nil, err
	}

	return &pb.Exception{
		Id:          exception.ID.String(),
		Entity:      entity,
		ExpiresAt:   timestamppb.New(exception.ExpiresAt),
		CreatedAt:   timestamppb.New(exception.CreatedAt),
		ProfileName: exception.ProfileName,
		RuleName:    exception.RuleName,
		Reason:      exception.Reason,
	}, nil
}
