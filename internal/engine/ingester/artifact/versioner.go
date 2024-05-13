// Copyright 2024 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

type versioner interface {
	// Gets the available versions of a given artifact
	GetVersions(ctx context.Context) (map[string]*minderv1.ArtifactVersion, error)
}

func getVersioner(prov provifv1.Provider, a *minderv1.Artifact) (versioner, error) {
	ghprov, err := provifv1.As[provifv1.GitHub](prov)
	if err == nil {
		return &githubVersioner{
			ghCli:    ghprov,
			artifact: a,
		}, nil
	}

	ociprov, err := provifv1.As[provifv1.OCI](prov)
	if err == nil {
		return &ociVersioner{
			ocicli:   ociprov,
			artifact: a,
		}, nil
	}

	return nil, fmt.Errorf("Unable to get version lister from provider implementation")
}

type githubVersioner struct {
	ghCli    provifv1.GitHub
	artifact *minderv1.Artifact
}

// in case of the GitHub provider, a package version may be
// linked to multiple tags
func (gv *githubVersioner) GetVersions(ctx context.Context) (map[string]*minderv1.ArtifactVersion, error) {
	artifactName := url.QueryEscape(gv.artifact.GetName())
	upstreamVersions, err := gv.ghCli.GetPackageVersions(
		ctx, gv.artifact.GetOwner(), gv.artifact.GetTypeLower(), artifactName,
	)
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	out := make(map[string]*minderv1.ArtifactVersion, len(upstreamVersions))
	for _, uv := range upstreamVersions {
		tags := uv.Metadata.Container.Tags
		sort.Strings(tags)

		// only the tags and creation time is relevant to us.
		out[*uv.Name] = &minderv1.ArtifactVersion{
			Tags:      tags,
			CreatedAt: timestamppb.New(uv.CreatedAt.Time),
		}
	}

	return out, nil
}

type ociVersioner struct {
	ocicli   provifv1.OCI
	artifact *minderv1.Artifact
}

func (ov *ociVersioner) GetVersions(ctx context.Context) (map[string]*minderv1.ArtifactVersion, error) {
	tags, err := ov.ocicli.ListTags(ctx, ov.artifact.GetName())
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	out := make(map[string]*minderv1.ArtifactVersion, len(tags))
	for _, t := range tags {
		// TODO: We probably should try to surface errors while returning a subset
		// of manifests.
		man, err := ov.ocicli.GetManifest(ctx, ov.artifact.GetName(), t)
		if err != nil {
			return nil, err
		}

		// NOTE/FIXME: This is going to be a hassle as not a lot of
		// container images have the needed annotations. We'd need
		// go down to a specific image configuration (e.g. for _some_
		// architecture) to actually verify the creation date...
		// Anybody has other ideas?
		strcreated, ok := man.Annotations[v1.AnnotationCreated]
		var createdAt time.Time
		if ok {
			// TODO: Verify if this is correct
			createdAt, err = time.Parse(time.RFC3339, strcreated)
			if err != nil {
				return nil, fmt.Errorf("unable to get creation time for tag %s: %w", t, err)
			}
		} else {
			// FIXME: This is a hack
			createdAt = time.Now()
		}

		// TODO: Consider caching
		digest, err := ov.ocicli.GetDigest(ctx, ov.artifact.GetName(), t)
		if err != nil {
			return nil, fmt.Errorf("unable to get digest")
		}

		out[t] = &minderv1.ArtifactVersion{
			Tags:      []string{t},
			Sha:       digest,
			CreatedAt: timestamppb.New(createdAt),
		}
	}

	return out, nil
}
