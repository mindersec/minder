// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package api package api provides a gRPC interceptor that validates incoming requests.
package api

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
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
			var validationErr *protovalidate.ValidationError
			if errors.As(err, &validationErr) {
				// Convert ValidationError to validate.Violations
				violations := validationErr.ToProto()
				// Convert violations to a util.NiceStatus message and return it
				return nil, util.UserVisibleError(codes.InvalidArgument, "Validation failed:\n%s", formatViolations(violations))
			}
			// Default to generic validation error
			return nil, status.Errorf(codes.InvalidArgument, "Validation failed: %v", err)
		}
		// Proceed to the handler
		return handler(ctx, req)
	}
}

// NewValidator creates a new validator.
func NewValidator() (*protovalidate.Validator, error) {
	options := []protovalidate.ValidatorOption{
		protovalidate.WithUTC(true),
		// TODO: add protovalidate.WithDescriptors() in the future
	}
	return protovalidate.New(options...)
}

// formatViolations is a helper function to format violations
func formatViolations(violations *validate.Violations) string {
	var res []string
	for _, v := range violations.Violations {
		res = append(res, fmt.Sprintf("- Field '%s': %s", *v.FieldPath, *v.Message))
	}
	return strings.Join(res, "\n")
}
