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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

// Package cli contains utility for the cli
package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/constants"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli/useragent"
)

// PrintCmd prints a message using the output defined in the cobra Command
func PrintCmd(cmd *cobra.Command, msg string, args ...interface{}) {
	Print(cmd.OutOrStdout(), msg, args...)
}

// Print prints a message using the given io.Writer
func Print(out io.Writer, msg string, args ...interface{}) {
	fmt.Fprintf(out, msg+"\n", args...)
}

// PrintYesNoPrompt prints a yes/no prompt to the user and returns false if the user did not respond with yes or y
func PrintYesNoPrompt(cmd *cobra.Command, promptMsg, confirmMsg, fallbackMsg string, defaultYes bool) bool {
	// Print the warning banner with the prompt message
	PrintCmd(cmd, WarningBanner.Render(promptMsg))

	// Determine the default confirmation value
	defConf := confirmation.No
	if defaultYes {
		defConf = confirmation.Yes
	}

	// Prompt the user for confirmation
	input := confirmation.New(confirmMsg, defConf)
	ok, err := input.RunPrompt()
	if err != nil {
		PrintCmd(cmd, WarningBanner.Render(fmt.Sprintf("Error reading input: %v", err)))
		ok = false
	}

	// If the user did not confirm, print the fallback message
	if !ok {
		PrintCmd(cmd, Header.Render(fallbackMsg))
	}
	return ok
}

// GrpcForCommand is a helper for getting a testing connection from cobra flags
func GrpcForCommand(cmd *cobra.Command, v *viper.Viper) (*grpc.ClientConn, error) {
	grpc_host := util.GetConfigValue(v, "grpc_server.host", "grpc-host", cmd, constants.MinderGRPCHost).(string)
	grpc_port := util.GetConfigValue(v, "grpc_server.port", "grpc-port", cmd, 443).(int)
	insecureDefault := grpc_host == "localhost" || grpc_host == "127.0.0.1" || grpc_host == "::1"
	allowInsecure := util.GetConfigValue(v, "grpc_server.insecure", "grpc-insecure", cmd, insecureDefault).(bool)

	issuerUrl := util.GetConfigValue(v, "identity.cli.issuer_url", "identity-url", cmd, constants.IdentitySeverURL).(string)
	clientId := util.GetConfigValue(v, "identity.cli.client_id", "identity-client", cmd, "minder-cli").(string)

	return util.GetGrpcConnection(
		grpc_host, grpc_port, allowInsecure, issuerUrl, clientId, grpc.WithUserAgent(useragent.GetUserAgent()))
}

// GetAppContext is a helper for getting the cmd app context
func GetAppContext(ctx context.Context, v *viper.Viper) (context.Context, context.CancelFunc) {
	v.SetDefault("cli.context_timeout", 10)
	timeout := v.GetInt("cli.context_timeout")

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	return ctx, cancel
}

// GRPCClientWrapRunE is a wrapper for cobra commands that sets up the grpc client and context
func GRPCClientWrapRunE(
	runEFunc func(ctx context.Context, cmd *cobra.Command, c *grpc.ClientConn) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetAppContext(cmd.Context(), viper.GetViper())
		defer cancel()

		c, err := GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return err
		}

		defer c.Close()

		return runEFunc(ctx, cmd, c)
	}
}
