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

// Package oci provides a client for interacting with OCI registries
package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// OCI is the struct that contains the OCI client
type OCI struct {
	cred provifv1.Credential

	baseURL string
}

// Ensure that the OCI client implements the OCI interface
var _ provifv1.OCI = (*OCI)(nil)

// New creates a new OCI client
func New(cred provifv1.Credential, baseURL string) *OCI {
	return &OCI{
		cred:    cred,
		baseURL: baseURL,
	}
}

// ListTags lists the tags for a given container
func (o *OCI) ListTags(ctx context.Context, contname string) ([]string, error) {
	// join base name with contname
	// TODO make this more robust
	src := fmt.Sprintf("%s/%s", o.baseURL, contname)
	repo, err := name.NewRepository(src)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %w", src, err)
	}

	puller, err := remote.NewPuller()
	if err != nil {
		return nil, err
	}

	lister, err := puller.Lister(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("reading tags for %s: %w", repo, err)
	}

	var outtags []string

	for lister.HasNext() {
		tags, err := lister.Next(ctx)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags.Tags {
			// Should we be ommiting digest tags?
			if strings.HasPrefix(tag, "sha256-") {
				continue
			}

			outtags = append(outtags, repo.Tag(tag).String())
		}
	}

	return outtags, nil
}
