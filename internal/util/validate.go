// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ProtoValidationInterceptor is a gRPC interceptor that validates incoming requests.
func ProtoValidationInterceptor(validator *protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Assert that req implements proto.Message
		msg, ok := req.(proto.Message)
		if !ok {
			// Handle the error: req is not a proto.Message
			return nil, status.Errorf(codes.Internal, "Request does not implement proto.Message")
		}

		// Validate the incoming request
		if err := validator.Validate(msg); err != nil {
			// Return validation error
			return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
		}
		// Proceed to the handler
		return handler(ctx, req)
	}
}

// NewValidator creates a new validator.
func NewValidator() (*protovalidate.Validator, error) {
	return protovalidate.New()
}
