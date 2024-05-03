// Copyright 2023 Stacklok, Inc.
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

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func urlFromEndpointAndPaths(
	baseUrl string,
	endpoint string,
	packageName string,
	ecosystem string,
) (*url.URL, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}
	u = u.JoinPath(endpoint)

	// Add query parameters for package_name and package_type
	q := u.Query()
	q.Set("package_name", packageName)
	q.Set("package_type", ecosystem)
	u.RawQuery = q.Encode()

	return u, nil
}

type trustyClient struct {
	client  *http.Client
	baseUrl string
}

// Alternative is an alternative package returned from the package intelligence API
type Alternative struct {
	PackageName    string  `json:"package_name"`
	Score          float64 `json:"score"`
	PackageNameURL string
}

// ScoreSummary is the summary score returned from the package intelligence API
type ScoreSummary struct {
	Score       *float64       `json:"score"`
	Description map[string]any `json:"description"`
}

// Reply is the response from the package intelligence API
type Reply struct {
	PackageName  string       `json:"package_name"`
	PackageType  string       `json:"package_type"`
	Summary      ScoreSummary `json:"summary"`
	Alternatives struct {
		Status   string        `json:"status"`
		Packages []Alternative `json:"packages"`
	} `json:"alternatives"`
}

func newPiClient(baseUrl string) *trustyClient {
	return &trustyClient{
		client:  &http.Client{},
		baseUrl: baseUrl,
	}
}

func (p *trustyClient) newRequest(ctx context.Context, dep *pb.Dependency) (*http.Request, error) {
	u, err := urlFromEndpointAndPaths(p.baseUrl, "v1/report", dep.Name, strings.ToLower(dep.Ecosystem.AsString()))
	if err != nil {
		return nil, fmt.Errorf("could not parse endpoint: %w", err)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (p *trustyClient) SendRecvRequest(ctx context.Context, dep *pb.Dependency) (*Reply, error) {
	req, err := p.newRequest(ctx, dep)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var piReply Reply
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&piReply); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return &piReply, nil
}
