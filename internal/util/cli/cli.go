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

// Package cli contains utility for the cli
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli/useragent"
)

// ErrWrappedCLIError is an error that wraps another error and provides a message used from within the CLI
type ErrWrappedCLIError struct {
	Message string
	Err     error
}

func (e *ErrWrappedCLIError) Error() string {
	return e.Err.Error()
}

// PrintYesNoPrompt prints a yes/no prompt to the user and returns false if the user did not respond with yes or y
func PrintYesNoPrompt(cmd *cobra.Command, promptMsg, confirmMsg, fallbackMsg string, defaultYes bool) bool {
	// Print the warning banner with the prompt message
	cmd.Println(WarningBanner.Render(promptMsg))

	// Determine the default confirmation value
	defConf := confirmation.No
	if defaultYes {
		defConf = confirmation.Yes
	}

	// Prompt the user for confirmation
	input := confirmation.New(confirmMsg, defConf)
	ok, err := input.RunPrompt()
	if err != nil {
		cmd.Println(WarningBanner.Render(fmt.Sprintf("Error reading input: %v", err)))
		ok = false
	}

	// If the user did not confirm, print the fallback message
	if !ok {
		cmd.Println(Header.Render(fallbackMsg))
	}
	return ok
}

// GrpcForCommand is a helper for getting a testing connection from cobra flags
func GrpcForCommand(v *viper.Viper) (*grpc.ClientConn, error) {
	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](v)
	if err != nil {
		return nil, fmt.Errorf("unable to read config: %w", err)
	}

	grpcHost := clientConfig.GRPCClientConfig.Host
	grpcPort := clientConfig.GRPCClientConfig.Port
	insecureDefault := grpcHost == "localhost" || grpcHost == "127.0.0.1" || grpcHost == "::1"
	allowInsecure := clientConfig.GRPCClientConfig.Insecure || insecureDefault

	issuerUrl := clientConfig.Identity.CLI.IssuerUrl
	clientId := clientConfig.Identity.CLI.ClientId

	return util.GetGrpcConnection(
		grpcHost, grpcPort, allowInsecure, issuerUrl, clientId, grpc.WithUserAgent(useragent.GetUserAgent()))
}

// GetAppContext is a helper for getting the cmd app context
func GetAppContext(ctx context.Context, v *viper.Viper) (context.Context, context.CancelFunc) {
	return GetAppContextWithTimeoutDuration(ctx, v, 10)
}

// GetAppContextWithTimeoutDuration is a helper for getting the cmd app context with a custom timeout
func GetAppContextWithTimeoutDuration(ctx context.Context, v *viper.Viper, tout int) (context.Context, context.CancelFunc) {
	v.SetDefault("cli.context_timeout", tout)
	timeout := v.GetInt("cli.context_timeout")

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	return ctx, cancel
}

// GRPCClientWrapRunE is a wrapper for cobra commands that sets up the grpc client and context
func GRPCClientWrapRunE(
	runEFunc func(ctx context.Context, cmd *cobra.Command, args []string, c *grpc.ClientConn) error,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}

		ctx, cancel := GetAppContext(cmd.Context(), viper.GetViper())
		defer cancel()

		c, err := GrpcForCommand(viper.GetViper())
		if err != nil {
			return err
		}

		defer c.Close()

		return runEFunc(ctx, cmd, args, c)
	}
}

// MessageAndError prints a message and returns an error.
func MessageAndError(msg string, err error) error {
	return &ErrWrappedCLIError{Message: msg, Err: err}
}

// ExitNicelyOnError print a message and exit with the right code
func ExitNicelyOnError(err error, userMsg string) {
	var message string
	var details string
	exitCode := 1 // Default to 1
	if err != nil {
		if userMsg != "" {
			// This handles the case where we want to print an explicit message before processing the error
			fmt.Fprintf(os.Stderr, "Message: %s\n", userMsg)
		}
		// Check if the error is wrapped
		var wrappedErr *ErrWrappedCLIError
		if errors.As(err, &wrappedErr) {
			// Print the wrapped message
			message = wrappedErr.Message
			// Continue processing the wrapped error
			err = wrappedErr.Err
		}
		// Check if the error is a grpc status
		if rpcStatus, ok := status.FromError(err); ok {
			nice := util.FromRpcError(rpcStatus)
			// If the error is unauthenticated, we want to print a helpful message and exit, no need to print details
			if rpcStatus.Code() == codes.Unauthenticated {
				message = "It seems you are logged out. Please run \"minder auth login\" first."
			} else {
				details = nice.Details
			}
			exitCode = int(nice.Code)
		} else {
			details = err.Error()
		}
		// Print the message, if any
		if message != "" {
			fmt.Fprintf(os.Stderr, "Message: %s\n", message)
		}
		// Print the details, if any
		if details != "" {
			fmt.Fprintf(os.Stderr, "Details: %s\n", details)
		}
		// Exit with the right code
		os.Exit(exitCode)
	}
}

// GetRepositoryName returns the repository name in the format owner/name
func GetRepositoryName(owner, name string) string {
	if owner == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", owner, name)
}

// ConcatenateAndWrap takes a string and a maximum line length (maxLen),
// then outputs the string as a multiline string where each line does not exceed maxLen characters.
func ConcatenateAndWrap(input string, maxLen int) string {
	if maxLen <= 0 {
		return input
	}

	var result string
	var lineLength int

	for _, runeValue := range input {
		// If the line length equals the len, append a newline and reset lineLength
		if lineLength == maxLen {
			if result[len(result)-1] != ' ' {
				// We trim at a word
				result += "-\n"
			} else {
				// We trim at a space, no need to add "-"
				result += "\n"
			}
			lineLength = 0
		}
		result += string(runeValue)
		lineLength++
	}

	return result
}
