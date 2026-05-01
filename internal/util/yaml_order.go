// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// Standard Minder resource proto full names — used to dispatch ordered
// YAML marshaling without importing the v1 package (which would create
// an import cycle since v1 already imports internal/util).
const (
	profileFullName    = "minder.v1.Profile"
	ruleTypeFullName   = "minder.v1.RuleType"
	dataSourceFullName = "minder.v1.DataSource"
)

// orderedProfile mirrors minderv1.Profile with fields in the standard
// Minder field order: version, type, name, context, … deep structure last.
// camelCase yaml tags match the protojson default output used by GetYamlFromProto.
type orderedProfile struct {
	Version     interface{} `yaml:"version,omitempty"`
	Type        interface{} `yaml:"type,omitempty"`
	Name        interface{} `yaml:"name,omitempty"`
	Context     interface{} `yaml:"context,omitempty"`
	Id          interface{} `yaml:"id,omitempty"`
	DisplayName interface{} `yaml:"displayName,omitempty"`
	Labels      interface{} `yaml:"labels,omitempty"`
	Remediate   interface{} `yaml:"remediate,omitempty"`
	Alert       interface{} `yaml:"alert,omitempty"`
	// entity rule lists
	Repository       interface{} `yaml:"repository,omitempty"`
	BuildEnvironment interface{} `yaml:"buildEnvironment,omitempty"`
	Artifact         interface{} `yaml:"artifact,omitempty"`
	PullRequest      interface{} `yaml:"pullRequest,omitempty"`
	Release          interface{} `yaml:"release,omitempty"`
	PipelineRun      interface{} `yaml:"pipelineRun,omitempty"`
	TaskRun          interface{} `yaml:"taskRun,omitempty"`
	Build            interface{} `yaml:"build,omitempty"`
	Selection        interface{} `yaml:"selection,omitempty"`
}

// orderedRuleType mirrors minderv1.RuleType with fields in the standard
// Minder field order. camelCase yaml tags match protojson default output.
type orderedRuleType struct {
	Version             interface{} `yaml:"version,omitempty"`
	Type                interface{} `yaml:"type,omitempty"`
	Name                interface{} `yaml:"name,omitempty"`
	Context             interface{} `yaml:"context,omitempty"`
	Id                  interface{} `yaml:"id,omitempty"`
	DisplayName         interface{} `yaml:"displayName,omitempty"`
	ShortFailureMessage interface{} `yaml:"shortFailureMessage,omitempty"`
	Description         interface{} `yaml:"description,omitempty"`
	Guidance            interface{} `yaml:"guidance,omitempty"`
	Severity            interface{} `yaml:"severity,omitempty"`
	ReleasePhase        interface{} `yaml:"releasePhase,omitempty"`
	Def                 interface{} `yaml:"def,omitempty"`
}

// orderedDataSource mirrors minderv1.DataSource with fields in the standard
// Minder field order. camelCase yaml tags match protojson default output.
type orderedDataSource struct {
	Version    interface{} `yaml:"version,omitempty"`
	Type       interface{} `yaml:"type,omitempty"`
	Name       interface{} `yaml:"name,omitempty"`
	Context    interface{} `yaml:"context,omitempty"`
	Id         interface{} `yaml:"id,omitempty"`
	Rest       interface{} `yaml:"rest,omitempty"`
	Structured interface{} `yaml:"structured,omitempty"`
}

// GetOrderedYamlFromProto serializes a proto.Message to YAML with fields in
// the standard Minder order (version, type, name, context, …) for the three
// resource types (Profile, RuleType, DataSource). For all other message types
// it falls back to the default (alphabetical) ordering.
func GetOrderedYamlFromProto(msg proto.Message) (string, error) {
	if msg == nil {
		return "{}\n", nil
	}

	m := getProtoMarshalOptions()
	jsonBytes, err := m.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshaling proto to JSON: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return "", fmt.Errorf("unmarshaling JSON: %w", err)
	}

	protoName := string(msg.ProtoReflect().Descriptor().FullName())

	var ordered interface{}
	switch protoName {
	case profileFullName:
		ordered = mapToOrderedProfile(raw)
	case ruleTypeFullName:
		ordered = mapToOrderedRuleType(raw)
	case dataSourceFullName:
		ordered = mapToOrderedDataSource(raw)
	default:
		// Fall back to alphabetical (previous behaviour).
		node, err := mapToSortedYAMLNode(raw)
		if err != nil {
			return "", err
		}
		out, err := yaml.Marshal(node)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	out, err := yaml.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("marshaling ordered struct to YAML: %w", err)
	}
	return string(out), nil
}

// mapToOrderedProfile fills an orderedProfile from the JSON-decoded map.
func mapToOrderedProfile(m map[string]interface{}) orderedProfile {
	return orderedProfile{
		Version:          m["version"],
		Type:             m["type"],
		Name:             m["name"],
		Context:          m["context"],
		Id:               m["id"],
		DisplayName:      m["displayName"],
		Labels:           m["labels"],
		Remediate:        m["remediate"],
		Alert:            m["alert"],
		Repository:       m["repository"],
		BuildEnvironment: m["buildEnvironment"],
		Artifact:         m["artifact"],
		PullRequest:      m["pullRequest"],
		Release:          m["release"],
		PipelineRun:      m["pipelineRun"],
		TaskRun:          m["taskRun"],
		Build:            m["build"],
		Selection:        m["selection"],
	}
}

// mapToOrderedRuleType fills an orderedRuleType from the JSON-decoded map.
func mapToOrderedRuleType(m map[string]interface{}) orderedRuleType {
	return orderedRuleType{
		Version:             m["version"],
		Type:                m["type"],
		Name:                m["name"],
		Context:             m["context"],
		Id:                  m["id"],
		DisplayName:         m["displayName"],
		ShortFailureMessage: m["shortFailureMessage"],
		Description:         m["description"],
		Guidance:            m["guidance"],
		Severity:            m["severity"],
		ReleasePhase:        m["releasePhase"],
		Def:                 m["def"],
	}
}

// mapToOrderedDataSource fills an orderedDataSource from the JSON-decoded map.
func mapToOrderedDataSource(m map[string]interface{}) orderedDataSource {
	return orderedDataSource{
		Version:    m["version"],
		Type:       m["type"],
		Name:       m["name"],
		Context:    m["context"],
		Id:         m["id"],
		Rest:       m["rest"],
		Structured: m["structured"],
	}
}

// mapToSortedYAMLNode converts a map to a yaml.MappingNode with
// alphabetically-sorted keys (the previous default behaviour).
func mapToSortedYAMLNode(m map[string]interface{}) (*yaml.Node, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		valNode := &yaml.Node{}
		if err := valNode.Encode(m[k]); err != nil {
			return nil, err
		}
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}
		mapping.Content = append(mapping.Content, keyNode, valNode)
	}
	return mapping, nil
}
