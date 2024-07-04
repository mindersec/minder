// Copyright 2024 Stacklok, Inc
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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// TokenValidationInterceptor is a server interceptor that validates the bearer token
func TokenValidationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (any, error) {

	opts, err := optionsForMethod(info)
	if err != nil {
		// Fail closed safely, rather than log and proceed.
		return nil, status.Errorf(codes.Internal, "Error getting options for method: %v", err)
	}

	ctx = withRpcOptions(ctx, opts)

	if opts.GetTargetResource() == minder.TargetResource_TARGET_RESOURCE_NONE {
		if !opts.GetNoLog() {
			zerolog.Ctx(ctx).Info().Msgf("Bypassing authentication")
		}
		return handler(ctx, req)
	}

	token, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		if statusErr, ok := status.FromError(err); ok {
			return nil, util.FromRpcError(statusErr)
		}
		return nil, status.Errorf(codes.Unauthenticated, "no auth token: %v", err)
	}

	server := info.Server.(*Server)

	parsedToken, err := server.jwt.ParseAndValidate(token)
	if err != nil {
		// We don't want to _actually_ log a bearer token.  JWTs will always be > 10 chars,
		// but by logging the start, we can see if it's actually a JWT or something else.
		shortToken := token
		if len(token) > 10 {
			shortToken = token[:10]
		}
		zerolog.Ctx(ctx).Info().Msgf("Error validating token %s", shortToken)
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	ctx = jwt.WithAuthTokenContext(ctx, parsedToken)

	// Attach the login sha for telemetry usage (hash of the user subject from the JWT)
	loginSHA := sha256.Sum256([]byte(parsedToken.Subject()))
	logger.BusinessRecord(ctx).LoginHash = hex.EncodeToString(loginSHA[:])

	return handler(ctx, req)
}

func withRpcOptions(ctx context.Context, opts *minder.RpcOptions) context.Context {
	return context.WithValue(ctx, rpcOptionsKey{}, opts)
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
