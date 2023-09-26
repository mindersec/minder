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

// Package rule_type provides the CLI subcommand for developing rules
// e.g. the 'rule type test' subcommand.
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
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/cmd/dev/app"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	"github.com/stacklok/mediator/internal/entities"
	"github.com/stacklok/mediator/internal/providers"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// TestCmd is the root command for the rule subcommands
var testCmd = &cobra.Command{
	Use:          "rule type test",
	Short:        "test a rule type definition",
	Long:         `The 'rule type test' subcommand allows you test a rule type definition`,
	RunE:         testCmdRun,
	SilenceUsage: true,
}

func init() {
	app.RootCmd.AddCommand(testCmd)
	testCmd.Flags().StringP("rule-type", "r", "", "file to read rule type definition from")
	testCmd.Flags().StringP("entity", "e", "", "YAML file containing the entity to test the rule against")
	testCmd.Flags().StringP("policy", "p", "", "YAML file containing a policy to test the rule against")
	testCmd.Flags().StringP("token", "t", "", "token to authenticate to the provider."+
		"Can also be set via the AUTH_TOKEN environment variable.")

	if err := testCmd.MarkFlagRequired("rule-type"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := testCmd.MarkFlagRequired("entity"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlag("auth.token", testCmd.Flags().Lookup("token")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %s\n", err)
		os.Exit(1)
	}
	// bind environment variable
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

func testCmdRun(cmd *cobra.Command, _ []string) error {
	rtpath := cmd.Flag("rule-type")
	epath := cmd.Flag("entity")
	ppath := cmd.Flag("policy")
	token := viper.GetString("auth.token")

	// set rego env variable for debugging
	if err := os.Setenv(rego.EnablePrintEnvVar, "true"); err != nil {
		fmt.Printf("Unable to set %s environment variable: %s\n", rego.EnablePrintEnvVar, err)
		fmt.Println("If the rule you're testing is rego-based, you will not be able to use `print` statements for debugging.")
	}

	rt, err := readRuleTypeFromFile(rtpath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading rule type from file: %w", err)
	}

	ent, err := readEntityFromFile(epath.Value.String(), entities.FromString(rt.Def.InEntity))
	if err != nil {
		return fmt.Errorf("error reading entity from file: %w", err)
	}

	p, err := engine.ReadPolicyFromFile(ppath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading fragment from file: %w", err)
	}

	rules, err := engine.GetRulesFromPolicyOfType(p, rt)
	if err != nil {
		return fmt.Errorf("error getting relevant fragment: %w", err)
	}

	// TODO: Read this from a providers file instead so we can make it pluggable
	eng, err := engine.NewRuleTypeEngine(rt, providers.NewProviderBuilder(
		&db.Provider{
			Name: "test",
			Implements: []db.ProviderType{
				"rest",
				"git",
				"github",
			},
		},
		db.ProviderAccessToken{},
		token,
	))
	if err != nil {
		return fmt.Errorf("error creating rule type engine: %w", err)
	}

	if len(rules) == 0 {
		return fmt.Errorf("no rules found with type %s", rt.Name)
	}

	return runEvaluationForRules(eng, ent, rules)
}

func runEvaluationForRules(eng *engine.RuleTypeEngine, ent protoreflect.ProtoMessage, frags []*pb.Policy_Rule) error {
	for idx := range frags {
		frag := frags[idx]

		def := frag.Def.AsMap()
		val := eng.GetRuleInstanceValidator()
		err := val.ValidateRuleDefAgainstSchema(def)
		if err != nil {
			return fmt.Errorf("error validating rule against schema: %w", err)
		}
		fmt.Printf("Policy valid according to the JSON schema!\n")

		var params map[string]any
		if err := val.ValidateParamsAgainstSchema(frag.GetParams()); err != nil {
			return fmt.Errorf("error validating params against schema: %w", err)
		}

		if frag.GetParams() != nil {
			params = frag.GetParams().AsMap()
		}

		if err := eng.Eval(context.Background(), ent, def, params); err != nil {
			return fmt.Errorf("error evaluating rule type: %w", err)
		}

		fmt.Printf("The rule type is valid and the entity conforms to it\n")
	}

	return nil
}

func readRuleTypeFromFile(fpath string) (*pb.RuleType, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return engine.ParseRuleType(f)
}

// readEntityFromFile reads an entity from a file and returns it as a protobuf
// golang structure.
// TODO: We probably want to move this code to a utility once we land the server
// side code.
func readEntityFromFile(fpath string, entType pb.Entity) (protoreflect.ProtoMessage, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := util.TranscodeYAMLToJSON(f, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	var out protoreflect.ProtoMessage

	switch entType {
	case pb.Entity_ENTITY_REPOSITORIES:
		out = &pb.RepositoryResult{}
	case pb.Entity_ENTITY_ARTIFACTS:
		out = &pb.VersionedArtifact{}
	case pb.Entity_ENTITY_PULL_REQUESTS:
		out = &pb.PullRequest{}
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return nil, fmt.Errorf("build environments not yet supported")
	case pb.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entType)
	}

	if err := json.NewDecoder(w).Decode(out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return out, nil
}
