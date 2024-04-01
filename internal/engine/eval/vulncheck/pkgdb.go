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
	"net/url"
	"strings"

	"github.com/puzpuzpuz/xsync/v3"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func urlFromEndpointAndPaths(
	endpoint string,
	pathComponents ...string,
) (*url.URL, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}
	u = u.JoinPath(pathComponents...)

	return u, nil
}

// The patchLocatorFormatter interface is used to format the patch suggestion
// for the particular package manager. The interface should probably be refactored
// and each type implementing its own interface should handle the indenting rather
// than the review handler.
type patchLocatorFormatter interface {
	LineHasDependency(line string) bool
	IndentedString(indent int, oldDepLine string, oldDep *pb.Dependency) string
	HasPatchedVersion() bool
	GetPatchedVersion() string
}

// RepoQuerier is the interface for querying a repository
type RepoQuerier interface {
	SendRecvRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool) (patchLocatorFormatter, error)
	NoPatchAvailableFormatter(dep *pb.Dependency) patchLocatorFormatter
}

type repoCache struct {
	cache *xsync.MapOf[string, RepoQuerier]
}

func newRepoCache() *repoCache {
	return &repoCache{
		cache: xsync.NewMapOf[string, RepoQuerier](),
	}
}

func (rc *repoCache) newRepository(ecoConfig *ecosystemConfig) (RepoQuerier, error) {
	if repo, exists := rc.cache.Load(ecoConfig.Name); exists {
		return repo, nil
	}

	var repo RepoQuerier
	switch ecoConfig.Name {
	case "npm":
		repo = newNpmRepository(ecoConfig.PackageRepository.Url)
	case "go":
		repo = newGoProxySumRepository(ecoConfig.PackageRepository.Url, ecoConfig.SumRepository.Url)
	case "pypi":
		repo = newPyPIRepository(ecoConfig.PackageRepository.Url)
	default:
		return nil, fmt.Errorf("unknown ecosystem: %s", ecoConfig.Name)
	}

	rc.cache.Store(ecoConfig.Name, repo)
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

func (pj *packageJson) IndentedString(indent int, oldDepLine string, _ *pb.Dependency) string {
	padding := fmt.Sprintf("%*s", indent, "")
	innerPadding := padding + "  " // Add 2 extra spaces

	// use the old dependency to get the correct package path
	data := fmt.Sprintf("%s\n", oldDepLine)
	// format each line with leadingWhitespace and 2 extra spaces
	data += innerPadding + fmt.Sprintf("\"version\": \"%s\",\n", pj.Version)
	data += innerPadding + fmt.Sprintf("\"resolved\": \"%s\",\n", pj.Dist.Tarball)
	data += innerPadding + fmt.Sprintf("\"integrity\": \"%s\",", pj.Dist.Integrity)

	return data
}

func (pj *packageJson) LineHasDependency(versionPack string) bool {
	// In npmVersionPack we are searching for the following:
	// 0: the current file line (presumably the npm package name)
	// 1: the version of the npm package
	// 2: the version line that we are looking for
	npmVersionPack := strings.Split(versionPack, "\n")
	if npmVersionPack == nil || len(npmVersionPack) != 3 {
		return false
	}

	pkgLine := fmt.Sprintf(`/%s": {`, pj.Name)
	return strings.Contains(npmVersionPack[0], pkgLine) && strings.Contains(npmVersionPack[1], npmVersionPack[2])
}

func (pj *packageJson) HasPatchedVersion() bool {
	return pj.Version != ""
}

func (pj *packageJson) GetPatchedVersion() string {
	return pj.Version
}

// check that pypi repository implements RepoQuerier
var _ RepoQuerier = (*pypiRepository)(nil)

type pypiRepository struct {
	client   *http.Client
	endpoint string
}

// PyPiReply is the reply from the PyPi API
type PyPiReply struct {
	Info struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"info"`
}

// IndentedString returns the patch suggestion for a requirement.txt file
// This method satisfies the patchLocatorFormatter interface where different
// package managers have different patch formats and different ways of presenting
// them. Since PyPi doesn't indent, but can specify zero or multiple versions, we
// don't care about the indent parameter. This is ripe for refactoring, though,
// see the comment in the patchLocatorFormatter interface.
func (p *PyPiReply) IndentedString(_ int, oldDepLine string, oldDep *pb.Dependency) string {
	return strings.Replace(oldDepLine, oldDep.Version, p.Info.Version, 1)
}

// LineHasDependency returns true if the requirement.txt line is for the same package as the receiver
func (p *PyPiReply) LineHasDependency(line string) bool {
	nameMatch := util.PyRequestsNameRegexp.FindStringIndex(line)
	if nameMatch == nil {
		return false
	}

	name := strings.TrimSpace(line[:nameMatch[0]])
	return name == p.Info.Name
}

// HasPatchedVersion returns true if the vulnerable package can be updated to a patched version
func (p *PyPiReply) HasPatchedVersion() bool {
	return p.Info.Version != ""
}

// GetPatchedVersion returns the suggested patch version for a vulnerable package
func (p *PyPiReply) GetPatchedVersion() string {
	return p.Info.Version
}

func (p *pypiRepository) SendRecvRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool,
) (patchLocatorFormatter, error) {
	req, err := p.newRequest(ctx, dep, patched, latest)
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

	var pkgJson PyPiReply
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&pkgJson); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return &pkgJson, nil
}

func (_ *pypiRepository) NoPatchAvailableFormatter(dep *pb.Dependency) patchLocatorFormatter {
	return &PyPiReply{
		Info: struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}{Name: dep.Name, Version: ""},
	}
}

func newPyPIRepository(endpoint string) *pypiRepository {
	return &pypiRepository{
		client:   &http.Client{},
		endpoint: endpoint,
	}
}

func (p *pypiRepository) newRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool) (*http.Request, error) {
	var u *url.URL
	var err error

	if latest {
		u, err = urlFromEndpointAndPaths(p.endpoint, dep.Name, "json")
	} else {
		u, err = urlFromEndpointAndPaths(p.endpoint, dep.Name, patched, "json")
	}

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

func (n *npmRepository) newRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool) (*http.Request, error) {
	var version string
	if latest {
		version = "latest"
	} else {
		version = patched
	}
	u, err := urlFromEndpointAndPaths(n.endpoint, dep.Name, version)
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

func (n *npmRepository) SendRecvRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool,
) (patchLocatorFormatter, error) {
	req, err := n.newRequest(ctx, dep, patched, latest)
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

func (_ *npmRepository) NoPatchAvailableFormatter(dep *pb.Dependency) patchLocatorFormatter {
	return &packageJson{
		Name:    dep.Name,
		Version: "",
	}
}

type goModPackage struct {
	// just for locating in the patch
	oldVersion string

	Name           string `json:"name"`
	Version        string `json:"version"`
	ModuleHash     string `json:"module_hash"`
	DependencyHash string `json:"dependency_hash"`
}

func (gmp *goModPackage) IndentedString(indent int, _ string, _ *pb.Dependency) string {
	return fmt.Sprintf("%s%s %s", strings.Repeat(" ", indent), gmp.Name, gmp.Version)
}

func (gmp *goModPackage) LineHasDependency(line string) bool {
	return strings.Contains(line, gmp.Name) && strings.Contains(line, gmp.oldVersion)
}

func (gmp *goModPackage) HasPatchedVersion() bool {
	return gmp.Version != ""
}

func (gmp *goModPackage) GetPatchedVersion() string {
	return gmp.Version
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

func (r *goProxyRepository) goProxyRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool,
) (*http.Request, error) {
	var u *url.URL
	var err error

	if latest {
		u, err = urlFromEndpointAndPaths(r.proxyEndpoint, dep.Name, "@latest")
	} else {
		var version string
		if !strings.HasPrefix(patched, "v") {
			version = "v" + patched
		} else {
			version = patched
		}
		u, err = urlFromEndpointAndPaths(r.proxyEndpoint, dep.Name, "@v", version+".info")
	}

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

func (r *goProxyRepository) goSumRequest(ctx context.Context, depName, depVersion string) (*http.Request, error) {
	u, err := urlFromEndpointAndPaths(r.sumEndpoint,
		"lookup",
		fmt.Sprintf("%s@%s", depName, depVersion))
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

func (r *goProxyRepository) SendRecvRequest(ctx context.Context, dep *pb.Dependency, patched string, latest bool,
) (patchLocatorFormatter, error) {
	proxyReq, err := r.goProxyRequest(ctx, dep, patched, latest)
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
		return nil, fmt.Errorf("could not find patched version for %s", dep.Name)
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

func (_ *goProxyRepository) NoPatchAvailableFormatter(dep *pb.Dependency) patchLocatorFormatter {
	return &goModPackage{
		Name:       dep.Name,
		oldVersion: dep.Version,
	}
}
