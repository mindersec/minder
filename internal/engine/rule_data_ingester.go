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

package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// RuleDataIngest is the interface for rule data ingest
// It allows for different mechanisms for ingesting data
// in order to evaluate a rule.
type RuleDataIngest interface {
	Eval(ctx context.Context, ent any, pol any) error
}

// NewRuleDataIngest creates a new rule data ingest based no the given rule
// type definition.
func NewRuleDataIngest(rt *pb.RuleType, cli ghclient.RestAPI) (RuleDataIngest, error) {
	// TODO: make this more generic and/or use constants
	switch rt.Def.DataEval.Type {
	case "rest":
		if rt.Def.DataEval.GetRest() == nil {
			return nil, fmt.Errorf("rule type engine missing rest configuration")
		}

		eval := rt.Def.GetDataEval()
		return NewRestRuleDataIngest(eval, eval.GetRest(), cli)
	default:
		return nil, fmt.Errorf("rule type engine only supports REST data ingest")
	}
}

// RestRuleDataIngest is the engine for a rule type that uses REST data ingest
type RestRuleDataIngest struct {
	cfg              *pb.RuleType_Definition_DataEval
	restCfg          *pb.RuleType_Definition_DataEval_RestType
	cli              ghclient.RestAPI
	endpointTemplate *template.Template
	method           string
}

// NewRestRuleDataIngest creates a new REST rule data ingest engine
func NewRestRuleDataIngest(
	cfg *pb.RuleType_Definition_DataEval,
	restCfg *pb.RuleType_Definition_DataEval_RestType,
	cli ghclient.RestAPI,
) (*RestRuleDataIngest, error) {
	tmpl := template.New("path")
	tmpl, err := tmpl.Parse(restCfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot parse endpoint template: %w", err)
	}

	method := strings.ToUpper(restCfg.Method)
	if len(method) == 0 {
		method = http.MethodGet
	}

	// TODO: parse key-type

	return &RestRuleDataIngest{
		cfg:              cfg,
		restCfg:          restCfg,
		cli:              cli,
		endpointTemplate: tmpl,
		method:           method,
	}, nil
}

// Eval evaluates the rule type against the given entity and policy
func (rdi *RestRuleDataIngest) Eval(ctx context.Context, ent any, pol any) error {
	endpoint := new(bytes.Buffer)
	if err := rdi.endpointTemplate.Execute(endpoint, ent); err != nil {
		return fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	req, err := rdi.cli.NewRequest(rdi.method, endpoint.String(), rdi.restCfg.Body)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	bodyBuf := new(bytes.Buffer)
	_, err = rdi.cli.Do(ctx, req, bodyBuf)
	if err != nil {
		return fmt.Errorf("cannot make request: %w", err)
	}

	var data any
	data = bodyBuf

	if rdi.restCfg.Parse == "json" {
		var jsonData any
		dec := json.NewDecoder(bodyBuf)
		if err := dec.Decode(&jsonData); err != nil {
			return fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	}

	// TODO: Handle formats other than `jq`

	for key, val := range rdi.cfg.Data {
		policyVal, err := JQGetValuesFromAccessor(ctx, key, pol)
		if err != nil {
			return fmt.Errorf("cannot get values from policy accessor: %w", err)
		}

		dataVal, err := JQGetValuesFromAccessor(ctx, val.Def, data)
		if err != nil {
			return fmt.Errorf("cannot get values from data accessor: %w", err)
		}

		// Deep compare
		if !reflect.DeepEqual(policyVal, dataVal) {
			return fmt.Errorf("data does not match policy: got %v, want %v", dataVal, policyVal)
		}
	}

	return nil
}

// JQGetValuesFromAccessor gets the values from the given accessor
// the path is the accessor path in jq format.
// the obj is the object to be evaluated using the accessor.
func JQGetValuesFromAccessor(ctx context.Context, path string, obj any) (any, error) {
	out := []any{}
	accessor, err := gojq.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("data parse: cannot parse key: %w", err)
	}

	iter := accessor.RunWithContext(ctx, obj)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			log.Fatalln(err)
		}

		out = append(out, v)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no values found")
	}

	if len(out) == 1 {
		return out[0], nil
	}

	return out, nil
}
