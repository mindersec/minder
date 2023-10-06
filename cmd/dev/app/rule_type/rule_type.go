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
	"github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	"github.com/stacklok/mediator/internal/util/jsonyaml"
	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
	testCmd.Flags().StringP("profile", "p", "", "YAML file containing a profile to test the rule against")
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
	ppath := cmd.Flag("profile")
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

	rootProject := "00000000-0000-0000-0000-000000000002"
	rt.Context = &mediatorv1.Context{
		Provider: "test",
		Project:  &rootProject,
	}

	ent, err := readEntityFromFile(epath.Value.String(), mediatorv1.EntityFromString(rt.Def.InEntity))
	if err != nil {
		return fmt.Errorf("error reading entity from file: %w", err)
	}

	p, err := engine.ReadProfileFromFile(ppath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading fragment from file: %w", err)
	}

	rules, err := engine.GetRulesFromProfileOfType(p, rt)
	if err != nil {
		return fmt.Errorf("error getting relevant fragment: %w", err)
	}

	// TODO: Read this from a providers file instead so we can make it pluggable
	eng, err := engine.NewRuleTypeEngine(rt, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "test",
			Version: "v1",
			Implements: []db.ProviderType{
				"rest",
				"git",
				"github",
			},
			Definition: json.RawMessage(`{
				"rest": {},
				"github": {}
			}`),
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

	return runEvaluationForRules(eng, ent, interfaces.RemediationActionOptFromString(p.Remediate), rules)
}

func runEvaluationForRules(
	eng *engine.RuleTypeEngine,
	ent protoreflect.ProtoMessage,
	rem interfaces.RemediateActionOpt,
	frags []*mediatorv1.Profile_Rule,
) error {
	for idx := range frags {
		frag := frags[idx]

		def := frag.Def.AsMap()
		val := eng.GetRuleInstanceValidator()
		err := val.ValidateRuleDefAgainstSchema(def)
		if err != nil {
			return fmt.Errorf("error validating rule against schema: %w", err)
		}
		fmt.Printf("Profile valid according to the JSON schema!\n")

		var params map[string]any
		if err := val.ValidateParamsAgainstSchema(frag.GetParams()); err != nil {
			return fmt.Errorf("error validating params against schema: %w", err)
		}

		if frag.GetParams() != nil {
			params = frag.GetParams().AsMap()
		}

		evalErr, remediateErr := eng.Eval(context.Background(), ent, def, params, rem)
		if evalErr != nil {
			return fmt.Errorf("error evaluating rule type: %w", evalErr)
		}

		if errors.IsRemediateFatalError(remediateErr) {
			fmt.Printf("Remediation failed with fatal error: %s", remediateErr)
		}

		fmt.Printf("The rule type is valid and the entity conforms to it\n")
	}

	return nil
}

func readRuleTypeFromFile(fpath string) (*mediatorv1.RuleType, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return mediatorv1.ParseRuleType(f)
}

// readEntityFromFile reads an entity from a file and returns it as a protobuf
// golang structure.
// TODO: We probably want to move this code to a utility once we land the server
// side code.
func readEntityFromFile(fpath string, entType mediatorv1.Entity) (protoreflect.ProtoMessage, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := jsonyaml.TranscodeYAMLToJSON(f, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	var out protoreflect.ProtoMessage

	switch entType {
	case mediatorv1.Entity_ENTITY_REPOSITORIES:
		out = &mediatorv1.RepositoryResult{}
	case mediatorv1.Entity_ENTITY_ARTIFACTS:
		out = &mediatorv1.Artifact{}
	case mediatorv1.Entity_ENTITY_PULL_REQUESTS:
		out = &mediatorv1.PullRequest{}
	case mediatorv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return nil, fmt.Errorf("build environments not yet supported")
	case mediatorv1.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entType)
	}

	if err := json.NewDecoder(w).Decode(out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return out, nil
}
