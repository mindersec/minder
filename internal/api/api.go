// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package api package api provides a gRPC interceptor that validates incoming requests.
package api

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
)

// ProtoValidationInterceptor is a gRPC interceptor that validates incoming requests.
func ProtoValidationInterceptor(validator protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Assert that req implements proto.Message
		msg, ok := req.(proto.Message)
		if !ok {
			// Handle the error: req is not a proto.Message
			return nil, status.Errorf(codes.Internal, "Request does not implement proto.Message")
		}

		// Validate the incoming request
		if err := validator.Validate(msg); err != nil {
			var validationErr *protovalidate.ValidationError
			if errors.As(err, &validationErr) {
				return nil, util.UserVisibleError(codes.InvalidArgument, "%s", validationErr.Error())
			}
			// Default to generic validation error
			return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
		}
		// Proceed to the handler
		return handler(ctx, req)
	}
}

// NewValidator creates a new validator.
func NewValidator() (protovalidate.Validator, error) {
	// TODO: add protovalidate.WithDescriptors() in the future
	return protovalidate.New()
}
