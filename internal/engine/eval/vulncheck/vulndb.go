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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Vulnerability is a vulnerability JSON representation
type Vulnerability struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Details    string `json:"details"`
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

// VulnerabilityResponse is a response from the vulnerability database
type VulnerabilityResponse struct {
	Vulns []Vulnerability `json:"vulns"`
}

// TODO(jakub): it's ugly that we depend on types from ingester/diff
type vulnDb interface {
	NewQuery(ctx context.Context, dep *pb.Dependency, eco pb.DepEcosystem) (*http.Request, error)
	SendRecvRequest(r *http.Request) (*VulnerabilityResponse, error)
}

// OSVResponse is a response from the OSV database
type OSVResponse struct {
	Vulns []struct {
		ID               string    `json:"id"`
		Summary          string    `json:"summary"`
		Details          string    `json:"details"`
		Aliases          []string  `json:"aliases"`
		Modified         time.Time `json:"modified"`
		Published        time.Time `json:"published"`
		DatabaseSpecific struct {
			GithubReviewedAt string   `json:"github_reviewed_at"`
			GithubReviewed   bool     `json:"github_reviewed"`
			Severity         string   `json:"severity"`
			CweIDs           []string `json:"cwe_ids"`
			NvdPublishedAt   string   `json:"nvd_published_at"`
		} `json:"database_specific"`
		References []struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"references"`
		Affected []struct {
			Package struct {
				Name      string `json:"name"`
				Ecosystem string `json:"ecosystem"`
				Purl      string `json:"purl"`
			} `json:"package"`
			Ranges []struct {
				Type   string `json:"type"`
				Events []struct {
					Introduced string `json:"introduced,omitempty"`
					Fixed      string `json:"fixed,omitempty"`
				} `json:"events"`
			} `json:"ranges"`
			DatabaseSpecific struct {
				Source string `json:"source"`
			} `json:"database_specific"`
		} `json:"affected"`
		SchemaVersion string `json:"schema_version"`
		Severity      []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		} `json:"severity"`
	} `json:"vulns"`
}

func toVulnerabilityResponse(osvResp *OSVResponse) *VulnerabilityResponse {
	var vulnResp VulnerabilityResponse

	for _, osvVuln := range osvResp.Vulns {
		vuln := Vulnerability{
			ID:      osvVuln.ID,
			Summary: osvVuln.Summary,
			Details: osvVuln.Details,
		}

		// TODO(jakub): this only takes the first introduced/fixed version
		for _, affected := range osvVuln.Affected {
			for _, r := range affected.Ranges {
				for _, event := range r.Events {
					if event.Introduced != "" {
						vuln.Introduced = event.Introduced
					}
					if event.Fixed != "" {
						vuln.Fixed = event.Fixed
					}
				}
			}
		}

		// Add to the result
		vulnResp.Vulns = append(vulnResp.Vulns, vuln)
	}
	return &vulnResp
}

type osvdb struct {
	endpoint string
}

func newOsvDb(endpoint string) *osvdb {
	return &osvdb{
		endpoint: endpoint,
	}
}

func (o *osvdb) NewQuery(ctx context.Context, dep *pb.Dependency, eco pb.DepEcosystem) (*http.Request, error) {
	reqBody := map[string]interface{}{
		"version": dep.Version,
		"package": map[string]string{
			"name":      dep.Name,
			"ecosystem": pbEcosystemAsString(eco),
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", o.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (_ *osvdb) SendRecvRequest(r *http.Request) (*VulnerabilityResponse, error) {
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// TODO(jakub): use the JQ accessor isntead?
	var response OSVResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("could not decode response body: %w", err)
	}

	return toVulnerabilityResponse(&response), nil
}
