// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ghcr provides a client for interacting with the GitHub Container Registry
package ghcr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v63/github"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/verifier/verifyif"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// GHCR is the string that represents the GHCR provider
const GHCR = "ghcr"

// ImageLister is the struct that contains the ImageLister client
type ImageLister struct {
	client *github.Client
	cfg    *minderv1.GHCRProviderConfig
}

// Ensure that the GHCR client implements the ContainerLister interface
var _ provifv1.ImageLister = (*ImageLister)(nil)

// New creates a new GHCR client
func New(cred provifv1.OAuth2TokenCredential, cfg *minderv1.GHCRProviderConfig) *ImageLister {
	tc := oauth2.NewClient(context.Background(), cred.GetAsOAuth2TokenSource())

	return &ImageLister{
		client: github.NewClient(tc),
		cfg:    cfg,
	}
}

// ParseV1Config parses the raw configuration into a GHCR configuration
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.GHCRProviderConfig, error) {
	type wrapper struct {
		GHCR *minderv1.GHCRProviderConfig `json:"ghcr" yaml:"ghcr" mapstructure:"ghcr"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	return w.GHCR, nil
}

// CanImplement returns true/false depending on whether the GHCR client can implement the specified trait
func (*ImageLister) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER
}

// FromGitHubClient creates a new GHCR client from an existing GitHub client
func FromGitHubClient(client *github.Client, namespace string) *ImageLister {
	return &ImageLister{
		client: client,
		cfg: &minderv1.GHCRProviderConfig{
			Namespace: proto.String(namespace),
		},
	}
}

// GetNamespaceURL returns the URL of the GHCR container namespace
func (g *ImageLister) GetNamespaceURL() string {
	return fmt.Sprintf("ghcr.io/%s", g.getNamespace())
}

// getNamespace returns the namespace of the GHCR client
func (g *ImageLister) getNamespace() string {
	return g.cfg.GetNamespace()
}

// ListImages lists the containers in the GHCR
func (g *ImageLister) ListImages(ctx context.Context) ([]string, error) {
	pageNumber := 0
	itemsPerPage := 100
	pt := string(verifyif.ArtifactTypeContainer)
	opt := &github.PackageListOptions{
		PackageType: &pt,
		ListOptions: github.ListOptions{
			Page:    pageNumber,
			PerPage: itemsPerPage,
		},
	}
	// create a slice to hold the containers
	var allContainers []string
	for {
		var artifacts []*github.Package
		var resp *github.Response
		var err error

		// TODO: handle organizations
		// artifacts, resp, err = g.client.Organizations.ListPackages(ctx, g.namespace, opt)
		artifacts, resp, err = g.client.Users.ListPackages(ctx, g.getNamespace(), opt)
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return allContainers, fmt.Errorf("packages not found for namespace %s: %w",
					g.getNamespace(), errors.New("not found"))
			}

			return allContainers, err
		}

		for _, artifact := range artifacts {
			allContainers = append(allContainers, artifact.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allContainers, nil
}

// FetchAllProperties implements the provider interface
// TODO: Implement this
func (*ImageLister) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the provider interface
// TODO: Implement this
func (*ImageLister) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the provider interface
// TODO: Implement this
func (*ImageLister) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

// SupportsEntity implements the Provider interface
func (*ImageLister) SupportsEntity(_ minderv1.Entity) bool {
	// TODO: implement
	return false
}

// RegisterEntity implements the Provider interface
func (i *ImageLister) RegisterEntity(
	_ context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !i.SupportsEntity(entType) {
		return nil, provifv1.ErrUnsupportedEntity
	}
	// we don't need to do any explicit registration
	return props, nil
}

// DeregisterEntity implements the Provider interface
func (*ImageLister) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	// TODO: implement
	return nil
}

// PropertiesToProtoMessage implements the Provider interface
func (*ImageLister) PropertiesToProtoMessage(_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	// TODO: Implement
	return nil, nil
}
