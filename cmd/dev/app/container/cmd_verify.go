// Copyright 2023 Stacklok, Inc.
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

// Package container provides the root command for the container subcommands
package container

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/verifier"
)

// CmdVerify returns the verify container command
func CmdVerify() *cobra.Command {
	var verifyCmd = &cobra.Command{
		Use:          "verify",
		Short:        "verify a container signature",
		RunE:         runCmdVerify,
		SilenceUsage: true,
	}

	verifyCmd.Flags().StringP("owner", "o", "", "owner of the artifact")
	verifyCmd.Flags().StringP("name", "n", "", "name of the artifact")
	verifyCmd.Flags().StringP("digest", "s", "", "digest of the artifact")

	if err := verifyCmd.MarkFlagRequired("owner"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := verifyCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := verifyCmd.MarkFlagRequired("digest"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return verifyCmd
}

func runCmdVerify(cmd *cobra.Command, _ []string) error {
	owner := cmd.Flag("owner")
	name := cmd.Flag("name")
	digest := cmd.Flag("digest")

	token := viper.GetString("auth.token")

	artifactVerifier, err := verifier.NewVerifier(verifier.VerifierSigstore, token)
	if err != nil {
		return fmt.Errorf("error getting sigstore verifier: %w", err)
	}
	defer artifactVerifier.ClearCache()

	res, err := artifactVerifier.Verify(context.Background(), verifier.ArtifactTypeContainer, "",
		owner.Value.String(), name.Value.String(), digest.Value.String())
	if err != nil {
		return fmt.Errorf("error verifying container: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Signature: %s\n", res.SignatureInfo)
	fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s\n", res.WorkflowInfo)
	fmt.Fprintf(cmd.OutOrStdout(), "URI: %s\n", res.URI)

	return nil
}
