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

package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/pkg/util"
)

// HTTPServerConfig is the configuration for the HTTP server
type HTTPServerConfig struct {
	// Host is the host to bind to
	Host string `mapstructure:"host"`
	// Port is the port to bind to
	Port int `mapstructure:"port"`
}

// GetAddress returns the address to bind to
func (s *HTTPServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GRPCServerConfig is the configuration for the gRPC server
type GRPCServerConfig struct {
	// Host is the host to bind to
	Host string `mapstructure:"host"`
	// Port is the port to bind to
	Port int `mapstructure:"port"`
}

// GetAddress returns the address to bind to
func (s *GRPCServerConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// RegisterHTTPServerFlags registers the flags for the HTTP server
func RegisterHTTPServerFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := util.BindConfigFlag(v, flags, "http_server.host", "http-host", "",
		"The host to bind to for the HTTP server", flags.String)
	if err != nil {
		return err
	}

	return util.BindConfigFlag(v, flags, "http_server.port", "http-port", 8080,
		"The port to bind to for the HTTP server", flags.Int)
}

// RegisterGRPCServerFlags registers the flags for the gRPC server
func RegisterGRPCServerFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := util.BindConfigFlag(v, flags, "grpc_server.host", "grpc-host", "",
		"The host to bind to for the gRPC server", flags.String)
	if err != nil {
		return err
	}

	return util.BindConfigFlag(v, flags, "grpc_server.port", "grpc-port", 8090,
		"The port to bind to for the gRPC server", flags.Int)
}
