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

package rule_type

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sqlc-dev/pqtype"
	"google.golang.org/protobuf/reflect/protoreflect"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/rego"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// CmdTest is the root command for the rule subcommands
func CmdTest() *cobra.Command {
	var testCmd = &cobra.Command{
		Use:          "test",
		Short:        "test a rule type definition",
		Long:         `The 'rule type test' subcommand allows you test a rule type definition`,
		RunE:         testCmdRun,
		SilenceUsage: true,
	}

	testCmd.Flags().StringP("rule-type", "r", "", "file to read rule type definition from")
	testCmd.Flags().StringP("entity", "e", "", "YAML file containing the entity to test the rule against")
	testCmd.Flags().StringP("profile", "p", "", "YAML file containing a profile to test the rule against")
	testCmd.Flags().StringP("remediate-status", "", "", "The previous remediate status (optional)")
	testCmd.Flags().StringP("remediate-metadata", "", "", "YAML file containing the remediate metadata (optional)")
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
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	return testCmd
}

func testCmdRun(cmd *cobra.Command, _ []string) error {
	rtpath := cmd.Flag("rule-type")
	epath := cmd.Flag("entity")
	ppath := cmd.Flag("profile")
	rstatus := cmd.Flag("remediate-status")
	rMetaPath := cmd.Flag("remediate-metadata")
	token := viper.GetString("auth.token")

	// set rego env variable for debugging
	if err := os.Setenv(rego.EnablePrintEnvVar, "true"); err != nil {
		cmd.Printf("Unable to set %s environment variable: %s\n", rego.EnablePrintEnvVar, err)
		cmd.Println("If the rule you're testing is rego-based, you will not be able to use `print` statements for debugging.")
	}

	rt, err := readRuleTypeFromFile(rtpath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading rule type from file: %w", err)
	}

	provider := "test"
	rootProject := "00000000-0000-0000-0000-000000000002"
	rt.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &rootProject,
	}

	ent, err := readEntityFromFile(epath.Value.String(), minderv1.EntityFromString(rt.Def.InEntity))
	if err != nil {
		return fmt.Errorf("error reading entity from file: %w", err)
	}

	p, err := engine.ReadProfileFromFile(ppath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading fragment from file: %w", err)
	}

	remediateStatus := db.NullRemediationStatusTypes{}
	if rstatus.Value.String() != "" {
		remediateStatus = db.NullRemediationStatusTypes{
			RemediationStatusTypes: db.RemediationStatusTypes(rstatus.Value.String()),
			Valid:                  true,
		}
	}

	remMetadata := pqtype.NullRawMessage{}
	if rMetaPath.Value.String() != "" {
		f, err := os.Open(filepath.Clean(rMetaPath.Value.String()))
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}

		jsonMetadata := json.RawMessage{}
		err = json.NewDecoder(f).Decode(&jsonMetadata)
		if err != nil {
			return fmt.Errorf("error decoding json: %w", err)
		}

		remMetadata = pqtype.NullRawMessage{
			RawMessage: jsonMetadata,
			Valid:      true,
		}
	}

	// Disable actions
	off := "off"
	p.Alert = &off

	rules, err := engine.GetRulesFromProfileOfType(p, rt)
	if err != nil {
		return fmt.Errorf("error getting relevant fragment: %w", err)
	}

	// TODO: Read this from a providers file instead so we can make it pluggable
	eng, err := engine.NewRuleTypeEngine(context.Background(), p, rt, providers.NewProviderBuilder(
		&db.Provider{
			Name:    "test",
			Version: "v1",
			Implements: []db.ProviderType{
				"rest",
				"repo-lister",
				"git",
				"github",
			},
			Definition: json.RawMessage(`{
				"github-app": {}
			}`),
		},
		sql.NullString{},
		false,
		credentials.NewGitHubTokenCredential(token),
		&serverconfig.ProviderConfig{
			GitHubApp: &serverconfig.GitHubAppConfig{
				AppName: "test",
			},
		},
	))
	inf := &entities.EntityInfoWrapper{
		Entity:      ent,
		ExecutionID: &uuid.Nil,
	}
	if err != nil {
		return fmt.Errorf("error creating rule type engine: %w", err)
	}

	if len(rules) == 0 {
		return fmt.Errorf("no rules found with type %s", rt.Name)
	}

	return runEvaluationForRules(cmd, eng, inf, remediateStatus, remMetadata, rules)
}

func runEvaluationForRules(
	cmd *cobra.Command,
	eng *engine.RuleTypeEngine,
	inf *entities.EntityInfoWrapper,
	remediateStatus db.NullRemediationStatusTypes,
	remMetadata pqtype.NullRawMessage,
	frags []*minderv1.Profile_Rule,
) error {
	for idx := range frags {
		frag := frags[idx]

		val := eng.GetRuleInstanceValidator()
		err := val.ValidateRuleDefAgainstSchema(frag.Def.AsMap())
		if err != nil {
			return fmt.Errorf("error validating rule against schema: %w", err)
		}
		cmd.Printf("Profile valid according to the JSON schema!\n")

		if err := val.ValidateParamsAgainstSchema(frag.GetParams()); err != nil {
			return fmt.Errorf("error validating params against schema: %w", err)
		}

		// Create the eval status params
		evalStatus := &engif.EvalStatusParams{
			Rule: frag,
			EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{
				RemStatus:   remediateStatus,
				RemMetadata: remMetadata,
			},
		}
		// Perform rule evaluation
		evalStatus.SetEvalErr(eng.Eval(context.Background(), inf, evalStatus))

		// Perform the actions, if any
		evalStatus.SetActionsErr(context.Background(), eng.Actions(context.Background(), inf, evalStatus))

		if errors.IsActionFatalError(evalStatus.GetActionsErr().RemediateErr) {
			cmd.Printf("Remediation failed with fatal error: %s", evalStatus.GetActionsErr().RemediateErr)
		}

		if evalStatus.GetEvalErr() != nil {
			return fmt.Errorf("error evaluating rule type: %w", evalStatus.GetEvalErr())
		}

		cmd.Printf("The rule type is valid and the entity conforms to it\n")
	}

	return nil
}

func readRuleTypeFromFile(fpath string) (*minderv1.RuleType, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return minderv1.ParseRuleType(f)
}

// readEntityFromFile reads an entity from a file and returns it as a protobuf
// golang structure.
// TODO: We probably want to move this code to a utility once we land the server
// side code.
func readEntityFromFile(fpath string, entType minderv1.Entity) (protoreflect.ProtoMessage, error) {
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
	case minderv1.Entity_ENTITY_REPOSITORIES:
		out = &minderv1.Repository{}
	case minderv1.Entity_ENTITY_ARTIFACTS:
		out = &minderv1.Artifact{}
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		out = &minderv1.PullRequest{}
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return nil, fmt.Errorf("build environments not yet supported")
	case minderv1.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entType)
	}

	if err := json.NewDecoder(w).Decode(out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return out, nil
}
