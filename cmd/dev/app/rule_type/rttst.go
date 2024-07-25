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
	"github.com/stacklok/minder/internal/engine/actions"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/rego"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/engine/rtengine"
	"github.com/stacklok/minder/internal/engine/selectors"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/profiles/models"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	"github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provsel "github.com/stacklok/minder/internal/providers/selectors"
	"github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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

	testCmd.Flags().String("log-level", "error", "Log Level")
	testCmd.Flags().StringP("rule-type", "r", "", "file to read rule type definition from")
	testCmd.Flags().StringP("entity", "e", "", "YAML file containing the entity to test the rule against")
	testCmd.Flags().StringP("profile", "p", "", "YAML file containing a profile to test the rule against")
	testCmd.Flags().StringP("provider", "P", "github", "The provider class to test the rule against")
	testCmd.Flags().StringP("provider-config", "c", "", "YAML file containing the provider configuration (optional)")
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
	providerclass := cmd.Flag("provider")
	providerconfig := cmd.Flag("provider-config")

	// set rego env variable for debugging
	if err := os.Setenv(rego.EnablePrintEnvVar, "true"); err != nil {
		cmd.Printf("Unable to set %s environment variable: %s\n", rego.EnablePrintEnvVar, err)
		cmd.Println("If the rule you're testing is rego-based, you will not be able to use `print` statements for debugging.")
	}

	ruletype, err := readRuleTypeFromFile(rtpath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading rule type from file: %w", err)
	}

	provider := "test"
	rootProject := "00000000-0000-0000-0000-000000000002"
	ruletype.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &rootProject,
	}

	eiw, err := getEiwFromFile(ruletype, epath.Value.String())
	if err != nil {
		return fmt.Errorf("error reading entity from file: %w", err)
	}

	profile, err := profiles.ReadProfileFromFile(ppath.Value.String())
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
	profile.Alert = &off

	rules, err := rtengine.GetRulesFromProfileOfType(profile, ruletype)
	if err != nil {
		return fmt.Errorf("error getting relevant fragment: %w", err)
	}
	if len(rules) == 0 {
		return fmt.Errorf("no rules found with type %s", ruletype.Name)
	}

	// TODO: Whenever we add more Provider classes, we will need to rethink this
	prov, err := getProvider(providerclass.Value.String(), token, providerconfig.Value.String())
	if err != nil {
		return err
	}

	actionConfig := models.ActionConfiguration{
		Remediate: actionOptFromString(profile.Remediate, models.ActionOptOff),
		Alert:     actionOptFromString(profile.Alert, models.ActionOptOff),
	}

	// TODO: use cobra context here
	ctx := context.Background()
	eng, err := rtengine.NewRuleTypeEngine(ctx, ruletype, prov)
	if err != nil {
		return fmt.Errorf("cannot create rule type engine: %w", err)
	}
	actionEngine, err := actions.NewRuleActions(ctx, ruletype, prov, &actionConfig)
	if err != nil {
		return fmt.Errorf("cannot create rule actions engine: %w", err)
	}

	profSel, err := getProfileSelectors(eiw.Type, profile)
	if err != nil {
		return fmt.Errorf("error creating selectors: %w", err)
	}

	return runEvaluationForRules(cmd, eng, eiw, prov, profSel, remediateStatus, remMetadata, rules, actionEngine)
}

func getProfileSelectors(entType minderv1.Entity, profile *minderv1.Profile) (selectors.Selection, error) {
	selectorEnv := selectors.NewEnv()

	profSel, err := selectorEnv.NewSelectionFromProfile(entType, profile.Selection)
	if err != nil {
		return nil, fmt.Errorf("error creating selectors: %w", err)
	}

	return profSel, nil
}

func getEiwFromFile(ruletype *minderv1.RuleType, epath string) (*entities.EntityInfoWrapper, error) {
	entType := minderv1.EntityFromString(ruletype.Def.InEntity)
	ent, err := readEntityFromFile(epath, entType)
	if err != nil {
		return nil, fmt.Errorf("error reading entity from file: %w", err)
	}

	return &entities.EntityInfoWrapper{
		Entity:      ent,
		Type:        entType,
		ExecutionID: &uuid.Nil,
	}, nil
}

