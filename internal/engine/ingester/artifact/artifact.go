// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"google.golang.org/protobuf/proto"

	artif "github.com/mindersec/minder/internal/providers/artifact"
	"github.com/mindersec/minder/internal/verifier"
	"github.com/mindersec/minder/internal/verifier/sigstore/container"
	"github.com/mindersec/minder/internal/verifier/verifyif"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	evalerrors "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/entities/v1/checkpoints"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// ArtifactRuleDataIngestType is the type of the artifact rule data ingest engine
	ArtifactRuleDataIngestType = "artifact"

	// githubTokenIssuer is the issuer stamped into sigstore certs
	// when authenticating through GitHub tokens
	//nolint : gosec // Not an embedded credential
	githubTokenIssuer = "https://token.actions.githubusercontent.com"
)

// Ingest is the engine for a rule type that uses artifact data ingest
// Implements enginer.ingester.Ingester
type Ingest struct {
	prov interfaces.Provider

	// artifactVerifier is the verifier for sigstore. It's only used in the Ingest method
	// but we store it in the Ingest structure to allow tests to set a custom artifactVerifier
	artifactVerifier verifyif.ArtifactVerifier
}

type verification struct {
	IsSigned          bool                 `json:"is_signed"`
	IsVerified        bool                 `json:"is_verified"`
	Repository        string               `json:"repository"`
	Branch            string               `json:"branch"`
	SignerIdentity    string               `json:"signer_identity"`
	RunnerEnvironment string               `json:"runner_environment"`
	CertIssuer        string               `json:"cert_issuer"`
	Attestation       *verifiedAttestation `json:"attestation,omitempty"`
}

type verifiedAttestation struct {
	PredicateType string `json:"predicate_type,omitempty"`
	Predicate     any    `json:"predicate,omitempty"`
}

// imageRef locates an artifact version in its registry, independent of
// verification status.
type imageRef struct {
	Registry   string   `json:"registry,omitempty"`
	Repository string   `json:"repository"`
	Tags       []string `json:"tags,omitempty"`
	Digest     string   `json:"digest"`
}

// imageInfo is the typed result per artifact version. Fields are
// capitalized (not JSON-tagged) so Rego sees "Identity", "Verification",
// "Manifest" as keys.
type imageInfo struct {
	Verification verification
	Identity     imageRef
	Manifest     *provifv1.RawManifest
}

// NewArtifactDataIngest creates a new artifact rule data ingest engine
func NewArtifactDataIngest(prov interfaces.Provider) (*Ingest, error) {
	return &Ingest{
		prov: prov,
	}, nil
}

// GetType returns the type of the artifact rule data ingest engine
func (*Ingest) GetType() string {
	return ArtifactRuleDataIngestType
}

// GetConfig returns the config for the artifact rule data ingest engine
func (*Ingest) GetConfig() proto.Message {
	return nil
}

// Ingest checks the passed in artifact, makes sure it is applicable to the current rule
// and if it is, returns the appropriately marshalled data.
func (i *Ingest) Ingest(
	ctx context.Context,
	ent proto.Message,
	params map[string]any,
) (*interfaces.Ingested, error) {
	cfg, err := configFromParams(params)
	if err != nil {
		return nil, err
	}

	artifact, ok := ent.(*pb.Artifact)
	if !ok {
		return nil, fmt.Errorf("expected Artifact, got %T", ent)
	}

	// Filter the versions of the artifact that are applicable to this rule
	applicable, digest, err := i.getApplicableArtifactVersions(ctx, artifact, cfg)
	if err != nil {
		// Take into consideration that the returned error is later wrapped in an error of type evalerrors
		return nil, err
	}

	// A repo:tag entity (DockerHub, Quay) evaluates exactly one digest, so the
	// checkpoint records it. Other artifacts (e.g. GHCR packages) may evaluate
	// several digests at once, so they only record the time.
	checkpoint := checkpoints.NewCheckpointV1Now()
	if digest != "" {
		checkpoint = checkpoint.WithDigest(digest)
	}

	return &interfaces.Ingested{
		Object:     applicable,
		Checkpoint: checkpoint,
	}, nil
}

