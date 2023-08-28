//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util provides helper functions for the mediator CLI.
package util

import (
	"context"
	"fmt"

	"github.com/itchyny/gojq"
)

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
			return nil, fmt.Errorf("error processing JQ statement: %w", err)
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
