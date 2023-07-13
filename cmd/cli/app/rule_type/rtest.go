// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package rule_type

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// TestCmd is the root command for the rule subcommands
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test a rule type definition",
	Long:  `The 'rule_type test' subcommand allows you test a rule type definition`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fpath := cmd.Flag("file")
		epath := cmd.Flag("entity")

		rt, err := readRuleTypeFromFile(fpath.Value.String())
		if err != nil {
			return fmt.Errorf("error reading rule type from file: %w", err)
		}

		fmt.Printf("Rule Type: %+v\n", rt)

		e, err = readEntityFromFile(epath.Value.String(), rt.Def.InEntity)
		if err != nil {
			return fmt.Errorf("error reading entity from file: %w", err)
		}

		_, err = getProviderClient(context.Background(), rt)
		if err != nil {
			return fmt.Errorf("error getting provider client: %w", err)
		}

		return nil
	},
}

func readRuleTypeFromFile(fpath string) (*pb.RuleType, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := util.TranscodeYAMLToJSON(f, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	r := &pb.RuleType{}
	if err := json.NewDecoder(w).Decode(r); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return r, nil
}

// readEntityFromFile reads an entity from a file and returns it as a protobuf
// golang structure.
// TODO: We probably want to move this code to a utility once we land the server
// side code.
func readEntityFromFile(fpath string, entType string) (any, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := util.TranscodeYAMLToJSON(f, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	var out any

	switch entType {
	case "repository":
		out = &pb.GetRepositoryResponse{}
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entType)
	}

	if err := json.NewDecoder(w).Decode(out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return out, nil
}

// getProviderClient returns a client for the provider specified in the rule type
// definition.
// TODO: This should be moved to a provider package and we should have some
// generic interface for clients.
func getProviderClient(ctx context.Context, rt *pb.RuleType) (ghclient.RestAPI, error) {
	token := viper.GetString("auth.token")
	switch rt.Context.Provider {
	case ghclient.Github:
		return ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
			Token: token,
		})
	default:
		return nil, fmt.Errorf("unknown provider: %s", rt.Context.Provider)
	}
}

func init() {
	ruleTypeCmd.AddCommand(testCmd)
	testCmd.Flags().StringP("file", "f", "", "file to read rule type definition from")
	if err := testCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	testCmd.Flags().StringP("token", "t", "", "token to authenticate to the provider."+
		"Can also be set via the AUTH_TOKEN environment variable.")
	if err := viper.BindPFlag("auth.token", testCmd.Flags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %s\n", err)
		os.Exit(1)
	}
	// bind environment variable
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	testCmd.Flags().StringP("entity", "e", "", "YAML file containing the entity to test the rule against")
	if err := testCmd.MarkFlagRequired("entity"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
