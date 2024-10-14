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
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/constants"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// OCI is the struct that contains the OCI client
type OCI struct {
	cred provifv1.Credential

	registry string
	baseURL  string
}

// New creates a new OCI client
func New(cred provifv1.Credential, registry, baseURL string) *OCI {
	return &OCI{
		cred:     cred,
		registry: registry,
		baseURL:  baseURL,
	}
}

// CanImplement returns true/false depending on whether the OCI client can implement the specified trait
func (_ *OCI) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_OCI
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
		if strings.Contains(err.Error(), "status code 404") {
			return nil, fmt.Errorf("no such repository: %s", repo)
		}
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

			outtags = append(outtags, tag)
		}
	}

	return outtags, nil
}

// GetDigest returns the digest for a given tag
func (o *OCI) GetDigest(ctx context.Context, contname, tag string) (string, error) {
	ref, err := o.getReference(contname, tag)
	if err != nil {
		return "", fmt.Errorf("failed to get reference: %w", err)
	}

	return getDigestFromRef(ctx, ref)
}

// GetReferrer returns the referrer for the given tag of the given container in the given namespace
// for the OCI provider. It returns the referrer as a golang struct given the OCI spec.
func (o *OCI) GetReferrer(ctx context.Context, contname, tag, artifactType string) (any, error) {
	ref, err := o.getReference(contname, tag)
	if err != nil {
		return "", fmt.Errorf("failed to get reference: %w", err)
	}

	dig, err := getDigestFromRef(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("failed to get digest: %w", err)
	}

	digname, err := name.NewDigest(fmt.Sprintf("%s@%s", ref.Context().RepositoryStr(), dig))
	if err != nil {
		return "", fmt.Errorf("failed to get digest name: %w", err)
	}

	refer, err := remote.Referrers(digname,
		remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent),
		remote.WithFilter("artifactType", artifactType))
	if err != nil {
		return "", fmt.Errorf("failed to get referrer: %w", err)
	}

	return refer, nil
}

// GetManifest returns the manifest for the given tag of the given container in the given namespace
// for the OCI provider. It returns the manifest as a golang struct given the OCI spec.
func (o *OCI) GetManifest(ctx context.Context, contname, tag string) (*v1.Manifest, error) {
	ref, err := o.getReference(contname, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get reference: %w", err)
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent))
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	man, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	return man, nil
}

// GetRegistry returns the registry name
func (o *OCI) GetRegistry() string {
	return o.registry
}

// GetAuthenticator returns the authenticator for the OCI provider
func (o *OCI) GetAuthenticator() (authn.Authenticator, error) {
	if o.cred == nil {
		return authn.Anonymous, nil
	}

	oauth2cred, ok := o.cred.(provifv1.OAuth2TokenCredential)
	if !ok {
		return nil, fmt.Errorf("credential is not an OAuth2 token credential")
	}
	s := oauth2cred.GetAsOAuth2TokenSource()
	t, err := s.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return &authn.Bearer{Token: t.AccessToken}, nil
}

// GetArtifactVersions returns the artifact versions for the given artifact
func (ov *OCI) GetArtifactVersions(
	ctx context.Context,
	artifact *minderv1.Artifact,
	filter provifv1.GetArtifactVersionsFilter,
) ([]*minderv1.ArtifactVersion, error) {
	tags, err := ov.ListTags(ctx, artifact.GetName())
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	out := make([]*minderv1.ArtifactVersion, 0, len(tags))
	for _, t := range tags {
		// TODO: We probably should try to surface errors while returning a subset
		// of manifests.
		man, err := ov.GetManifest(ctx, artifact.GetName(), t)
		if err != nil {
			return nil, err
		}

		// NOTE/FIXME: This is going to be a hassle as not a lot of
		// container images have the needed annotations. We'd need
		// go down to a specific image configuration (e.g. for _some_
		// architecture) to actually verify the creation date...
		// Anybody has other ideas?
		strcreated, ok := man.Annotations[imgspecv1.AnnotationCreated]
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

		if err := filter.IsSkippable(createdAt, []string{t}); err != nil {
			zerolog.Ctx(ctx).Debug().Str("name", artifact.GetName()).Strs("tags", tags).
				Str("reason", err.Error()).Msg("skipping artifact version")
			continue
		}

		// TODO: Consider caching
		digest, err := ov.GetDigest(ctx, artifact.GetName(), t)
		if err != nil {
			return nil, fmt.Errorf("unable to get digest")
		}

		out = append(out, &minderv1.ArtifactVersion{
			Tags:      []string{t},
			Sha:       digest,
			CreatedAt: timestamppb.New(createdAt),
		})
	}

	return out, nil
}

// getReferenceString returns the reference string for a given container name and tag
func (o *OCI) getReferenceString(contname, tag string) string {
	return fmt.Sprintf("%s/%s:%s", o.baseURL, contname, tag)
}

// getReference returns the reference for a given container name and tag
func (o *OCI) getReference(contname, tag string) (name.Reference, error) {
	ref, err := name.ParseReference(o.getReferenceString(contname, tag))
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	return ref, nil
}

// getDigestFromRef returns the digest of a container image reference
// TODO: Move this to a more appropriate location
// TODO: Implement authentication
// TODO: Implement authentication
func getDigestFromRef(ctx context.Context, ref name.Reference) (string, error) {
	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent))
	if err != nil {
		return "", fmt.Errorf("failed to get image: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("failed to get digest: %w", err)
	}

	return digest.String(), nil
}
