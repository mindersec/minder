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

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/stacklok/minder/internal/util/jsonyaml"
)

const (
	// IngestTypeDiff is the ingest type for a diff
	IngestTypeDiff = "diff"
)

const (
	// DiffTypeDep is the diff type for a dependency
	DiffTypeDep = "dep"

	// DiffTypeFull is the diff type for including all files from the PR diff
	DiffTypeFull = "full"
)

// ParseRuleType parses a rule type from a reader
func ParseRuleType(r io.Reader) (*RuleType, error) {
	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := jsonyaml.TranscodeYAMLToJSON(r, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	rt := &RuleType{}
	if err := json.NewDecoder(w).Decode(rt); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return rt, nil
}

// GetContext returns the context from the nested RuleType
func (r *CreateRuleTypeRequest) GetContext() *Context {
	if r != nil && r.RuleType != nil {
		return r.RuleType.Context
	}
	return nil
}

// GetContext returns the context from the nested RuleType
func (r *UpdateRuleTypeRequest) GetContext() *Context {
	if r != nil && r.RuleType != nil {
		return r.RuleType.Context
	}
	return nil
}
