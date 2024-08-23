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

	"google.golang.org/protobuf/proto"

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

// WithDefaultDisplayName sets the display name if it is not set
func (r *RuleType) WithDefaultDisplayName() *RuleType {
	if r == nil {
		return nil
	}

	if r.DisplayName == "" {
		r.DisplayName = r.Name
	}

	return r
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

// InitializedStringValue returns the string value of the severity
// with initialization done.
func (s *Severity) InitializedStringValue() string {
	return s.EnsureDefault().GetValue().Enum().AsString()
}

// EnsureDefault ensures the rule type has a default value
func (s *Severity) EnsureDefault() *Severity {
	if s == nil {
		s = &Severity{}
	}

	if s.Value == Severity_VALUE_UNSPECIFIED {
		s.Value = Severity_VALUE_UNKNOWN
	}

	return s
}

// AsString returns a human-readable string for the severity value
func (s *Severity_Value) AsString() string {
	if s == nil {
		return "unknown"
	}

	v := s.Descriptor().Values().ByNumber(s.Number())
	if v == nil {
		return ""
	}
	extension := proto.GetExtension(v.Options(), E_Name)
	n, ok := extension.(string)
	if !ok {
		return ""
	}

	return n
}

// FromString sets the severity value from a string
func (s *Severity_Value) FromString(str string) error {
	vals := s.Descriptor().Values()
	for i := 0; i < vals.Len(); i++ {
		v := vals.Get(i)
		extension := proto.GetExtension(v.Options(), E_Name)
		n, ok := extension.(string)
		if !ok {
			continue
		}

		if n == str {
			num := v.Number()
			*s = Severity_Value(num)

			return nil
		}
	}

	return fmt.Errorf("unknown severity value: %s", str)
}

// MarshalJSON marshals the severity value to a JSON string
func (s *Severity_Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.AsString())
}

// UnmarshalJSON unmarshalls the severity value from a JSON string
func (s *Severity_Value) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	return s.FromString(str)
}

// InitializedStringValue returns the string value of the severity
// with initialization done.
func (s *RuleTypeState) InitializedStringValue() (string, error) {
	return s.EnsureDefault().Enum().AsString()
}

// EnsureDefault ensures the rule type has a default value
func (s *RuleTypeState) EnsureDefault() *RuleTypeState {
	if s == nil || *s == RuleTypeState_RULE_TYPE_STATE_UNSPECIFIED {
		*s = RuleTypeState_RULE_TYPE_STATE_GA
	}
	return s
}

// MarshalJSON marshals the rule type state value to a JSON string
func (s *RuleTypeState) MarshalJSON() ([]byte, error) {
	str, err := s.AsString()
	if err != nil {
		return nil, err
	}
	return json.Marshal(str)
}

// UnmarshalJSON unmarshalls the rule type state value from a JSON string
func (s *RuleTypeState) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	return s.FromString(str)
}

// AsString returns a human-readable string for the rule type state value
func (s *RuleTypeState) AsString() (string, error) {
	if s == nil {
		return "ga", nil
	}

	v := s.Descriptor().Values().ByNumber(s.Number())
	if v == nil {
		return "", fmt.Errorf("unknown rule type state value: %d", s)
	}
	extension := proto.GetExtension(v.Options(), E_Name)
	n, ok := extension.(string)
	if !ok {
		return "", fmt.Errorf("unknown rule type state value: %d", s)
	}

	return n, nil
}

// FromString sets the rule type state from a string
func (s *RuleTypeState) FromString(str string) error {
	vals := s.Descriptor().Values()
	for i := 0; i < vals.Len(); i++ {
		v := vals.Get(i)
		extension := proto.GetExtension(v.Options(), E_Name)
		n, ok := extension.(string)
		if !ok {
			continue
		}

		if n == str {
			num := v.Number()
			*s = RuleTypeState(num)

			return nil
		}
	}

	return fmt.Errorf("unknown rule type state value: %s", str)
}
