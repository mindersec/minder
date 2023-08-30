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

// Package builtin provides the builtin ingestion engine
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/rule_methods"
)

const (
	// BuiltinRuleDataIngestType is the type of the builtin rule data ingest engine
	BuiltinRuleDataIngestType = "builtin"
)

// BuiltinRuleDataIngest is the engine for a rule type that uses builtin methods
type BuiltinRuleDataIngest struct {
	builtinCfg  *pb.BuiltinType
	method      string
	accessToken string
}

// NewBuiltinRuleDataIngest creates a new builtin rule data ingest engine
func NewBuiltinRuleDataIngest(
	builtinCfg *pb.BuiltinType,
	access_token string,
) (*BuiltinRuleDataIngest, error) {
	return &BuiltinRuleDataIngest{
		builtinCfg:  builtinCfg,
		accessToken: access_token,
		method:      builtinCfg.GetMethod(),
	}, nil
}

// Ingest calls the builtin method and populates the data to be returned
func (idi *BuiltinRuleDataIngest) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (any, error) {
	// call internal method stored in pkg and method
	rm := rule_methods.RuleMethods{}
	value := reflect.ValueOf(rm)
	method := value.MethodByName(idi.method)

	// Check if the method exists
	if !method.IsValid() {
		return nil, fmt.Errorf("rule method not found")
	}

	matches, err := entityMatchesParams(ctx, ent, params)
	if err != nil {
		return nil, fmt.Errorf("cannot check if entity matches params: %w", err)
	}

	// TODO: this should be a warning
	if !matches {
		log.Printf("entity not matching parameters, skipping")
		return nil, nil
	}

	// call method
	// Call the method (empty parameter list)
	result := method.Call([]reflect.Value{reflect.ValueOf(ctx),
		reflect.ValueOf(idi.accessToken), reflect.ValueOf(ent)})
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

	return resultObj, nil
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
