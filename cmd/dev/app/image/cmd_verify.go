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

package image

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/verifier"
	"github.com/stacklok/minder/internal/verifier/sigstore"
	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// CmdVerify is the root command for the container verify subcommands
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
	//nolint:goconst // let's not use a const for this one
	verifyCmd.Flags().StringP("token", "t", "", "token to authenticate to the provider."+
		//nolint:goconst // let's not use a const for this one
		"Can also be set via the AUTH_TOKEN environment variable.")
	verifyCmd.Flags().StringP("tuf-root", "r", sigstore.SigstorePublicTrustedRootRepo, "TUF root to use for verification")

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

	if err := viper.BindPFlag("auth.token", verifyCmd.Flags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %s\n", err)
		os.Exit(1)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	return verifyCmd
}

func runCmdVerify(cmd *cobra.Command, _ []string) error {
	owner := cmd.Flag("owner")
	name := cmd.Flag("name")
	digest := cmd.Flag("digest")
	tufRoot := cmd.Flag("tuf-root")

	token := viper.GetString("auth.token")

	ghcli, err := buildGitHubClient(token)
	if err != nil {
		return fmt.Errorf("cannot build github client: %w", err)
	}

	artifactVerifier, err := verifier.NewVerifier(
		verifier.VerifierSigstore,
		tufRoot.Value.String(),
		container.WithGitHubClient(ghcli))
	if err != nil {
		return fmt.Errorf("error getting sigstore verifier: %w", err)
	}

	res, err := artifactVerifier.Verify(context.Background(), verifyif.ArtifactTypeContainer,
		owner.Value.String(), name.Value.String(), digest.Value.String())
	if err != nil {
		return fmt.Errorf("error verifying container: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Result: %+v\n", res)
	return nil
}

func buildGitHubClient(token string) (provifv1.GitHub, error) {
	return clients.NewRestClient(
		&minderv1.GitHubProviderConfig{},
		nil,
		&ratecache.NoopRestClientCache{},
		credentials.NewGitHubTokenCredential(token),
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		properties.NewPropertyFetcherFactory(),
		"",
	)
}
