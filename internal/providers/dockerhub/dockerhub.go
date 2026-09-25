// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package dockerhub provides a client for interacting with Docker Hub
package dockerhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/verifier/verifyif"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// DockerHub is the string that represents the DockerHub provider
const DockerHub = "dockerhub"

const (
	dockerioBaseURL = "docker.io"

	defaultTag = "latest"
)

// Implements is the list of provider types that the DockerHub provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeImageLister,
	db.ProviderTypeOci,
}

// AuthorizationFlows is the list of authorization flows that the DockerHub provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
}

// dockerHubImageLister is the struct that contains the Docker Hub specific operations
type dockerHubImageLister struct {
	*oci.OCI
	cred      provifv1.OAuth2TokenCredential
	cli       *http.Client
	namespace string
	target    *url.URL
	cfg       *minderv1.DockerHubProviderConfig
}

// Ensure that the Docker Hub client implements the ImageLister interface
var _ provifv1.ImageLister = (*dockerHubImageLister)(nil)

// New creates a new Docker Hub client
func New(cred provifv1.OAuth2TokenCredential, cfg *minderv1.DockerHubProviderConfig) (*dockerHubImageLister, error) {
	cli := oauth2.NewClient(context.Background(), cred.GetAsOAuth2TokenSource())

	u, err := url.Parse("https://hub.docker.com/v2/repositories")
	if err != nil {
		return nil, fmt.Errorf("error parsing base URL: %w", err)
	}

	ns := cfg.GetNamespace()
	t := u.JoinPath(ns)

	o := oci.New(cred, dockerioBaseURL, path.Join(dockerioBaseURL, cfg.GetNamespace()))
	return &dockerHubImageLister{
		OCI:       o,
		namespace: ns,
		cred:      cred,
		cli:       cli,
		target:    t,
		cfg:       cfg,
	}, nil
}

type dhConfigWrapper struct {
	DockerHub *minderv1.DockerHubProviderConfig `json:"dockerhub" yaml:"dockerhub" mapstructure:"dockerhub" validate:"required"`
}

// ParseV1Config parses the raw config into a DockerHubProviderConfig struct
//
// TODO: This should be moved to a common location
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.DockerHubProviderConfig, error) {
	var w dhConfigWrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.DockerHub.Validate(); err != nil {
		return nil, fmt.Errorf("error validating DockerHub v1 provider config: %w", err)
	}

	return w.DockerHub, nil
}

// MarshalV1Config marshals the DockerHubProviderConfig struct into a raw config
func MarshalV1Config(rawCfg json.RawMessage) (json.RawMessage, error) {
	var w dhConfigWrapper
	if err := json.Unmarshal(rawCfg, &w); err != nil {
		return nil, err
	}

	err := w.DockerHub.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating provider config: %w", err)
	}

	return json.Marshal(w)
}

func (d *dockerHubImageLister) GetNamespaceURL() string {
	return d.target.String()
}

// CanImplement returns true if the provider can implement the specified trait
func (*dockerHubImageLister) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_OCI
}

// ListImages lists the containers in the Docker Hub
func (d *dockerHubImageLister) ListImages(ctx context.Context) ([]string, error) {
	req, err := http.NewRequest("GET", d.target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := d.cli.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized: %s", resp.Status)
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, errors.New("not found")
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// parse body
	toParse := struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&toParse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var containers []string
	for _, r := range toParse.Results {
		containers = append(containers, r.Name)
	}

	return containers, nil
}

func parseImageRef(ref string) (string, string, error) {
	repo, tag, hasTag := strings.Cut(ref, ":")
	if repo == "" {
		return "", "", fmt.Errorf("invalid image reference %q: missing repository", ref)
	}
	if hasTag && tag == "" {
		return "", "", fmt.Errorf("invalid image reference %q: empty tag", ref)
	}
	if !hasTag {
		tag = defaultTag
	}
	return repo, tag, nil
}

func artifactNameFromProperties(props *properties.Properties) (string, error) {
	name, err := props.GetProperty(properties.PropertyName).AsString()
	if err != nil {
		return "", fmt.Errorf("failed to get artifact name: %w", err)
	}
	if name == "" {
		return "", errors.New("artifact name is empty")
	}
	return name, nil
}

// FetchAllProperties implements the provider interface
func (d *dockerHubImageLister) FetchAllProperties(
	ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	if !d.SupportsEntity(entType) {
		return nil, provifv1.ErrUnsupportedEntity
	}

	name, err := artifactNameFromProperties(getByProps)
	if err != nil {
		return nil, err
	}

	repo, tag, err := parseImageRef(name)
	if err != nil {
		return nil, err
	}

	digest, err := d.GetDigest(ctx, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve digest for %q: %w", name, err)
	}

	return properties.NewProperties(map[string]any{
		properties.PropertyName:         name,
		properties.PropertyUpstreamID:   digest,
		properties.ArtifactPropertyType: string(verifyif.ArtifactTypeContainer),
	}), nil
}

// FetchProperty implements the provider interface
// TODO: Implement this
func (*dockerHubImageLister) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the provider interface
func (d *dockerHubImageLister) GetEntityName(
	entType minderv1.Entity, props *properties.Properties,
) (string, error) {
	if !d.SupportsEntity(entType) {
		return "", fmt.Errorf("entity type %s not supported", entType)
	}

	return artifactNameFromProperties(props)
}

// SupportsEntity implements the Provider interface
func (*dockerHubImageLister) SupportsEntity(entType minderv1.Entity) bool {
	return entType == minderv1.Entity_ENTITY_ARTIFACTS
}

// CreationOptions implements the Provider interface
func (d *dockerHubImageLister) CreationOptions(entType minderv1.Entity) *provifv1.EntityCreationOptions {
	if !d.SupportsEntity(entType) {
		return nil
	}

	return &provifv1.EntityCreationOptions{
		RegisterWithProvider:       false,
		PublishReconciliationEvent: false,
	}
}

// RegisterEntity implements the Provider interface
func (d *dockerHubImageLister) RegisterEntity(
	_ context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !d.SupportsEntity(entType) {
		return nil, provifv1.ErrUnsupportedEntity
	}
	// we don't need to do any explicit registration
	return props, nil
}

// DeregisterEntity implements the Provider interface
func (*dockerHubImageLister) DeregisterEntity(
	_ context.Context, _ minderv1.Entity, _ *properties.Properties,
) error {
	// TODO: implement
	return nil
}

// PropertiesToProtoMessage implements the Provider interface
func (*dockerHubImageLister) PropertiesToProtoMessage(
	_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	// TODO: Implement
	return nil, nil
}
