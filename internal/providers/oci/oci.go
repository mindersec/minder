// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package oci provides a client for interacting with OCI registries
package oci

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
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

// GetRawManifest returns the manifest or image index exactly as returned by
// the registry, fetched by digest since digests are canonical and always
// available on an artifact version, unlike tags. Unlike GetManifest, this
// does not resolve a multi-platform index down to a single platform.
func (o *OCI) GetRawManifest(ctx context.Context, contname, digest string) (*provifv1.RawManifest, error) {
	ref, err := name.NewDigest(fmt.Sprintf("%s/%s@%s", o.baseURL, contname, digest))
	if err != nil {
		return nil, fmt.Errorf("failed to parse digest reference: %w", err)
	}

	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent))
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	return &provifv1.RawManifest{
		Descriptor: desc.Descriptor,
		Content:    desc.Manifest,
	}, nil
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
		ref, err := ov.getReference(artifact.GetName(), t)
		if err != nil {
			return nil, fmt.Errorf("failed to get reference for tag %s: %w", t, err)
		}

		desc, err := remote.Get(ref,
			remote.WithContext(ctx),
			remote.WithUserAgent(constants.ServerUserAgent),
		)
		if err != nil {
			if isNotFound(err) {
				zerolog.Ctx(ctx).Info().
					Str("name", artifact.GetName()).
					Str("tag", t).
					Msg("skipping artifact version: tag no longer exists")
				continue
			}
			return nil, fmt.Errorf("failed to get descriptor for tag %s: %w", t, err)
		}

		var createdAt time.Time
		//nolint:exhaustive // only index types need special handling; all other media types are treated as single-platform images
		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			createdAt, err = resolveIndexCreatedAt(ctx, desc, t)
			if err != nil {
				return nil, err
			}

		default:
			img, err := desc.Image()
			if err != nil {
				return nil, fmt.Errorf("failed to get image for tag %s: %w", t, err)
			}
			man, err := img.Manifest()
			if err != nil {
				return nil, fmt.Errorf("failed to get manifest for tag %s: %w", t, err)
			}
			createdAt, err = resolveCreatedAt(man, img.ConfigFile)
			if err != nil {
				return nil, fmt.Errorf("unable to get creation time for tag %s: %w", t, err)
			}
		}

		if err := filter.IsSkippable(createdAt, []string{t}); err != nil {
			zerolog.Ctx(ctx).Debug().
				Str("name", artifact.GetName()).
				Str("tag", t).
				Str("reason", err.Error()).
				Msg("skipping artifact version")
			continue
		}

		out = append(out, &minderv1.ArtifactVersion{
			Tags:      []string{t},
			Sha:       desc.Digest.String(),
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

// isNotFound returns true when err is a registry 404 / MANIFEST_UNKNOWN,
// meaning the tag was deleted after it appeared in the listing.
func isNotFound(err error) bool {
	var terr *transport.Error
	return errors.As(err, &terr) && terr.StatusCode == http.StatusNotFound
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
// TODO(#6972): Implement authentication
func getDigestFromRef(ctx context.Context, ref name.Reference) (string, error) {
	// Use the tag's top-level digest (the index for multi-arch tags), not a platform image's.
	desc, err := remote.Get(ref, remote.WithContext(ctx), remote.WithUserAgent(constants.ServerUserAgent))
	if err != nil {
		return "", fmt.Errorf("failed to get descriptor: %w", err)
	}

	return desc.Digest.String(), nil
}

// resolveIndexCreatedAt returns the creation time for a multi-platform image
// index. It prefers the index-level org.opencontainers.image.created annotation
// and falls back to the first child image timestamp when the annotation is absent.
func resolveIndexCreatedAt(ctx context.Context, desc *remote.Descriptor, tag string) (time.Time, error) {
	idx, err := desc.ImageIndex()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read image index for tag %s: %w", tag, err)
	}
	idxManifest, err := idx.IndexManifest()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read index manifest for tag %s: %w", tag, err)
	}
	strcreated, ok := idxManifest.Annotations[imgspecv1.AnnotationCreated]
	if ok {
		createdAt, err := time.Parse(time.RFC3339, strcreated)
		if err != nil {
			return time.Time{}, fmt.Errorf("parsing created annotation for tag %s: %w", tag, err)
		}
		return createdAt, nil
	}
	return firstChildCreatedAt(ctx, idx)
}

// firstChildCreatedAt returns the creation timestamp of the first child image
// that has a non-zero created field. All platforms in a multi-arch index are
// typically built in the same CI run so their timestamps are effectively equal;
// stopping at the first valid one avoids unnecessary registry calls.
func firstChildCreatedAt(ctx context.Context, idx v1.ImageIndex) (time.Time, error) {
	idxManifest, err := idx.IndexManifest()
	if err != nil {
		return time.Time{}, fmt.Errorf("reading index manifest: %w", err)
	}

	for _, desc := range idxManifest.Manifests {
		img, err := idx.Image(desc.Digest)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Str("digest", desc.Digest.String()).
				Msg("skipping child image when determining index creation time")
			continue
		}
		cfg, err := img.ConfigFile()
		if err != nil {
			zerolog.Ctx(ctx).Warn().Str("digest", desc.Digest.String()).
				Msg("skipping child config when determining index creation time")
			continue
		}
		if !cfg.Created.IsZero() {
			return cfg.Created.Time, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to determine creation time from any child image in index")
}