func runEvaluationForRules(
	cmd *cobra.Command,
	eng *rtengine.RuleTypeEngine,
	inf *entities.EntityInfoWrapper,
	provider provifv1.Provider,
	entitySelectors selectors.Selection,
	remediateStatus db.NullRemediationStatusTypes,
	remMetadata pqtype.NullRawMessage,
	frags []*minderv1.Profile_Rule,
	actionEngine *actions.RuleActionsEngine,
) error {
	for _, frag := range frags {
		val := eng.GetRuleInstanceValidator()
		err := val.ValidateRuleDefAgainstSchema(frag.Def.AsMap())
		if err != nil {
			return fmt.Errorf("error validating rule against schema: %w", err)
		}
		cmd.Printf("Profile valid according to the JSON schema!\n")

		if err := val.ValidateParamsAgainstSchema(frag.GetParams()); err != nil {
			return fmt.Errorf("error validating params against schema: %w", err)
		}

		rule := models.RuleFromPB(
			uuid.New(), // Actual rule type ID does not matter here
			frag,
		)

		// Create the eval status params
		evalStatus := &engif.EvalStatusParams{
			Rule: &rule,
			EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{
				RemStatus:   remediateStatus,
				RemMetadata: remMetadata,
			},
		}

		// Enable logging for the engine
		ctx := context.Background()
		logConfig := serverconfig.LoggingConfig{Level: cmd.Flag("log-level").Value.String()}
		ctx = logger.FromFlags(logConfig).WithContext(ctx)

		// Perform rule evaluation
		evalErr := selectAndEval(ctx, eng, provider, inf, evalStatus, entitySelectors)
		evalStatus.SetEvalErr(evalErr)

		// Perform the actions, if any
		evalStatus.SetActionsErr(ctx, actionEngine.DoActions(ctx, inf.Entity, evalStatus))

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

func selectAndEval(
	ctx context.Context,
	eng *rtengine.RuleTypeEngine,
	provider provifv1.Provider,
	inf *entities.EntityInfoWrapper,
	evalStatus *engif.EvalStatusParams,
	profileSelectors selectors.Selection,
) error {
	selEnt := provsel.EntityToSelectorEntity(ctx, provider, inf.Type, inf.Entity)
	if selEnt == nil {
		return fmt.Errorf("error converting entity to selector entity")
	}

	selected, err := profileSelectors.Select(selEnt)
	if err != nil {
		return fmt.Errorf("error selecting entity: %w", err)
	}

	var evalErr error
	if selected {
		evalErr = eng.Eval(ctx, inf, evalStatus)
	} else {
		evalErr = errors.NewErrEvaluationSkipped("entity not selected by selectors")
	}

	return evalErr
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
	case minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS, minderv1.Entity_ENTITY_RELEASE,
		minderv1.Entity_ENTITY_PIPELINE_RUN,
		minderv1.Entity_ENTITY_TASK_RUN, minderv1.Entity_ENTITY_BUILD:
		return nil, fmt.Errorf("entity type not yet supported")
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

func getProvider(pstr string, token string, providerConfigFile string) (provifv1.Provider, error) {
	cfgbytes, err := readProviderConfig(providerConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error reading provider config: %w", err)
	}

	switch pstr {
	case "github":
		client, err := clients.NewGitHubAppProvider(
			&minderv1.GitHubAppProviderConfig{},
			&serverconfig.ProviderConfig{
				GitHubApp: &serverconfig.GitHubAppConfig{AppName: "test"},
			},
			&ratecache.NoopRestClientCache{},
			credentials.NewGitHubTokenCredential(token),
			nil,
			clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
			false,
		)
		if err != nil {
			return nil, fmt.Errorf("error instantiating github provider: %w", err)
		}

		return client, nil
	case "dockerhub":
		// read provider config
		cfg, err := dockerhub.ParseV1Config(cfgbytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing dockerhub provider config: %w", err)
		}

		client, err := dockerhub.New(credentials.NewOAuth2TokenCredential(token), cfg)
		if err != nil {
			return nil, fmt.Errorf("error instantiating dockerhub provider: %w", err)
		}

		return client, nil
	default:
		return nil, fmt.Errorf("unknown or unsupported provider: %s", pstr)
	}
}

func readProviderConfig(fpath string) ([]byte, error) {
	if fpath == "" {
		return []byte{}, nil
	}

	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	w := &bytes.Buffer{}
	if err := jsonyaml.TranscodeYAMLToJSON(f, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	return w.Bytes(), nil
}

func actionOptFromString(s *string, defAction models.ActionOpt) models.ActionOpt {
	var actionOptMap = map[string]models.ActionOpt{
		"on":      models.ActionOptOn,
		"off":     models.ActionOptOff,
		"dry_run": models.ActionOptDryRun,
	}

	if s == nil {
		return defAction
	}

	if v, ok := actionOptMap[*s]; ok {
		return v
	}

	return models.ActionOptUnknown
}
