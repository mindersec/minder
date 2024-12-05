// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"
)

const (
	// IngestTypeDiff is the ingest type for a diff
	IngestTypeDiff = "diff"
)

const (
	// DiffTypeDep is the diff type for a dependency
	DiffTypeDep = "dep"

	// DiffTypeNewDeps returns scalibr dependency diffs
	DiffTypeNewDeps = "new-dep"

	// DiffTypeFull is the diff type for including all files from the PR diff
	DiffTypeFull = "full"
)

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

// WithDefaultShortFailureMessage sets the evaluation failure message if it is not set
func (r *RuleType) WithDefaultShortFailureMessage() *RuleType {
	if r == nil {
		return nil
	}

	if r.ShortFailureMessage == "" {
		r.ShortFailureMessage = fmt.Sprintf("Rule %s evaluation failed", r.Name)
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
func (s *RuleTypeReleasePhase) InitializedStringValue() (string, error) {
	return s.EnsureDefault().Enum().AsString()
}

// EnsureDefault ensures the rule type release phase has a default value
func (s *RuleTypeReleasePhase) EnsureDefault() *RuleTypeReleasePhase {
	if s == nil || *s == RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED {
		*s = RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_GA
	}
	return s
}

// MarshalJSON marshals the rule type release phase value to a JSON string
func (s *RuleTypeReleasePhase) MarshalJSON() ([]byte, error) {
	str, err := s.AsString()
	if err != nil {
		return nil, err
	}
	return json.Marshal(str)
}

// UnmarshalJSON unmarshalls the rule type release phase value from a JSON string
func (s *RuleTypeReleasePhase) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	return s.FromString(str)
}

// AsString returns a human-readable string for the rule type release phase value
func (s *RuleTypeReleasePhase) AsString() (string, error) {
	if s == nil {
		return "ga", nil
	}

	v := s.Descriptor().Values().ByNumber(s.Number())
	if v == nil {
		return "", fmt.Errorf("unknown rule type release phase value: %d", s)
	}
	extension := proto.GetExtension(v.Options(), E_Name)
	n, ok := extension.(string)
	if !ok {
		return "", fmt.Errorf("unknown rule type release phase value: %d", s)
	}

	return n, nil
}

// FromString sets the rule type release phase from a string
func (s *RuleTypeReleasePhase) FromString(str string) error {
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
			*s = RuleTypeReleasePhase(num)

			return nil
		}
	}

	return fmt.Errorf("unknown rule type release phase value: %s", str)
}
