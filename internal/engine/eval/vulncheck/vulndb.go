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
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"

	pbinternal "github.com/stacklok/minder/internal/proto"
)

// Vulnerability is a vulnerability JSON representation
type Vulnerability struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Details    string `json:"details"`
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
	Type       string `json:"type"`
}

// VulnerabilityResponse is a response from the vulnerability database
type VulnerabilityResponse struct {
	Vulns []Vulnerability `json:"vulns"`
}

// TODO(jakub): it's ugly that we depend on types from ingester/diff
type vulnDb interface {
	NewQuery(ctx context.Context, dep *pbinternal.Dependency, eco pbinternal.DepEcosystem) (*http.Request, error)
	SendRecvRequest(r *http.Request, dep *pbinternal.Dependency) (*VulnerabilityResponse, error)
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

func toVulnerabilityResponse(osvResp *OSVResponse, dep *pbinternal.Dependency) *VulnerabilityResponse {
	var vulnResp VulnerabilityResponse

	for _, osvVuln := range osvResp.Vulns {
		vuln := Vulnerability{
			ID:      osvVuln.ID,
			Summary: osvVuln.Summary,
			Details: osvVuln.Details,
		}

	affectedLoop:
		for _, affected := range osvVuln.Affected {
			for _, r := range affected.Ranges {
				vuln.Type = r.Type
				var introduced string
				var fixed string
				for _, event := range r.Events {
					if event.Introduced != "" {
						introduced = event.Introduced
					}
					if event.Fixed != "" {
						fixed = event.Fixed
					}
				}
				if r.Type == "SEMVER" && currentVersionInRange(dep.Version, introduced, fixed) {
					// we have found the fixed version with the smallest delta from the current version
					vuln.Introduced = introduced
					vuln.Fixed = fixed
					break affectedLoop
				}
				// if we can't determine which range the current version belongs to, use any range
				if introduced != "" {
					vuln.Introduced = introduced
				}
				if fixed != "" {
					vuln.Fixed = fixed
				}
			}
		}

		// Add to the result
		vulnResp.Vulns = append(vulnResp.Vulns, vuln)
	}
	return &vulnResp
}

func currentVersionInRange(currentVersion string, introducedVersion string, fixedVersion string) bool {
	if introducedVersion == "" || fixedVersion == "" {
		return false
	}
	versionString := strings.TrimPrefix(currentVersion, "v")
	current, err := version.NewVersion(versionString)
	if err != nil {
		return false
	}
	introduced, err := version.NewVersion(introducedVersion)
	if err != nil {
		return false
	}
	fixed, err := version.NewVersion(fixedVersion)
	if err != nil {
		return false
	}
	return introduced.LessThanOrEqual(current) && fixed.GreaterThan(current)
}

type osvdb struct {
	endpoint string
}

func newOsvDb(endpoint string) *osvdb {
	return &osvdb{
		endpoint: endpoint,
	}
}

func (o *osvdb) NewQuery(ctx context.Context, dep *pbinternal.Dependency, eco pbinternal.DepEcosystem) (*http.Request, error) {
	var dependencyName string

	if eco == pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI {
		dependencyName = pyNormalizeName(dep.Name)
	} else {
		dependencyName = dep.Name
	}

	reqBody := map[string]interface{}{
		"version": dep.Version,
		"package": map[string]string{
			"name":      dependencyName,
			"ecosystem": eco.AsString(),
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

func (_ *osvdb) SendRecvRequest(r *http.Request, dep *pbinternal.Dependency) (*VulnerabilityResponse, error) {
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

	return toVulnerabilityResponse(&response, dep), nil
}

// Normalize the package name for PyPI
// See https://packaging.python.org/en/latest/specifications/name-normalization/#name-normalization)
func pyNormalizeName(pkgName string) string {
	regex := regexp.MustCompile(`[-_.]+`)
	result := regex.ReplaceAllString(pkgName, "-")
	return strings.ToLower(result)
}
