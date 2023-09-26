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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

type patchFormatter interface {
	IndentedString(int) string
}

// RepoQuerier is the interface for querying a repository
type RepoQuerier interface {
	SendRecvRequest(ctx context.Context, dep *pb.Dependency) (patchFormatter, error)
}

type repoCache struct {
	cache map[string]RepoQuerier
}

func newRepoCache() *repoCache {
	return &repoCache{
		cache: make(map[string]RepoQuerier),
	}
}

func (rc *repoCache) newRepository(ecoConfig *ecosystemConfig) (RepoQuerier, error) {
	if repo, exists := rc.cache[ecoConfig.Name]; exists {
		return repo, nil
	}

	var repo RepoQuerier
	switch ecoConfig.Name {
	case "npm":
		repo = newNpmRepository(ecoConfig.PackageRepository.Url)
	case "go":
		repo = newGoProxySumRepository(ecoConfig.PackageRepository.Url, ecoConfig.SumRepository.Url)
	default:
		return nil, fmt.Errorf("unknown ecosystem: %s", ecoConfig.Name)
	}

	rc.cache[ecoConfig.Name] = repo
	return repo, nil
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
	client   *http.Client
	endpoint string
}

func newNpmRepository(endpoint string) *npmRepository {
	return &npmRepository{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

// check that npmRepository implements RepoQuerier
var _ RepoQuerier = (*npmRepository)(nil)

func (n *npmRepository) newRequest(ctx context.Context, dep *pb.Dependency) (*http.Request, error) {
	pkgUrl := fmt.Sprintf("%s/%s/latest", n.endpoint, dep.Name)
	req, err := http.NewRequest("GET", pkgUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (n *npmRepository) SendRecvRequest(ctx context.Context, dep *pb.Dependency) (patchFormatter, error) {
	req, err := n.newRequest(ctx, dep)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := n.client.Do(req)
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

type goModPackage struct {
	// just for locating in the patch
	oldVersion string

	Name           string `json:"name"`
	Version        string `json:"version"`
	ModuleHash     string `json:"module_hash"`
	DependencyHash string `json:"dependency_hash"`
}

func (gmp *goModPackage) IndentedString(_ int) string {
	return fmt.Sprintf("%s %s %s\n%s %s/go.mod %s",
		gmp.Name, gmp.Version, gmp.ModuleHash,
		gmp.Name, gmp.Version, gmp.DependencyHash)
}

func (gmp *goModPackage) LineHasDependency(line string) bool {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return false
	}
	return parts[0] == gmp.Name && parts[1] == gmp.oldVersion
}

type goProxyRepository struct {
	proxyClient   *http.Client
	sumClient     *http.Client
	proxyEndpoint string
	sumEndpoint   string
}

func newGoProxySumRepository(proxyEndpoint, sumEndpoint string) *goProxyRepository {
	return &goProxyRepository{
		proxyClient:   &http.Client{},
		sumClient:     &http.Client{},
		proxyEndpoint: proxyEndpoint,
		sumEndpoint:   sumEndpoint,
	}
}

// check that npmRepository implements RepoQuerier
var _ RepoQuerier = (*goProxyRepository)(nil)

func (r *goProxyRepository) goProxyRequest(ctx context.Context, dep *pb.Dependency) (*http.Request, error) {
	pkgUrl := fmt.Sprintf("%s/%s/@latest", r.proxyEndpoint, dep.Name)
	req, err := http.NewRequest("GET", pkgUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (r *goProxyRepository) goSumRequest(ctx context.Context, depName, depVersion string) (*http.Request, error) {
	sumUrl := fmt.Sprintf("%s/lookup/%s@%s", r.sumEndpoint, depName, depVersion)
	req, err := http.NewRequest("GET", sumUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func parseGoSumReply(goPkg *goModPackage, reply io.Reader) error {
	lines := []string{}

	scanner := bufio.NewScanner(reply)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) == 3 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read go.sum reply: %w", err)
	}

	parts := strings.Split(lines[1], " ")
	if len(parts) != 3 {
		return fmt.Errorf("unexpected format for go.mod checksum line")
	}
	if parts[0] != goPkg.Name {
		return fmt.Errorf("go.mod checksum line does not match dependency name (got %s, expected %s)", parts[0], goPkg.Name)
	}
	if parts[1] != goPkg.Version {
		return fmt.Errorf("go.mod checksum line does not match dependency version (got %s, expected %s)", parts[1], goPkg.Version)
	}
	goPkg.ModuleHash = parts[2]

	parts = strings.Split(lines[2], " ")
	if len(parts) != 3 {
		return fmt.Errorf("unexpected format for go.mod checksum line")
	}
	goPkg.DependencyHash = parts[2]

	return nil
}

func (r *goProxyRepository) SendRecvRequest(ctx context.Context, dep *pb.Dependency) (patchLocatorFormatter, error) {
	proxyReq, err := r.goProxyRequest(ctx, dep)
	if err != nil {
		return nil, fmt.Errorf("could not create pkg db request: %w", err)
	}

	proxyResp, err := r.proxyClient.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer proxyResp.Body.Close()

	if proxyResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response: %d", proxyResp.StatusCode)
	}

	goPackage := &goModPackage{
		Name:       dep.Name,
		oldVersion: dep.Version,
	}
	dec := json.NewDecoder(proxyResp.Body)
	if err := dec.Decode(&goPackage); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	if goPackage.Version == "" {
		return nil, fmt.Errorf("could not find latest version for %s", dep.Name)
	}

	sumReq, err := r.goSumRequest(ctx, goPackage.Name, goPackage.Version)
	if err != nil {
		return nil, fmt.Errorf("could not create sum db request: %w", err)
	}

	sumResp, err := r.sumClient.Do(sumReq)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer sumResp.Body.Close()

	if sumResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response from gosum: %d", proxyResp.StatusCode)
	}

	if err := parseGoSumReply(goPackage, sumResp.Body); err != nil {
		return nil, fmt.Errorf("could not parse go.sum reply: %w", err)
	}

	return goPackage, nil
}
