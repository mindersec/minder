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
func NewValidator() (protovalidate.Validator, error) {
	// TODO: add protovalidate.WithDescriptors() in the future
	return protovalidate.New()
}

// formatViolations is a helper function to format violations
func formatViolations(violations *validate.Violations) string {
	var res []string
	for _, v := range violations.Violations {
		res = append(res, fmt.Sprintf("- Field '%s': %s", getFullPath(v.Field), *v.Message))
	}
	return strings.Join(res, "\n")
}

func getFullPath(field *validate.FieldPath) string {
	var pathElements []string
	for _, element := range field.GetElements() {
		if element.GetFieldName() != "" {
			pathElements = append(pathElements, element.GetFieldName())
		} else if element.GetFieldNumber() != 0 {
			pathElements = append(pathElements, fmt.Sprintf("%d", element.GetFieldNumber()))
		}
		if element.GetSubscript() != nil {
			switch subscript := element.GetSubscript().(type) {
			case *validate.FieldPathElement_Index:
				pathElements[len(pathElements)-1] = fmt.Sprintf("%s[%d]", pathElements[len(pathElements)-1], subscript.Index)
			case *validate.FieldPathElement_BoolKey:
				pathElements[len(pathElements)-1] = fmt.Sprintf("%s[%t]", pathElements[len(pathElements)-1], subscript.BoolKey)
			case *validate.FieldPathElement_IntKey:
				pathElements[len(pathElements)-1] = fmt.Sprintf("%s[%d]", pathElements[len(pathElements)-1], subscript.IntKey)
			case *validate.FieldPathElement_UintKey:
				pathElements[len(pathElements)-1] = fmt.Sprintf("%s[%d]", pathElements[len(pathElements)-1], subscript.UintKey)
			case *validate.FieldPathElement_StringKey:
				pathElements[len(pathElements)-1] = fmt.Sprintf("%s[%s]", pathElements[len(pathElements)-1], subscript.StringKey)
			}
		}
	}
	return strings.Join(pathElements, ".")
}
