// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/oci"
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
	listCmd.Flags().StringP("owner", "o", "", "owner of the artifact")
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
	owner := cmd.Flag("owner")
	contname := cmd.Flag("container")

	if baseURL.Value.String() == "" {
		return fmt.Errorf("base URL is required")
	}
	if contname.Value.String() == "" {
		return fmt.Errorf("container name is required")
	}

	regWithOwner := fmt.Sprintf("%s/%s", owner.Value.String(), baseURL.Value.String())

	cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
	prov := oci.New(cred, baseURL.Value.String(), regWithOwner)

	// get the containers
	containers, err := prov.ListTags(ctx, contname.Value.String())
	if err != nil {
		return err
	}

	// print the containers
	for _, container := range containers {
		cmd.Println(container)
	}

	return nil
}
