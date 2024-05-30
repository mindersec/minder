// Copyright 2024 Stacklok, Inc
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

	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/oci"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// DockerHub is the string that represents the DockerHub provider
const DockerHub = "dockerhub"

const (
	dockerioBaseURL = "docker.io"
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

// ParseV1Config parses the raw config into a DockerHubProviderConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.DockerHubProviderConfig, error) {
	type wrapper struct {
		DockerHub *minderv1.DockerHubProviderConfig `json:"dockerhub" yaml:"dockerhub" mapstructure:"dockerhub" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.DockerHub.Validate(); err != nil {
		return nil, fmt.Errorf("error validating DockerHub v1 provider config: %w", err)
	}

	return w.DockerHub, nil
}

func (d *dockerHubImageLister) GetNamespaceURL() string {
	return d.target.String()
}

// CanImplement returns true if the provider can implement the specified trait
func (_ *dockerHubImageLister) CanImplement(trait minderv1.ProviderType) bool {
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
