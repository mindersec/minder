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
	"github.com/stacklok/minder/internal/providers/oci"
)

// CmdListTags returns the command for listing container tags
func CmdListTags() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:          "list-tags",
		Short:        "list container tags",
		RunE:         runCmdListTags,
		SilenceUsage: true,
	}

	listCmd.Flags().StringP("base-url", "b", "", "base URL for the OCI registry")
	listCmd.Flags().StringP("container", "c", "", "container name to list tags for")
	//nolint:goconst // let's not use a const for this one
	listCmd.Flags().StringP("token", "t", "", "token to authenticate to the provider."+
		"Can also be set via the AUTH_TOKEN environment variable.")

	if err := viper.BindPFlag("auth.token", listCmd.Flags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %s\n", err)
		os.Exit(1)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return listCmd
}

func runCmdListTags(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// get the provider
	baseURL := cmd.Flag("base-url")
	contname := cmd.Flag("container")

	if baseURL.Value.String() == "" {
		return fmt.Errorf("base URL is required")
	}
	if contname.Value.String() == "" {
		return fmt.Errorf("container name is required")
	}

	cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
	prov := oci.New(cred, baseURL.Value.String())

	// get the containers
	containers, err := prov.ListTags(ctx, contname.Value.String())
	if err != nil {
		return err
	}

	// print the containers
	for _, container := range containers {
		fmt.Println(container)
	}

	return nil
}
