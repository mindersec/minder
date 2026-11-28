// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type regoDecoder struct {
	filename string
	file     io.Reader
}

var _ Decoder = (*regoDecoder)(nil)

// ErrNoRegoMetadata is returned for a Rego file without a METADATA block, such as
// a test or helper policy, which therefore does not describe a rule type.
var ErrNoRegoMetadata = errors.New("could not find metadata in Rego file")

// Decode extracts a Minder RuleType from a rego rule file.
// The additional RuleType information is encoded as YAML comments at the
// beginning of the file, following the METADATA format:
//
// https://www.openpolicyagent.org/docs/policy-language#metadata
func (r *regoDecoder) Decode(v any) error {
	ruleTypePtr, ok := v.(*map[string]any)
	if !ok || ruleTypePtr == nil {
		return fmt.Errorf("unexpected type: %T", v)
	}
	ruleType := map[string]any{}

	var contents bytes.Buffer
	n, err := io.Copy(&contents, r.file)
	if err != nil {
		return err
	}
	if n == 0 {
		return io.EOF
	}

	err = extractMetadata(contents.Bytes(), ruleType)
	if err != nil {
		return err
	}
	for _, k := range opaAnnotationKeys {
		delete(ruleType, k)
	}

	// The OPA metadata spec says that custom fields should be under the "custom" key
	// We also accept them under the top-level object for convenience, despite possible
	// future conflicts.
	if customMap, ok := ruleType["custom"].(map[string]any); ok {
		for k, v := range customMap {
			ruleType[k] = v
		}
		delete(ruleType, "custom")
	}

	ruleType["type"] = string(minderv1.RuleTypeResource)
	ruleType["version"] = "v1"
	if _, ok := ruleType["name"]; !ok {
		name := filepath.Base(r.filename)
		name = name[:len(name)-len(filepath.Ext(name))]
		ruleType["name"] = name
	}
	ruleType["display_name"] = cmp.Or(ruleType["display_name"], ruleType["title"])
	delete(ruleType, "title")
	// the "description" key already matches

	defMap, err := ensureEntry(ruleType, "def", map[string]any{})
	if err != nil {
		return err
	}

	// Rules must have a schema, per validation, but an empty schema is fine.
	// For convenience, allow omitting this in the metadata.
	_, _ = ensureEntry(defMap, "rule_schema", map[string]any{})

	evalMap, err := ensureEntry(defMap, "eval", map[string]any{})
	if err != nil {
		return err
	}
	evalMap["type"] = "rego"
	regoMap, err := ensureEntry(evalMap, "rego", map[string]any{})
	if err != nil {
		return err
	}
	regoMap["type"] = cmp.Or(regoMap["type"], "deny-by-default")
	// Yes, we have a "def" inside another "def".
	regoMap["def"] = contents.String()

	// Atomically assign once there are no errors
	*ruleTypePtr = ruleType

	return nil
}

// RuleType fields that standard OPA annotations do not have.
var ruleTypeOnlyKeys = []string{"def", "guidance", "severity", "release_phase", "short_failure_message"}

// HasRuleTypeMetadata reports whether the METADATA block of a Rego file sets any
// field specific to rule types, either at the top level or under "custom". A
// block without them annotates an ordinary policy. It returns an error if there
// is no METADATA block or it is not valid YAML.
func HasRuleTypeMetadata(contents []byte) (bool, error) {
	metadata := map[string]any{}
	if err := extractMetadata(contents, metadata); err != nil {
		return false, err
	}
	custom, _ := metadata["custom"].(map[string]any)
	for _, k := range ruleTypeOnlyKeys {
		_, atTop := metadata[k]
		_, inCustom := custom[k]
		if atTop || inCustom {
			return true, nil
		}
	}
	return false, nil
}

// Standard OPA annotations that have no RuleType equivalent, dropped so strict
// unmarshalling does not reject them.
var opaAnnotationKeys = []string{
	"scope", "authors", "organizations", "related_resources", "schemas", "entrypoint",
}

// metadataExtractor extracts the YAML document
var (
	metadataExtractor   = regexp.MustCompile("(?m)^# +METADATA *\r?\n((?:#(?: [^\r\n]*)?\r?\n)+)")
	removeCommentPrefix = regexp.MustCompile("(?m)^# ?") // default greedy
)

// OPA uses YAML metadata inside a specially-headered comment.  Extracting
// the metadata requires finding the comment block, then stripping the comment
// prefixes from each line.
func extractMetadata(contents []byte, metadata map[string]any) error {
	matches := metadataExtractor.FindSubmatch(contents)
	if len(matches) == 0 {
		return ErrNoRegoMetadata
	}
	commented := matches[1]

	uncommented := removeCommentPrefix.ReplaceAll(commented, []byte(""))

	return yaml.Unmarshal(uncommented, &metadata)
}

// ensureEntry simplifies the process of traversing and fetching from JSON-object type maps.
func ensureEntry[T any](in map[string]any, key string, def T) (T, error) {
	if _, ok := in[key]; !ok {
		in[key] = def
	}
	if ret, ok := in[key].(T); ok {
		return ret, nil
	}
	return def, fmt.Errorf("unexpected %q tuple: %T", key, in[key])
}
