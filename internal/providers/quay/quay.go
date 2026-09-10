// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package quay provides a client for interacting with Quay.io
package quay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"golang.org/x/oauth2"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/oci"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// Quay is the string that represents the Quay provider
const Quay = "quay"

const (
	quayioBaseURL = "quay.io"
)

// Implements is the list of provider types that the Quay provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeImageLister,
	db.ProviderTypeOci,
}

// AuthorizationFlows is the list of authorization flows that the Quay provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
}

// quayImageLister is the struct that contains the Quay specific operations
type quayImageLister struct {
	*oci.OCI
	cred      provifv1.OAuth2TokenCredential
	cli       *http.Client
	namespace string
	target    *url.URL
	cfg       *minderv1.QuayProviderConfig
}

// Ensure that the Quay client implements the ImageLister interface
var _ provifv1.ImageLister = (*quayImageLister)(nil)

// New creates a new Quay client
func New(cred provifv1.OAuth2TokenCredential, cfg *minderv1.QuayProviderConfig) (*quayImageLister, error) {
	cli := oauth2.NewClient(context.Background(), cred.GetAsOAuth2TokenSource())

	u, err := url.Parse("https://quay.io/api/v1/repository")
	if err != nil {
		return nil, fmt.Errorf("error parsing base URL: %w", err)
	}

	ns := cfg.GetNamespace()
	q := u.Query()
	q.Set("namespace", ns)
	u.RawQuery = q.Encode()

	o := oci.New(cred, quayioBaseURL, path.Join(quayioBaseURL, cfg.GetNamespace()))
	return &quayImageLister{
		OCI:       o,
		namespace: ns,
		cred:      cred,
		cli:       cli,
		target:    u,
		cfg:       cfg,
	}, nil
}

type quayConfigWrapper struct {
	Quay *minderv1.QuayProviderConfig `json:"quay" yaml:"quay" mapstructure:"quay" validate:"required"`
}

// ParseV1Config parses the raw config into a QuayProviderConfig struct
//
// TODO: This should be moved to a common location
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.QuayProviderConfig, error) {
	var w quayConfigWrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.Quay.Validate(); err != nil {
		return nil, fmt.Errorf("error validating Quay v1 provider config: %w", err)
	}

	return w.Quay, nil
}

// MarshalV1Config marshals the QuayProviderConfig struct into a raw config
func MarshalV1Config(rawCfg json.RawMessage) (json.RawMessage, error) {
	var w quayConfigWrapper
	if err := json.Unmarshal(rawCfg, &w); err != nil {
		return nil, err
	}

	err := w.Quay.Validate()
	if err != nil {
		return nil, fmt.Errorf("error validating provider config: %w", err)
	}

	return json.Marshal(w)
}

func (q *quayImageLister) GetNamespaceURL() string {
	return q.target.String()
}

// CanImplement returns true if the provider can implement the specified trait
func (*quayImageLister) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_OCI
}

// ListImages lists the containers in the Quay namespace
func (q *quayImageLister) ListImages(ctx context.Context) ([]string, error) {
	req, err := http.NewRequest("GET", q.target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := q.cli.Do(req.WithContext(ctx))
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
		Repositories []struct {
			Name string `json:"name"`
		} `json:"repositories"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&toParse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	var containers []string
	for _, r := range toParse.Repositories {
		containers = append(containers, r.Name)
	}

	return containers, nil
}

// FetchAllProperties implements the provider interface
// TODO: Implement this
func (*quayImageLister) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the provider interface
// TODO: Implement this
func (*quayImageLister) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the provider interface
// TODO: Implement this
func (*quayImageLister) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

// SupportsEntity implements the Provider interface
func (*quayImageLister) SupportsEntity(_ minderv1.Entity) bool {
	// TODO: implement
	return false
}

// CreationOptions implements the Provider interface
func (*quayImageLister) CreationOptions(_ minderv1.Entity) *provifv1.EntityCreationOptions {
	// Quay doesn't support any entities yet
	return nil
}

// RegisterEntity implements the Provider interface
func (q *quayImageLister) RegisterEntity(
	_ context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !q.SupportsEntity(entType) {
		return nil, provifv1.ErrUnsupportedEntity
	}
	// we don't need to do any explicit registration
	return props, nil
}

// DeregisterEntity implements the Provider interface
func (*quayImageLister) DeregisterEntity(
	_ context.Context, _ minderv1.Entity, _ *properties.Properties,
) error {
	// TODO: implement
	return nil
}

// PropertiesToProtoMessage implements the Provider interface
func (*quayImageLister) PropertiesToProtoMessage(
	_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	// TODO: Implement
	return nil, nil
}
