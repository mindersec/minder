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

package logger

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	Marshaller = &jsonpb.Marshaler{}
	MaxSize    = 2048000
)

func LogTimestamp(logger *zerolog.Event, t time.Time) {
	*logger = *logger.Int64("Timestamp", t.UnixNano())
}

func LogResource(logger *zerolog.Event, dict map[string]interface{}) {
	jsonData, err := json.Marshal(dict)
	if err == nil {
		*logger = *logger.Fields(map[string]interface{}{"Resource": jsonData})
	}
}

func LogAttributes(logger *zerolog.Event, dict map[string]interface{}) {
	jsonData, err := json.Marshal(dict)
	if err == nil {
		*logger = *logger.Fields(map[string]interface{}{"Attributes": jsonData})
	}
}

// will logs calls based on https://github.com/open-telemetry/oteps/blob/main/text/logs/0097-log-data-model.md#example-log-records
func LogIncomingCall(ctx context.Context, logger *zerolog.Event, method string, t time.Time, req interface{}, res *status.Status) {

	LogTimestamp(logger, t)
	LogResource(logger, map[string]interface{}{
		"service": path.Dir(method)[1:],
		"method":  path.Base(method),
	})

	metadata, ok := metadata.FromIncomingContext(ctx)
	if ok {
		LogAttributes(logger, map[string]interface{}{
			"http.user_agent":   metadata.Get("user-agent"),
			"http.content-type": metadata.Get("content-type"),
			"http.code":         res.Code(),
			"http.duration":     time.Since(t).String(),
		})
	}

}

func LogStatusError(logger *zerolog.Event, err error) {
	statusErr := status.Convert(err)
	*logger = *logger.Err(err).Str("status", statusErr.Code().String()).Str("msg", statusErr.Message()).Interface("details", statusErr.Details())
}

func ViperLogLevelToZerologLevel(viperLogLevel string) zerolog.Level {
	switch viperLogLevel {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel // Default to info level if the mapping is not found
	}
}

// LoggerInterceptor creates a gRPC unary server interceptor that logs incoming requests and their responses using Zerolog.
// The interceptor logs the requests with the specified log level.
//
// Parameters:
//   - logLevel: The log level to use for logging. Valid values are "debug", "info", "warn", "error", and "fatal".
//
// Returns:
//   - grpc.UnaryServerInterceptor: The gRPC unary server interceptor function.
//
// Example usage:
//
//	logInterceptor := LoggerInterceptor("info")
//	server := grpc.NewServer(
//	  ...
//	  grpc.UnaryInterceptor(logInterceptor),
//	  ...
//	)
func LoggerInterceptor(logLevel string) grpc.UnaryServerInterceptor {
	// set log level according to config
	zlevel := ViperLogLevelToZerologLevel(logLevel)
	zerolog.SetGlobalLevel(zlevel)
	zlog := zerolog.New(os.Stdout).With().Timestamp().Logger()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		now := time.Now()
		resp, err := handler(ctx, req)
		ret := status.Convert(err)

		if zlog.Error().Enabled() {
			var logger *zerolog.Event
			if err != nil {
				logger = zlog.Error()
			} else {
				logger = zlog.Info()
			}
			LogIncomingCall(ctx, logger, info.FullMethod, now, req, ret)
			logger.Send()
		}
		return resp, err
	}
}
