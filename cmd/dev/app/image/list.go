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
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/dockerhub"
	"github.com/mindersec/minder/internal/providers/github/ghcr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// CmdList returns the command for listing containers
func CmdList() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list images",
		RunE:         runCmdList,
		SilenceUsage: true,
	}

	listCmd.Flags().StringP("provider", "p", "", "provider class to use for listing containers")
	listCmd.Flags().StringP("namespace", "n", "", "namespace to list containers from")
	//nolint:goconst // let's not use a const for this one
	listCmd.Flags().StringP("token", "t", "", "token to authenticate to the provider."+
		//nolint:goconst // let's not use a const for this one
		"Can also be set via the AUTH_TOKEN environment variable.")

	if err := viper.BindPFlag("auth.token", listCmd.Flags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %s\n", err)
		os.Exit(1)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return listCmd
}

func runCmdList(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// get the provider
	pclass := cmd.Flag("provider")
	if pclass.Value.String() == "" {
		return fmt.Errorf("provider class is required")
	}
	ns := cmd.Flag("namespace")
	if ns.Value.String() == "" {
		return fmt.Errorf("namespace is required")
	}

	var prov provifv1.ImageLister
	switch pclass.Value.String() {
	case "dockerhub":
		var err error
		cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
		prov, err = dockerhub.New(cred, &minderv1.DockerHubProviderConfig{
			Namespace: proto.String(ns.Value.String()),
		})
		if err != nil {
			return err
		}
	case "ghcr":
		cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
		prov = ghcr.New(cred, &minderv1.GHCRProviderConfig{
			Namespace: proto.String(ns.Value.String()),
		})
	default:
		return fmt.Errorf("unknown provider: %s", pclass.Value.String())
	}

	// get the containers
	containers, err := prov.ListImages(ctx)
	if err != nil {
		return err
	}

	// print the containers
	for _, container := range containers {
		cmd.Println(container)
	}

	return nil
}
