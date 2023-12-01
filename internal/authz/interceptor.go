//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package authz provides the authorization model for minder
package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// AuthzInterceptor is a gRPC interceptor that checks the authorization of the
// request.
func AuthzInterceptor(enf *casbin.Enforcer, jwtVal auth.JwtValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		opts, err := optionsForMethod(info)
		if err != nil {
			// Fail closed safely, rather than log and proceed.
			return nil, status.Errorf(codes.Internal, "Error getting options for method: %v", err)
		}

		ctx = withRpcOptions(ctx, opts)

		if opts.GetAnonymous() {
			if !opts.GetNoLog() {
				zerolog.Ctx(ctx).Info().Msgf("Bypassing authentication")
			}
			return handler(ctx, req)
		}

		token, err := gauth.AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "no auth token: %v", err)
		}

		parsedToken, err := jwtVal.ParseAndValidate(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
		}

		// TODO Call enforcer

		ret, err := handler(ctx, req)
		return ret, err
	}
}

func optionsForMethod(info *grpc.UnaryServerInfo) (*minder.RpcOptions, error) {
	formattedName := strings.ReplaceAll(info.FullMethod[1:], "/", ".")
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(formattedName))
	if err != nil {
		return nil, fmt.Errorf("unable to find descriptor for %q: %w", formattedName, err)
	}
	extension := proto.GetExtension(descriptor.Options(), minder.E_RpcOptions)
	opts, ok := extension.(*minder.RpcOptions)
	if !ok {
		return nil, fmt.Errorf("couldn't decode option for %q, wrong type: %T", formattedName, extension)
	}
	return opts, nil
}

type rpcOptionsKey struct{}

func withRpcOptions(ctx context.Context, opts *minder.RpcOptions) context.Context {
	return context.WithValue(ctx, rpcOptionsKey{}, opts)
}

func getRpcOptions(ctx context.Context) *minder.RpcOptions {
	// nil value default is okay here
	opts, _ := ctx.Value(rpcOptionsKey{}).(*minder.RpcOptions)
	return opts
}
