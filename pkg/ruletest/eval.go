// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/datasources"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/engine/v1/rtengine"
	"github.com/mindersec/minder/pkg/fileconvert"
	tkv1 "github.com/mindersec/minder/pkg/testkit/v1"
)

func (tr *testCaseRunner) builtinEval(
	_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
) (starlark.Value, error) {
	var ruleName string
	var entityDict *starlark.Dict
	var profileDict *starlark.Dict
	var paramsDict *starlark.Dict
	var mockHttpDict *starlark.Dict
	var mockFSDict *starlark.Dict
	var datasourcesList *starlark.List
	var providerTraitsPresentList *starlark.List

	err := starlark.UnpackArgs("eval", args, kwargs,
		"rule", &ruleName, "entity?", &entityDict,
		"profile?", &profileDict, "params?", &paramsDict, "mock_http?", &mockHttpDict,
		"mock_fs?", &mockFSDict, "data_sources?", &datasourcesList,
		"provider_traits_present?", &providerTraitsPresentList)
	if err != nil {
		return nil, err
	}

	// A nil list (the argument was not passed) means the default: every
	// trait is present. An explicit (possibly empty) list means only the
	// listed traits are present.
	presentTraits, err := parseProviderTraitsList(providerTraitsPresentList)
	if err != nil {
		return nil, fmt.Errorf("invalid provider_traits_present argument: %w", err)
	}

	mockFSMap, err := parseMockFSDict(mockFSDict)
	if err != nil {
		return nil, err
	}

	rt, err := tr.lookupRuleType(ruleName)
	if err != nil {
		return nil, err
	}

	profileMap, err := dictToGoMap(profileDict)
	if err != nil {
		return nil, fmt.Errorf("invalid profile argument: %w", err)
	}

	paramsMap := make(map[string]any)
	if paramsDict != nil {
		paramsMap, err = dictToGoMap(paramsDict)
		if err != nil {
			return nil, fmt.Errorf("invalid params argument: %w", err)
		}
	}

	entityMap, err := dictToGoMap(entityDict)
	if err != nil {
		return nil, fmt.Errorf("invalid entity argument: %w", err)
	}

	entityProto, err := mapToProto(rt.Def.InEntity, entityMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity map to proto: %w", err)
	}

	mockHandler, err := buildMockHTTPHandler(mockHttpDict)
	if err != nil {
		return nil, fmt.Errorf("invalid mock_http configuration: %w", err)
	}

	ctx := context.Background()

	tkOpts := []tkv1.Option{tkv1.WithHandlerFunc(mockHandler.ServeHTTP)}
	if mockFSDict != nil {
		tkOpts = append(tkOpts, tkv1.WithGitFiles(mockFSMap))
	}
	if providerTraitsPresentList != nil {
		tkOpts = append(tkOpts, tkv1.WithCanImplement(func(trait minderv1.ProviderType) bool {
			return slices.Contains(presentTraits, trait)
		}))
	}
	tk := tkv1.NewTestKit(tkOpts...)

	dsRegistry, err := buildDataSourceRegistry(datasourcesList, tk, tr.baseDir)
	if err != nil {
		return nil, fmt.Errorf("invalid data_sources argument: %w", err)
	}

	rte, err := rtengine.NewRuleTypeEngine(ctx, rt, tk, eoptions.WithDataSources(dsRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rule type engine: %w", err)
	}

	// An unknown provider_traits entry is a bug in the rule type under
	// test (a typo, or a trait renamed since the rule type was written),
	// not something a test author can work around with
	// provider_traits_present. Fail the eval() call outright instead of
	// reporting a "skip" result, so it can't be mistaken for the rule
	// type simply not applying to this test's provider traits.
	if unknown := rte.UnknownProviderTraits(); len(unknown) > 0 {
		return nil, fmt.Errorf("rule %q declares unknown provider trait(s) %s, valid values are: %s",
			ruleName, strings.Join(unknown, ", "), strings.Join(minderv1.ValidProviderTraitNames(), ", "))
	}

	if !rte.SupportedByProvider() {
		return skippedResult("rule type requires a provider trait not present in this test"), nil
	}

	if tk.ShouldOverrideIngest() {
		rte.WithCustomIngester(tk)
	}

	res, err := rte.Eval(ctx, entityProto, profileMap, paramsMap, &stubResultSink{})

	return formatEvalResult(res, err), nil
}

// skippedResult builds an eval() result for a rule type that was not
// evaluated because the test provider does not implement one of its
// required provider_traits.
//
// This intentionally diverges from the production executor, which produces
// no eval status row at all for this case (see executor.evaluateRule) —
// zero footprint, not even a skip. A Starlark test still needs eval() to
// return *something* observable so a test author can assert on it, so the
// harness reports it as a "skip" result rather than reproducing the
// executor's silent no-op.
func skippedResult(msg string) *starlark.Dict {
	result := starlark.NewDict(2)
	_ = result.SetKey(starlark.String("status"), starlark.String("skip"))
	_ = result.SetKey(starlark.String("message"), starlark.String(msg))
	return result
}

type stubResultSink struct{}

func (*stubResultSink) SetIngestResult(*interfaces.Ingested) {}

func formatEvalResult(res *interfaces.EvaluationResult, evalErr error) *starlark.Dict {
	result := starlark.NewDict(2)
	status, msg := "", ""

	switch {
	case evalErr == nil:
		status = "pass"
	case errors.Is(evalErr, interfaces.ErrEvaluationFailed):
		status = "fail"
		msg = evalErr.Error()
		var details interfaces.EvalError
		if errors.As(evalErr, &details) {
			msg = fmt.Sprintf("%s: %s", msg, details.Details())
		}
	case errors.Is(evalErr, interfaces.ErrEvaluationSkipped):
		status = "skip"
		msg = evalErr.Error()
	default:
		status = "error"
		msg = evalErr.Error()
	}

	_ = result.SetKey(starlark.String("status"), starlark.String(status))
	_ = result.SetKey(starlark.String("message"), starlark.String(msg))

	if res != nil && res.Output != nil {
		if slVal, err := goToStarlarkValue(res.Output); err == nil {
			_ = result.SetKey(starlark.String("output"), slVal)
		}
	}

	return result
}

func parseMockFSDict(mockFSDict *starlark.Dict) (map[string]string, error) {
	mockFSMap := make(map[string]string)
	if mockFSDict != nil {
		for _, item := range mockFSDict.Items() {
			k, v := item[0], item[1]
			ks, ok1 := k.(starlark.String)
			vs, ok2 := v.(starlark.String)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("mock_fs keys and values must be strings")
			}
			mockFSMap[string(ks)] = string(vs)
		}
	}
	return mockFSMap, nil
}

