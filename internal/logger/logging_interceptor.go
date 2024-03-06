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

// Package logger provides a general logging tools
package logger

import (
	"context"
	"encoding/json"
	"path"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	config "github.com/stacklok/minder/internal/config/server"
)

// Text is the constant for the text format
const Text = "text"

// Returns a resource log description for the given RPC method
func resource(method string) *zerolog.Event {
	return zerolog.Dict().Str("service", path.Dir(method)[1:]).Str("method", path.Base(method))
}

func commonAttributes(ctx context.Context, res *status.Status, duration time.Duration) *zerolog.Event {
	data := zerolog.Dict().Stringer("http.code", res.Code())
	meta, ok := metadata.FromIncomingContext(ctx)
	if ok {
		data = data.Fields(map[string]interface{}{
			"http.user_agent":   meta.Get("user-agent"),
			"http.content-type": meta.Get("content-type"),
			"http.duration":     duration.String(),
			"http.forwarded":    meta.Get("x-forwarded-for"),
		})
	}
	return data
}

// Interceptor creates a gRPC unary server interceptor that logs incoming
// requests and their responses using the Zerolog logger attached to the
// context.Context.  Successful requests are logged at the info level and
// error requests are logged at the error level.
//
// Returns:
//   - grpc.UnaryServerInterceptor: The gRPC unary server interceptor function.
//
// Example usage:
//
//	server := grpc.NewServer(
//	  ...
//	  grpc.UnaryServerInterceptor(logger.Interceptor(loggingConfig)),
//	  ...
//	)
func Interceptor(cfg config.LoggingConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Don't log health checks, they spam the logs
		if info.FullMethod == "/minder.v1.HealthService/CheckHealth" {
			return handler(ctx, req)
		}
		ts := TelemetryStore{}
		// Attach the resource to all logging events in the context
		logger := zerolog.Ctx(ctx).With().Dict("Resource", resource(info.FullMethod)).Logger()
		ctx = ts.WithTelemetry(logger.WithContext(ctx))
		now := time.Now()

		resp, err := handler(ctx, req)

		attrs := commonAttributes(ctx, status.Convert(err), time.Since(now))

		logMsg := logger.Info()
		if err != nil {
			logMsg = logger.Error()

			attrs = attrs.Err(err)
			if cfg.LogPayloads {
				if jsonText, err := json.Marshal(req); err == nil {
					logMsg = logMsg.RawJSON("Request", jsonText)
				}
			}
		}
		ts.Record(logMsg)

		// Note: Zerolog makes it hard to add attributes in multiple calls.
		logMsg.Dict("Attributes", attrs).Send()

		return resp, err
	}
}