// getApplicableArtifactVersions returns the provenance info for the artifact
// versions that apply to this rule. When the entity is a single repo:tag, it
// also returns the digest that tag resolved to; otherwise digest is empty.
func (i *Ingest) getApplicableArtifactVersions(
	ctx context.Context,
	artifact *pb.Artifact,
	cfg *ingesterConfig,
) ([]imageInfo, string, error) {
	if err := validateConfiguration(artifact, cfg); err != nil {
		return nil, "", err
	}

	var versions []*pb.ArtifactVersion
	var digest string
	if tagProv, isTagProv := i.prov.(provifv1.OCITagVersionProvider); isTagProv {
		// The provider owns its artifact identity format, so it extracts the tag.
		if tag, hasTag := tagProv.ArtifactTag(artifact); hasTag {
			// The entity is one repo:tag, so evaluate only the version that tag points to now.
			version, err := getOwnTagVersion(ctx, cfg, tagProv, artifact, tag)
			if err != nil {
				return nil, "", err
			}
			versions = []*pb.ArtifactVersion{version}
			digest = version.GetSha()
		}
	}
	if len(versions) == 0 {
		// Couldn't extract tag-specific versions, so list the repository's
		// versions and keep those that apply to this rule.
		vers, err := interfaces.As[provifv1.ArtifactProvider](i.prov)
		if err != nil {
			return nil, "", err
		}

		// Get all artifact versions filtering out those that don't apply to this rule
		versions, err = getAndFilterArtifactVersions(ctx, cfg, vers, artifact)
		if err != nil {
			return nil, "", err
		}
	}

	// Get the identity and provenance info for all artifact versions that apply to this rule
	result, err := i.getVerificationResult(ctx, cfg, artifact, versions)
	if err != nil {
		return nil, "", err
	}

	zerolog.Ctx(ctx).Debug().Any("result", result).Msg("ingestion result")

	// Return the list of provenance info for all applicable artifact versions
	return result, digest, nil
}

// getOwnTagVersion resolves the entity's own tag to its current version and
// checks it against the rule's filters. A tag that does not match the rule's
// tags or tag_regex skips the rule silently, since the rule targets other tags.
// Any other filter rejection (e.g. the image is older than the retention
// period) fails the evaluation, as it does on the repository-wide path.
//
// The entity carries a single tag, so a tags list matches when the entity's
// tag is any of the listed tags: tags: [a, b] applies to both a and b.
func getOwnTagVersion(
	ctx context.Context,
	cfg *ingesterConfig,
	prov provifv1.OCITagVersionProvider,
	artifact *pb.Artifact,
	tag string,
) (*pb.ArtifactVersion, error) {
	filter, err := artif.BuildFilter(cfg.Tags, cfg.TagRegex, artif.MatchAnyTag)
	if err != nil {
		return nil, fmt.Errorf("error building filter from artifact ingester config: %w", err)
	}

	version, err := prov.GetArtifactVersion(ctx, artifact.GetName(), tag)
	if err != nil {
		// Intentionally a failure, unlike the repository-wide path, which just
		// drops a deleted tag from its listing: here the deleted tag is the
		// entity itself, so it fails every evaluation until the entity is
		// removed. Reconciling registered tags against the registry is the
		// follow-up in #7090 (Reminder coverage for artifact entities).
		if errors.Is(err, provifv1.ErrArtifactTagNotFound) {
			return nil, evalerrors.NewErrEvaluationFailed("artifact tag %s:%s no longer exists", artifact.GetName(), tag)
		}
		return nil, fmt.Errorf("error retrieving artifact version for tag %s: %w", tag, err)
	}

	if err := filter.IsSkippable(version.GetCreatedAt().AsTime(), version.GetTags()); err != nil {
		if errors.Is(err, artif.ErrTagsDoNotMatch) {
			return nil, evalerrors.NewErrEvaluationSkipSilently("artifact tag %s does not match the rule's tags", tag)
		}
		return nil, evalerrors.NewErrEvaluationFailed("artifact version not applicable: %s", err)
	}

	return version, nil
}

func validateConfiguration(
	artifact *pb.Artifact,
	cfg *ingesterConfig,
) error {
	// Make sure the artifact type matches
	if newArtifactIngestType(artifact.Type) != cfg.Type {
		return evalerrors.NewErrEvaluationSkipSilently("artifact type mismatch")
	}

	if cfg.Type != artifactTypeContainer {
		return evalerrors.NewErrEvaluationSkipSilently("only container artifacts are supported at the moment")
	}

	// If a name is specified, make sure it matches
	if cfg.Name != "" && cfg.Name != artifact.Name {
		return evalerrors.NewErrEvaluationSkipSilently("artifact name mismatch")
	}

	return nil
}

