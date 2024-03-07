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
// Package rule provides the CLI subcommand for managing rules

// Package builtin provides the builtin ingestion engine
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-git/go-billy/v5"
	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stacklok/minder/pkg/rule_methods"
)

const (
	// BuiltinRuleDataIngestType is the type of the builtin rule data ingest engine
	BuiltinRuleDataIngestType = "builtin"
)

// BuiltinRuleDataIngest is the engine for a rule type that uses builtin methods
type BuiltinRuleDataIngest struct {
	builtinCfg  *pb.BuiltinType
	ruleMethods rule_methods.Methods
	method      string
}

// NewBuiltinRuleDataIngest creates a new builtin rule data ingest engine
func NewBuiltinRuleDataIngest(
	builtinCfg *pb.BuiltinType,
	_ *providers.ProviderBuilder,
) (*BuiltinRuleDataIngest, error) {
	return &BuiltinRuleDataIngest{
		builtinCfg:  builtinCfg,
		method:      builtinCfg.GetMethod(),
		ruleMethods: &rule_methods.RuleMethods{},
	}, nil
}

// FileContext returns a file context that an evaluator can use to do rule evaluation.
// the builtin engine does not support file context.
func (*BuiltinRuleDataIngest) FileContext() billy.Filesystem {
	return nil
}

// GetType returns the type of the builtin rule data ingest engine
func (*BuiltinRuleDataIngest) GetType() string {
	return BuiltinRuleDataIngestType
}

// GetConfig returns the config for the builtin rule data ingest engine
func (idi *BuiltinRuleDataIngest) GetConfig() protoreflect.ProtoMessage {
	return idi.builtinCfg
}

// Ingest calls the builtin method and populates the data to be returned
func (idi *BuiltinRuleDataIngest) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*engif.Result, error) {
	method, err := idi.ruleMethods.GetMethod(idi.method)
	if err != nil {
		return nil, fmt.Errorf("cannot get method: %w", err)
	}

	// Check if the method exists
	if !method.IsValid() {
		return nil, fmt.Errorf("rule method not found")
	}

	matches, err := entityMatchesParams(ctx, ent, params)
	if err != nil {
		return nil, fmt.Errorf("cannot check if entity matches params: %w", err)
	} else if !matches {
		return nil, evalerrors.ErrEvaluationSkipSilently
	}

	// call method
	// Call the method (empty parameter list)
	result := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(ent)})
	if len(result) != 2 {
		return nil, fmt.Errorf("rule method should return 3 values")
	}
	if !result[1].IsNil() {
		return nil, fmt.Errorf("error calling rule method")
	}
	if result[0].IsNil() {
		return nil, fmt.Errorf("error calling rule method")
	}
	methodResult := result[0].Interface().(json.RawMessage)
	var resultObj interface{}
	err = json.Unmarshal(methodResult, &resultObj)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal json: %w", err)
	}

	return &engif.Result{
		Object: resultObj,
	}, nil
}

func entityMatchesParams(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (bool, error) {
	// first convert to json string
	jsonStr, err := util.GetJsonFromProto(ent)
	if err != nil {
		return false, fmt.Errorf("cannot convert entity to json: %w", err)
	}
	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &jsonData)
	if err != nil {
		return false, fmt.Errorf("cannot unmarshal json: %w", err)
	}
	for key, val := range params {
		// if key does not start with dot add it
		if !strings.HasPrefix(key, ".") {
			key = "." + key
		}
		expectedVal, err := util.JQReadFrom[any](ctx, key, jsonData)
		if err != nil && !errors.Is(err, util.ErrNoValueFound) {
			return false, fmt.Errorf("cannot get values from data accessor: %w", err)
		}
		if !reflect.DeepEqual(expectedVal, val) {
			// just continue, this entity is not matching our parameters
			return false, nil
		}
	}
	return true, nil
}
