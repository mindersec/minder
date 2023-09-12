// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

type patchFormatter interface {
	IndentedString(int) string
}

// RepoQuerier is the interface for querying a repository
type RepoQuerier interface {
	NewRequest(ctx context.Context, dep *pb.Dependency) (*http.Request, error)
	SendRecvRequest(*http.Request) (patchFormatter, error)
}

func newRepository(ecoConfig *ecosystemConfig) (RepoQuerier, error) {
	switch ecoConfig.Name {
	case "npm":
		// TODO(jakub): make this configurable
		return newNpmRepository(ecoConfig.PackageRepository.Url), nil
	default:
		return nil, fmt.Errorf("unknown ecosystem: %s", ecoConfig.Name)
	}
}

type packageJson struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		Integrity string `json:"integrity"`
		Tarball   string `json:"tarball"`
	} `json:"dist"`
}

func (pj *packageJson) IndentedString(leadingWhitespace int) string {
	padding := fmt.Sprintf("%*s", leadingWhitespace, "")
	innerPadding := padding + "  " // Add 2 extra spaces

	// format each line with leadingWhitespace and 2 extra spaces
	data := padding + fmt.Sprintf("\"%s\": {\n", pj.Name)
	data += innerPadding + fmt.Sprintf("\"version\": \"%s\",\n", pj.Version)
	data += innerPadding + fmt.Sprintf("\"resolved\": \"%s\",\n", pj.Dist.Tarball)
	data += innerPadding + fmt.Sprintf("\"integrity\": \"%s\",", pj.Dist.Integrity)

	return data
}

type npmRepository struct {
	endpoint string
}

func newNpmRepository(endpoint string) *npmRepository {
	return &npmRepository{
		endpoint: endpoint,
	}
}

func (n npmRepository) NewRequest(ctx context.Context, dep *pb.Dependency) (*http.Request, error) {
	pkgUrl := fmt.Sprintf("%s/%s/latest", n.endpoint, dep.Name)
	req, err := http.NewRequest("GET", pkgUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (_ npmRepository) SendRecvRequest(request *http.Request) (patchFormatter, error) {
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	var pkgJson packageJson
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pkgJson); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return &pkgJson, nil
}