// parseProviderTraitsList converts a Starlark list of provider trait names
// (e.g. "github"), using the same short trait names as a rule type's
// provider_traits field, into their protobuf enum values, for use with
// tkv1.WithCanImplement. A nil list returns a nil slice.
func parseProviderTraitsList(list *starlark.List) ([]minderv1.ProviderType, error) {
	if list == nil {
		return nil, nil
	}

	traits := make([]minderv1.ProviderType, 0, list.Len())
	for val := range list.Elements() {
		s, ok := val.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("provider_traits_present must be a list of strings")
		}
		trait, ok := minderv1.ProviderTypeFromString(string(s))
		if !ok {
			return nil, fmt.Errorf("unknown provider trait %q, valid values are: %s",
				string(s), strings.Join(minderv1.ValidProviderTraitNames(), ", "))
		}
		traits = append(traits, trait)
	}
	return traits, nil
}

func buildDataSourceRegistry(
	datasourcesList *starlark.List, tk *tkv1.TestKit, baseDir string,
) (*v1datasources.DataSourceRegistry, error) {
	registry := v1datasources.NewDataSourceRegistry()
	if datasourcesList == nil {
		return registry, nil
	}

	for val := range datasourcesList.Elements() {
		pathStr, ok := val.(starlark.String)
		if !ok {
			return nil, fmt.Errorf("data_sources must be a list of strings")
		}

		path := string(pathStr)
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}

		ds, err := loadDataSource(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load data source %q: %w", path, err)
		}

		// Build the datasource, passing the TestKit as the HTTP RoundTripper
		builtDS, err := datasources.BuildFromProtobuf(ds, tk, v1datasources.WithTestOnlyTransport(tk))
		if err != nil {
			return nil, fmt.Errorf("failed to build data source %q: %w", path, err)
		}

		if err := registry.RegisterDataSource(ds.GetName(), builtDS); err != nil {
			return nil, fmt.Errorf("failed to register data source %q: %w", ds.GetName(), err)
		}
	}

	return registry, nil
}

func loadDataSource(path string) (*minderv1.DataSource, error) {
	decoder, closer := fileconvert.DecoderForFile(path)
	if decoder == nil {
		return nil, fmt.Errorf("error opening file: %s", path)
	}
	defer func() {
		_ = closer.Close()
	}()
	return fileconvert.ReadResourceTyped[*minderv1.DataSource](decoder)
}

func (tr *testCaseRunner) lookupRuleType(ruleName string) (*minderv1.RuleType, error) {
	if tr.ruleTypes != nil {
		if ruleType := tr.ruleTypes[ruleName]; ruleType != nil {
			return ruleType, nil
		}
	}
	return nil, fmt.Errorf("rule %q not found; make sure the rule type YAML is in the same directory as the test file", ruleName)
}

//nolint:gocyclo // this is a simple switch over many entity types
func mapToProto(entityType string, entityMap map[string]any) (proto.Message, error) {
	b, err := json.Marshal(entityMap)
	if err != nil {
		return nil, err
	}

	unmarshaller := protojson.UnmarshalOptions{DiscardUnknown: true}
	entEnum := minderv1.EntityFromString(entityType)

	var msg proto.Message

	switch entEnum {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		msg = &minderv1.Repository{}
	case minderv1.Entity_ENTITY_ARTIFACTS:
		msg = &minderv1.Artifact{}
	case minderv1.Entity_ENTITY_RELEASE:
		msg = &minderv1.Release{}
	case minderv1.Entity_ENTITY_PIPELINE_RUN:
		msg = &minderv1.PipelineRun{}
	case minderv1.Entity_ENTITY_TASK_RUN:
		msg = &minderv1.TaskRun{}
	case minderv1.Entity_ENTITY_BUILD:
		msg = &minderv1.Build{}
	case minderv1.Entity_ENTITY_UNSPECIFIED,
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS,
		minderv1.Entity_ENTITY_PULL_REQUESTS:
		fallthrough
	default:
		return nil, fmt.Errorf("unsupported entity type for mapping to proto: %s", entityType)
	}

	if err := unmarshaller.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	return msg, nil
}
