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
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
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
func PrintYesNoPrompt(cmd *cobra.Command, promptMsg, fallbackMsg string) bool {
	reader := bufio.NewReader(os.Stdin)
	// Print the warning banner with the prompt message
	PrintCmd(cmd, WarningBanner.Render(promptMsg))

	// Read the response
	response, _ := reader.ReadString('\n')

	// Normalise the response
	response = strings.ToLower(strings.TrimSpace(response))
	if response != "yes" && response != "y" {
		// Prompt was not confirmed, print the fallback message and return false
		PrintCmd(cmd, Header.Render(fallbackMsg))
		return false
	}
	// Prompt was confirmed, return true
	return true
}
