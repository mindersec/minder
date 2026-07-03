// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
func (*OCI) CanImplement(trait minderv1.ProviderType) bool {
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

// getImage returns the remote image for the given tag of the given container.
func (o *OCI) getImage(ctx context.Context, contname, tag string) (v1.Image, error) {
	ref, err := o.getReference(contname, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get reference: %w", err)
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent))
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return img, nil
}

// GetManifest returns the manifest for the given tag of the given container in the given namespace
// for the OCI provider. It returns the manifest as a golang struct given the OCI spec.
func (o *OCI) GetManifest(ctx context.Context, contname, tag string) (*v1.Manifest, error) {
	img, err := o.getImage(ctx, contname, tag)
	if err != nil {
		return nil, err
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
		img, err := ov.getImage(ctx, artifact.GetName(), t)
		if err != nil {
			return nil, err
		}

		man, err := img.Manifest()
		if err != nil {
			return nil, fmt.Errorf("failed to get manifest for tag %s: %w", t, err)
		}

		createdAt, err := resolveCreatedAt(man, img.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("unable to get creation time for tag %s: %w", t, err)
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

// resolveCreatedAt determines the creation time of an artifact version. It
// prefers the org.opencontainers.image.created manifest annotation and, when
// that annotation is absent, falls back to the Created field of the image
// configuration, which records the real build time. The resolved value always
// reflects the image itself: a zero or Unix-epoch timestamp is a legitimate
// reproducible-build value and is preserved as-is. The image config is passed
// as a getter so that the config blob is fetched only when the annotation is
// absent, and so the resolution logic stays testable without a registry.
func resolveCreatedAt(man *v1.Manifest, configFile func() (*v1.ConfigFile, error)) (time.Time, error) {
	if strcreated, ok := man.Annotations[imgspecv1.AnnotationCreated]; ok {
		createdAt, err := time.Parse(time.RFC3339, strcreated)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing created annotation %q: %w", strcreated, err)
		}
		return createdAt, nil
	}

	cfg, err := configFile()
	if err != nil {
		return time.Time{}, fmt.Errorf("reading image config: %w", err)
	}

	return cfg.Created.Time, nil
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
