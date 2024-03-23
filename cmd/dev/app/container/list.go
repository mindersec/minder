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

	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	"github.com/stacklok/minder/internal/providers/github/ghcr"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// CmdList returns the command for listing containers
func CmdList() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:          "list",
		Short:        "list containers",
		RunE:         runCmdList,
		SilenceUsage: true,
	}

	listCmd.Flags().StringP("provider", "p", "", "provider class to use for listing containers")
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

	var prov provifv1.ImageLister
	switch pclass.Value.String() {
	case "dockerhub":
		var err error
		cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
		prov, err = dockerhub.New(cred, "devopsfaith")
		if err != nil {
			return err
		}
	case "ghcr":
		cred := credentials.NewOAuth2TokenCredential(viper.GetString("auth.token"))
		prov = ghcr.New(cred, "jaormx")
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
		fmt.Println(container)
	}

	return nil
}
