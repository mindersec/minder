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
	"io"
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

type npmRepository struct {
	endpoint string
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

// todo(jakub): get rid of this and use jq to parse the json
type packageJson struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"repository"`
	Homepage                string   `json:"homepage"`
	Description             string   `json:"description"`
	Main                    string   `json:"main"`
	Browser                 string   `json:"browser"`
	Module                  string   `json:"module"`
	Types                   string   `json:"types"`
	DevDependencies         struct{} `json:"devDependencies"`
	DevDependenciesComments struct {
		Typescript string `json:"typescript"`
	} `json:"devDependenciesComments"`
	Author struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"author"`
	Contributors []struct {
		Name  string `json:"name"`
		Email string `json:"email,omitempty"`
		URL   string `json:"url"`
	} `json:"contributors"`
	Keywords []string `json:"keywords"`
	Scripts  struct {
		Start        string `json:"start"`
		Lint         string `json:"lint"`
		Build        string `json:"build"`
		Test         string `json:"test"`
		CheckTypes   string `json:"check-types"`
		CypressOpen  string `json:"cypress:open"`
		WebpackBuild string `json:"webpack-build"`
	} `json:"scripts"`
	Funding struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"funding"`
	Prettier struct {
		PrintWidth    int    `json:"printWidth"`
		Semi          bool   `json:"semi"`
		SingleQuote   bool   `json:"singleQuote"`
		QuoteProps    string `json:"quoteProps"`
		TrailingComma string `json:"trailingComma"`
	} `json:"prettier"`
	Bugs struct {
		URL string `json:"url"`
	} `json:"bugs"`
	License     string `json:"license"`
	GitHead     string `json:"gitHead"`
	ID          string `json:"_id"`
	NodeVersion string `json:"_nodeVersion"`
	NpmVersion  string `json:"_npmVersion"`
	Dist        struct {
		Integrity    string `json:"integrity"`
		Shasum       string `json:"shasum"`
		Tarball      string `json:"tarball"`
		FileCount    int    `json:"fileCount"`
		UnpackedSize int    `json:"unpackedSize"`
		Signatures   []struct {
			Keyid string `json:"keyid"`
			Sig   string `json:"sig"`
		} `json:"signatures"`
	} `json:"dist"`
	NpmUser struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"_npmUser"`
	Directories struct {
	} `json:"directories"`
	Maintainers []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"maintainers"`
	NpmOperationalInternal struct {
		Host string `json:"host"`
		Tmp  string `json:"tmp"`
	} `json:"_npmOperationalInternal"`
	HasShrinkwrap bool `json:"_hasShrinkwrap"`
}

func newNpmRepository(endpoint string) *npmRepository {
	return &npmRepository{
		endpoint: endpoint,
	}
}

// TODO(jakub): fugly signature
type npmPackageReply struct {
	name      string
	version   string
	integrity string
	tarball   string
}

func (r *npmPackageReply) IndentedString(leadingWhitespace int) string {
	padding := fmt.Sprintf("%*s", leadingWhitespace, "")
	innerPadding := padding + "  " // Add 2 extra spaces

	// format each line with leadingWhitespace and 2 extra spaces
	data := padding + fmt.Sprintf("\"%s\": {\n", r.name)
	data += innerPadding + fmt.Sprintf("\"version\": \"%s\",\n", r.version)
	data += innerPadding + fmt.Sprintf("\"resolved\": \"%s\",\n", r.tarball)
	data += innerPadding + fmt.Sprintf("\"integrity\": \"%s\",", r.integrity)
	// data += padding + "},"

	return data
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

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	var pkgJson packageJson
	err = json.Unmarshal(content, &pkgJson)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return &npmPackageReply{
		name:      pkgJson.Name,
		version:   pkgJson.Version,
		integrity: pkgJson.Dist.Integrity,
		tarball:   pkgJson.Dist.Tarball,
	}, nil
}