func (i *Ingest) getVerificationResult(
	ctx context.Context,
	cfg *ingesterConfig,
	artifact *pb.Artifact,
	versions []*pb.ArtifactVersion,
) ([]imageInfo, error) {
	var results []imageInfo
	// Get the verifier for sigstore
	artifactVerifier, err := getVerifier(i, cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting verifier: %w", err)
	}

	registry := getRegistryForProvider(i.prov)
	repository := buildRepository(artifact)
	// Raw-manifest needs the OCI interface (DockerHub, Quay); GHCR doesn't
	// implement it yet (#6778).
	ocicli, _ := interfaces.As[provifv1.OCI](i.prov)

	// Loop through all artifact versions that apply to this rule and get the provenance info for each
	for _, version := range versions {
		if version == nil {
			continue
		}
		identity := imageRef{
			Registry:   registry,
			Repository: repository,
			Tags:       version.Tags,
			Digest:     version.Sha,
		}

		var manifest *provifv1.RawManifest
		if ocicli != nil {
			manifest, err = ocicli.GetRawManifest(ctx, artifact.GetName(), version.GetSha())
			if err != nil {
				zerolog.Ctx(ctx).Info().Err(err).Str("name", artifact.GetName()).
					Msg("failed to fetch raw manifest, continuing without it")
				manifest = nil
			}
		}

		// Try getting provenance info for the artifact version
		verResults, err := artifactVerifier.Verify(ctx, verifyif.ArtifactTypeContainer,
			artifact.Owner, artifact.Name, version.Sha)
		if err != nil {
			// We consider err != nil as a fatal error, so we'll fail the rule evaluation here
			artifactName := container.BuildImageRef("", artifact.Owner, artifact.Name, version.Sha)
			zerolog.Ctx(ctx).Debug().Err(err).Str("name", artifactName).Msg("failed getting signature information")
			return nil, fmt.Errorf("failed getting signature information: %w", err)
		}
		// Loop through all results and build the verification result for each
		for _, res := range verResults {
			// Log a debug message in case we failed to find or verify any signature information for the artifact version
			if !res.IsSigned || !res.IsVerified {
				artifactName := container.BuildImageRef("", artifact.Owner, artifact.Name, version.Sha)
				zerolog.Ctx(ctx).Debug().Str("name", artifactName).Msg("failed to find or verify signature information")
			}

			// Begin building the verification result
			verResult := &verification{
				IsSigned:   res.IsSigned,
				IsVerified: res.IsVerified,
			}

			// If we got verified provenance info for the artifact version, populate the rest of the verification result
			if res.IsVerified {
				siIdentity, err := signerIdentityFromCertificate(res.Signature.Certificate)
				if err != nil {
					zerolog.Ctx(ctx).Err(err).Msg("error parsing signer identity")
				}

				verResult.Repository = res.Signature.Certificate.SourceRepositoryURI
				verResult.Branch = branchFromRef(res.Signature.Certificate.SourceRepositoryRef)
				verResult.SignerIdentity = siIdentity
				verResult.RunnerEnvironment = res.Signature.Certificate.RunnerEnvironment
				verResult.CertIssuer = res.Signature.Certificate.Issuer
			}

			if res.Statement != nil {
				verResult.Attestation = &verifiedAttestation{
					PredicateType: res.Statement.PredicateType,
					Predicate:     res.Statement.Predicate,
				}
			}
			results = append(results, imageInfo{
				Identity:     identity,
				Verification: *verResult,
				Manifest:     manifest,
			})
		}
	}
	return results, nil
}

// getRegistryForProvider returns the registry hostname. OCI providers
// report their own; GitHub falls back to container.GHCRRegistry. Anything
// else returns empty.
func getRegistryForProvider(prov interfaces.Provider) string {
	if ocicli, err := interfaces.As[provifv1.OCI](prov); err == nil {
		return ocicli.GetRegistry()
	}
	if _, err := interfaces.As[provifv1.GitHub](prov); err == nil {
		return container.GHCRRegistry
	}
	return ""
}

