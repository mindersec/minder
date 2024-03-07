// Copyright 2023 Stacklok, Inc
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

// Package rule_methods provides the methods that are used by the rules
package rule_methods

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Methods is the interface that is used to get the method by name
type Methods interface {
	GetMethod(string) (reflect.Value, error)
}

// RuleMethods is the struct that contains the methods that are used by the rules
type RuleMethods struct{}

// GetMethod gets the method by name from the RuleMethods struct
func (r *RuleMethods) GetMethod(mName string) (reflect.Value, error) {
	value := reflect.ValueOf(r)
	method := value.MethodByName(mName)

	// Check if the method exists
	if !method.IsValid() {
		return reflect.Value{}, fmt.Errorf("rule method not found")
	}

	return method, nil
}

// Passthrough is a method that passes the entity through, just marshalling it
func (_ *RuleMethods) Passthrough(_ context.Context, ent protoreflect.ProtoMessage) (json.RawMessage, error) {
	return protojson.Marshal(ent)
}
