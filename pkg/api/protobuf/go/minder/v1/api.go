// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/util/jsonyaml"
)

const (
	// VersionV1 is the v1 version of most API resources
	VersionV1 = "v1"
)

var (
	// ErrInvalidResource is returned when an invalid resource is provided
	ErrInvalidResource = fmt.Errorf("invalid resource")
	// ErrInvalidResourceType is returned when an invalid resource type is provided
	ErrInvalidResourceType = fmt.Errorf("invalid resource type")
	// ErrNotAResource is returned when the provided resource is not a Minder resource
	ErrNotAResource = fmt.Errorf("not a Minder resource")
	// ErrResourceTypeMismatch is returned when the resource type does not match the provided resource
	ErrResourceTypeMismatch = fmt.Errorf("resource type mismatch")
)

// ResourceType is the type of resource. Minder resources are all the objects that Minder manages.
// They include policy resources (rule types, profiles, etc.), entity resources (repositories, artifacts, etc.),
// and other resources like projects.
type ResourceType string

const (
	// RuleTypeResource is a rule type resource
	RuleTypeResource ResourceType = "rule-type"
	// ProfileResource is a profile resource
	ProfileResource ResourceType = "profile"
	// EntityInstanceResource is an entity instance resource
	EntityInstanceResource ResourceType = "entity-instance"
	// RepositoryResource is a repository resource. Note that this
	// will be deprecated in the future. In favor of EntityInstanceResource.
	RepositoryResource ResourceType = "repository"
	// ArtifactResource is an artifact resource. Note that this
	// will be deprecated in the future. In favor of EntityInstanceResource.
	ArtifactResource ResourceType = "artifact"
	// ProjectResource is a project resource
	ProjectResource ResourceType = "project"
	// DataSourceResource is a data source resource
	DataSourceResource ResourceType = "data-source"
)

// ResourceTypeIsValid checks if the resource type is valid
func ResourceTypeIsValid(rt ResourceType) bool {
	_, ok := resourceMatchers[rt]
	return ok
}

var (
	resourceMatchers = map[ResourceType]protoreflect.ProtoMessage{
		RuleTypeResource:       &RuleType{},
		ProfileResource:        &Profile{},
		EntityInstanceResource: &EntityInstance{},
		RepositoryResource:     &Repository{},
		ArtifactResource:       &Artifact{},
		DataSourceResource:     &DataSource{},
	}
)

// ResourceMeta defines the basic metadata for a resource within
// Minder. This is used as a common interface and to determine
// the type of resource.
type ResourceMeta interface {
	protoreflect.ProtoMessage

	GetVersion() string
	GetType() string
	GetName() string
}

var _ ResourceMeta = (*RuleType)(nil)
var _ ResourceMeta = (*Profile)(nil)

// ParseResource is a generic parser for Minder resources. It will
// attempt to parse the resource into the correct type based on the
// version and type of the resource.
func ParseResource(r io.Reader, rm ResourceMeta) error {
	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}

	if err := jsonyaml.TranscodeYAMLToJSON(r, w); err != nil {
		return fmt.Errorf("error converting yaml to json: %w", err)
	}

	if err := json.NewDecoder(w).Decode(rm); err != nil {
		return errors.Join(ErrNotAResource, fmt.Errorf("error decoding resource: %w", err))
	}

	if err := Validate(rm); err != nil {
		return fmt.Errorf("error validating resource meta: %w", err)
	}

	// Attempt to match resource type before trying to decode
	if !ResourceMatches(ResourceType(rm.GetType()), rm) {
		return fmt.Errorf("resource type does not match: %w", ErrResourceTypeMismatch)
	}

	return nil
}

// ParseResourceProto is a generic parser for Minder resources, similar to ParseResource.
// However, this function will decode the resource using the protojson package, which allows
// for more control over the decoding process and more complex cases such as one-of fields.
func ParseResourceProto(r io.Reader, rm ResourceMeta) error {
	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}

	if err := jsonyaml.TranscodeYAMLToJSON(r, w); err != nil {
		return fmt.Errorf("error converting yaml to json: %w", err)
	}

	if err := protojson.Unmarshal(w.Bytes(), rm); err != nil {
		return errors.Join(ErrNotAResource, fmt.Errorf("error decoding resource: %w", err))
	}

	if err := Validate(rm); err != nil {
		return fmt.Errorf("error validating resource meta: %w", err)
	}

	// Attempt to match resource type before trying to decode
	if !ResourceMatches(ResourceType(rm.GetType()), rm) {
		return fmt.Errorf("resource type does not match: %w", ErrResourceTypeMismatch)
	}

	return nil
}

// Validate is a utility function which allows for the validation of a struct.
func Validate(r ResourceMeta) error {
	if r == nil {
		return fmt.Errorf("resource meta is nil")
	}

	if err := validate.Struct(r); err != nil {
		return errors.Join(ErrInvalidResource, err)
	}

	if valid := ResourceTypeIsValid(ResourceType(r.GetType())); !valid {
		return fmt.Errorf("%w: invalid resource type: %s", ErrInvalidResourceType, r.GetType())
	}

	return nil
}

// ResourceMatches checks if the resource type matches the provided resource.
func ResourceMatches(rt ResourceType, r protoreflect.ProtoMessage) bool {
	if r == nil {
		return false
	}

	matcher, ok := resourceMatchers[rt]
	if !ok {
		return false
	}

	return r.ProtoReflect().Descriptor().FullName() == matcher.ProtoReflect().Descriptor().FullName()
}

// YouMayHaveTheWrongResource is a utility function to verify if the given
// error is due to the resource type mismatch.
func YouMayHaveTheWrongResource(err error) bool {
	return err != nil && (errors.Is(err, ErrResourceTypeMismatch) || errors.Is(err, ErrNotAResource) ||
		errors.Is(err, ErrInvalidResource) || errors.Is(err, ErrInvalidResourceType))
}
