// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fileconvert provides functions for marshalling Minder proto objects
// to and from on-disk formats like YAML.
package fileconvert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Encoder is a superset of the yaml.Encoder and json.Encoder interfaces.
type Encoder interface {
	Encode(v any) error
}

// Decoder is a superset of the yaml.Decoder and json.Decoder interfaces.
type Decoder interface {
	Decode(v any) error
}

var (
	_ Decoder = (*yaml.Decoder)(nil)
	_ Decoder = (*json.Decoder)(nil)
)

// DecoderForFile returns a Decoder for the file at the specified path,
// or nil if the file is not of the appropriate type.
func DecoderForFile(path string) (Decoder, io.Closer) {
	path = filepath.Clean(path)
	ext := filepath.Ext(path)
	var builder func(io.Reader) Decoder
	// we return functions here so that we can early-exit without opening the file if the extension is unmatched.
	switch ext {
	case ".json":
		builder = func(r io.Reader) Decoder { return json.NewDecoder(r) }
	case ".yaml", ".yml":
		builder = func(r io.Reader) Decoder { return yaml.NewDecoder(r) }
	case ".rego":
		builder = func(r io.Reader) Decoder { return &regoDecoder{filename: path, file: r} }
	default:
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	return builder(file), file
}

// WriteResource outputs a Minder proto resource to an existing Encoder.
// Only known resource types are supported; others will return an error.
//
// This uses the proto JSON serialization to convert the resource to an
// object, as a naive transformation from proto to YAML does not nicely
// deal with Structs and other proto features.  In the case of JSON, this
// *does* mean that we encode to JSON twice, but this is not expected to
// be a high-performance path.
func WriteResource(output Encoder, resource minderv1.ResourceMeta) error {
	var jsonData []byte
	var err error
	marshaller := protojson.MarshalOptions{UseProtoNames: true}
	switch r := resource.(type) {
	case *minderv1.Profile:
		r.Type = string(minderv1.ProfileResource)
		r.Version = "v1"
	case *minderv1.RuleType:
		r.Type = string(minderv1.RuleTypeResource)
		r.Version = "v1"
		jsonData, err = marshalRuleType(marshaller, r)
	case *minderv1.DataSource:
		r.Type = string(minderv1.DataSourceResource)
		r.Version = "v1"
	default:
		return fmt.Errorf("unknown resource type: %T", resource)
	}
	if jsonData == nil && err == nil {
		jsonData, err = marshaller.Marshal(resource)
	}
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}
	var genericObject any

	if err = json.Unmarshal(jsonData, &genericObject); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return output.Encode(genericObject)
}

// ReadResource reads a single resource from the specified Decoder.
// Only known resource types are supported; others will return an error.
// Given that multiple resources can be stored in a single file, the
// the input may not be fully consumed.  ReadResource will always return
// either a (pointer to) resource or an error; it will never return a nil
// resource.
//
// The resource is returned as a proto.Message, which can be type-asserted
// to the appropriate type (see also ReadResourceTyped).  Like
// WriteResource, this uses proto JSON serialization in addition to naive
// decoding to handle proto-specific encoding features.
func ReadResource(input Decoder) (minderv1.ResourceMeta, error) {
	// All our types are JSON object
	var genericObject map[string]any
	if err := input.Decode(&genericObject); err != nil {
		return nil, fmt.Errorf("error decoding: %w", err)
	}
	objectType, ok := genericObject["type"].(string)
	if !ok {
		return nil, fmt.Errorf("resource type not found")
	}
	objectVersion, ok := genericObject["version"].(string)
	if !ok || objectVersion != "v1" {
		return nil, fmt.Errorf("unsupported resource version: %s", objectVersion)
	}
	jsonData, err := json.Marshal(genericObject)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}
	switch objectType {
	case string(minderv1.ProfileResource):
		var profile minderv1.Profile
		if err := json.Unmarshal(jsonData, &profile); err != nil {
			return nil, fmt.Errorf("error unmarshaling profile: %w", err)
		}
		if err := profile.Validate(); err != nil {
			return nil, err
		}
		return &profile, nil
	case string(minderv1.RuleTypeResource):
		var ruleType minderv1.RuleType
		if err := unmarshalRuleType(jsonData, &ruleType); err != nil {
			return nil, fmt.Errorf("error unmarshaling rule type: %w", err)
		}
		if err := ruleType.Validate(); err != nil {
			return nil, err
		}
		return &ruleType, nil
	case string(minderv1.DataSourceResource):
		var dataSource minderv1.DataSource
		if err := protojson.Unmarshal(jsonData, &dataSource); err != nil {
			return nil, fmt.Errorf("error unmarshaling data source: %w", err)
		}
		if err := dataSource.Validate(); err != nil {
			return nil, err
		}
		return &dataSource, nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", objectType)
	}
}

// ReadResourceTyped reads a single resource from the specified Decoder and
// returns it as the specified subtype of proto.Messsage.  Given that T is
// pointer, this will always return either a non-nil pointer or an error.
//
// This is a convenience wrapper around ReadResource that handles the type
// assertion for you.
func ReadResourceTyped[T proto.Message](input Decoder) (T, error) {
	var zero T
	r, err := ReadResource(input)
	if err != nil {
		return zero, err
	}
	typed, ok := r.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected resource type: %T", r)
	}
	return typed, nil
}

func marshalRuleType(marshaller protojson.MarshalOptions, r *minderv1.RuleType) ([]byte, error) {
	protoJSON, err := marshaller.Marshal(r)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err := json.Unmarshal(protoJSON, &obj); err != nil {
		return nil, err
	}

	if r.GetSeverity() != nil {
		if severity, ok := obj["severity"].(map[string]any); ok {
			if _, ok := severity["value"]; ok {
				severity["value"] = r.GetSeverity().GetValue().Enum().AsString()
			}
		}
	}
	if _, ok := obj["release_phase"]; ok {
		releasePhase, err := r.GetReleasePhase().Enum().AsString()
		if err != nil {
			return nil, fmt.Errorf("error converting release phase: %w", err)
		}
		obj["release_phase"] = releasePhase
	}

	return json.Marshal(obj)
}

func unmarshalRuleType(jsonData []byte, ruleType *minderv1.RuleType) error {
	var obj map[string]any
	if err := json.Unmarshal(jsonData, &obj); err != nil {
		return err
	}

	for _, key := range []string{"release_phase", "releasePhase"} {
		if phase, ok := obj[key].(string); ok {
			if _, isProtoName := minderv1.RuleTypeReleasePhase_value[phase]; !isProtoName {
				var rp minderv1.RuleTypeReleasePhase
				if err := rp.FromString(phase); err != nil {
					return err
				}
				obj[key] = rp.String()
			}
		}
	}
	if severity, ok := obj["severity"].(map[string]any); ok {
		if value, ok := severity["value"].(string); ok {
			if _, isProtoName := minderv1.Severity_Value_value[value]; !isProtoName {
				var sv minderv1.Severity_Value
				if err := sv.FromString(value); err != nil {
					return err
				}
				severity["value"] = sv.String()
			}
		}
	}

	protoJSON, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return protojson.Unmarshal(protoJSON, ruleType)
}
