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

	"golang.org/x/oauth2"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// DockerHub is the struct that contains the Docker Hub client
type DockerHub struct {
	cli       *http.Client
	namespace string
	target    *url.URL
}

// Ensure that the Docker Hub client implements the ImageLister interface
var _ provifv1.ImageLister = (*DockerHub)(nil)

// New creates a new Docker Hub client
func New(cred provifv1.Credential, ns string) (*DockerHub, error) {
	var cli *http.Client
	oauth2cred, ok := cred.(provifv1.OAuth2TokenCredential)
	if ok {
		cli = oauth2.NewClient(context.Background(), oauth2cred.GetAsOAuth2TokenSource())
	} else {
		cli = http.DefaultClient
	}

	u, err := url.Parse("https://hub.docker.com/v2/repositories")
	if err != nil {
		return nil, fmt.Errorf("error parsing base URL: %w", err)
	}

	t := u.JoinPath(ns)

	return &DockerHub{
		namespace: ns,
		cli:       cli,
		target:    t,
	}, nil
}

// ListImages lists the containers in the Docker Hub
func (d *DockerHub) ListImages(ctx context.Context) ([]string, error) {
	req, err := http.NewRequest("GET", d.target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := d.cli.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized: %s", resp.Status)
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, errors.New("not found")
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// read body
	defer resp.Body.Close()

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