// buildRepository returns the artifact's registry path as owner/name.
// Owner is the GHCR package owner or the DockerHub/Quay namespace; if it is
// empty, the repository is just the name.
func buildRepository(artifact *pb.Artifact) string {
	if artifact.Owner == "" {
		return artifact.Name
	}
	return artifact.Owner + "/" + artifact.Name
}

func getVerifier(i *Ingest, cfg *ingesterConfig) (verifyif.ArtifactVerifier, error) {
	if i.artifactVerifier != nil {
		return i.artifactVerifier, nil
	}

	verifieropts := []container.AuthMethod{}
	if ghcli, err := interfaces.As[provifv1.GitHub](i.prov); err == nil {
		verifieropts = append(verifieropts, container.WithGitHubClient(ghcli))
	} else if ocicli, err := interfaces.As[provifv1.OCI](i.prov); err == nil {
		cauthn, err := ocicli.GetAuthenticator()
		if err != nil {
			return nil, fmt.Errorf("unable to get oci authenticator: %w", err)
		}
		verifieropts = append(verifieropts, container.WithRegistry(ocicli.GetRegistry()),
			container.WithAuthenticator(cauthn))
	}

	artifactVerifier, err := verifier.NewVerifier(
		verifier.VerifierSigstore,
		cfg.Sigstore,
		verifieropts...,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting sigstore verifier: %w", err)
	}

	return artifactVerifier, nil
}

// getAndFilterArtifactVersions fetches the available versions and filters the
// ones that apply to the rule.
func getAndFilterArtifactVersions(
	ctx context.Context,
	cfg *ingesterConfig,
	vers provifv1.ArtifactProvider,
	artifact *pb.Artifact,
) ([]*pb.ArtifactVersion, error) {
	// Build a tag filter based on the configuration
	filter, err := artif.BuildFilter(cfg.Tags, cfg.TagRegex, artif.MatchAllTags)
	if err != nil {
		return nil, fmt.Errorf("error building filter from artifact ingester config: %w", err)
	}

	// Fetch all available versions of the artifact, already filtered by the provider
	upstreamVersions, err := vers.GetArtifactVersions(ctx, artifact, filter)
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	// If no applicable artifact versions were found for this rule, we can go ahead and fail the rule evaluation here
	if len(upstreamVersions) == 0 {
		return nil, evalerrors.NewErrEvaluationFailed("no applicable artifact versions found")
	}

	return upstreamVersions, nil
}

var (
	// ArtifactTypeContainerRetentionPeriod represents the retention period for container artifacts
	ArtifactTypeContainerRetentionPeriod = time.Now().AddDate(0, -6, 0)
)

func branchFromRef(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") {
		return ref[len("refs/heads/"):]
	}

	return ""
}

// signerIdentityFromCertificate returns the signer identity. When the identity
// is a URI (from the BuildSignerURI extension or the cert SAN), we return only
// the URI path component. We split it this way to ensure we can make rules
// more generalizable (applicable to the same path regardless of the repo for example).
func signerIdentityFromCertificate(c *certificate.Summary) (string, error) {
	var builderURL string

	if c.SubjectAlternativeName == "" {
		return "", fmt.Errorf("certificate has no signer identity in SAN (is it a fulcio cert?)")
	}

	switch {
	case c.SubjectAlternativeName != "":
		builderURL = c.SubjectAlternativeName
	default:
		// Return the SAN in the cert as a last resort. This handles the case when
		// we don't have a signer identity but also when the SAN is an email
		// when a user authenticated using an OIDC provider or a SPIFFE ID.
		// Any other SAN types are returned verbatim
		return c.SubjectAlternativeName, nil
	}

	// Any signer identity not issued by github actions is returned verbatim
	if c.Issuer != githubTokenIssuer {
		return builderURL, nil
	}

	// When handling a cert issued through GitHub actions tokens, break the identity
	// into its components. The verifier captures the git reference and the
	// the repository URI.
	if c.SourceRepositoryURI == "" {
		return "", fmt.Errorf(
			"certificate extension dont have a SourceRepositoryURI set (oid 1.3.6.1.4.1.57264.1.5)",
		)
	}

	builderURL, _, _ = strings.Cut(builderURL, "@")
	builderURL = strings.TrimPrefix(builderURL, c.SourceRepositoryURI)

	return builderURL, nil
}
