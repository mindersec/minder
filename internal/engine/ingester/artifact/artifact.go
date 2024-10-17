// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"google.golang.org/protobuf/proto"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	artif "github.com/mindersec/minder/internal/providers/artifact"
	"github.com/mindersec/minder/internal/verifier"
	"github.com/mindersec/minder/internal/verifier/sigstore/container"
	"github.com/mindersec/minder/internal/verifier/verifyif"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	prov provifv1.Provider

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

// NewArtifactDataIngest creates a new artifact rule data ingest engine
func NewArtifactDataIngest(prov provifv1.Provider) (*Ingest, error) {
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
) (*interfaces.Result, error) {
	cfg, err := configFromParams(params)
	if err != nil {
		return nil, err
	}

	artifact, ok := ent.(*pb.Artifact)
	if !ok {
		return nil, fmt.Errorf("expected Artifact, got %T", ent)
	}

	// Filter the versions of the artifact that are applicable to this rule
	applicable, err := i.getApplicableArtifactVersions(ctx, artifact, cfg)
	if err != nil {
		// Take into consideration that the returned error is later wrapped in an error of type evalerrors
		return nil, err
	}

	return &interfaces.Result{
		Object: applicable,
		// We would ideally return an artifact's digest here, but
		// the current state of the artifact ingester is actually evaluating
		// multiple artifacts at the same time. This is not ideal, ideally
		// we should evaluate one impulse at a time. This has to be fixed,
		// but for now we return the current time as the checkpoint.
		// We need to track the "impulse" that triggered the evaluation
		// so we can return the correct checkpoint.
		Checkpoint: checkpoints.NewCheckpointV1Now(),
	}, nil
}

func (i *Ingest) getApplicableArtifactVersions(
	ctx context.Context,
	artifact *pb.Artifact,
	cfg *ingesterConfig,
) ([]map[string]any, error) {
	if err := validateConfiguration(artifact, cfg); err != nil {
		return nil, err
	}

	vers, err := getVersioner(i.prov)
	if err != nil {
		return nil, err
	}

	// Get all artifact checksums filtering out those that don't apply to this rule
	checksums, err := getAndFilterArtifactVersions(ctx, cfg, vers, artifact)
	if err != nil {
		return nil, err
	}

	// Get the provenance info for all artifact versions that apply to this rule
	verificationResults, err := i.getVerificationResult(ctx, cfg, artifact, checksums)
	if err != nil {
		return nil, err
	}

	// Build the result to be returned to the rule engine as a slice of map["Verification"]any
	result := make([]map[string]any, 0, len(verificationResults))
	for _, item := range verificationResults {
		result = append(result, map[string]any{
			"Verification": item,
		})
	}

	zerolog.Ctx(ctx).Debug().Any("result", result).Msg("ingestion result")

	// Return the list of provenance info for all applicable artifact versions
	return result, nil
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
	checksums []string,
) ([]verification, error) {
	var versionResults []verification
	// Get the verifier for sigstore
	artifactVerifier, err := getVerifier(i, cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting verifier: %w", err)
	}

	// Loop through all artifact versions that apply to this rule and get the provenance info for each
	for _, artifactChecksum := range checksums {
		// Try getting provenance info for the artifact version
		results, err := artifactVerifier.Verify(ctx, verifyif.ArtifactTypeContainer,
			artifact.Owner, artifact.Name, artifactChecksum)
		if err != nil {
			// We consider err != nil as a fatal error, so we'll fail the rule evaluation here
			artifactName := container.BuildImageRef("", artifact.Owner, artifact.Name, artifactChecksum)
			zerolog.Ctx(ctx).Debug().Err(err).Str("name", artifactName).Msg("failed getting signature information")
			return nil, fmt.Errorf("failed getting signature information: %w", err)
		}
		// Loop through all results and build the verification result for each
		for _, res := range results {
			// Log a debug message in case we failed to find or verify any signature information for the artifact version
			if !res.IsSigned || !res.IsVerified {
				artifactName := container.BuildImageRef("", artifact.Owner, artifact.Name, artifactChecksum)
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
			// Append the verification result to the list
			versionResults = append(versionResults, *verResult)
		}
	}
	return versionResults, nil
}

func getVerifier(i *Ingest, cfg *ingesterConfig) (verifyif.ArtifactVerifier, error) {
	if i.artifactVerifier != nil {
		return i.artifactVerifier, nil
	}

	verifieropts := []container.AuthMethod{}
	if i.prov.CanImplement(pb.ProviderType_PROVIDER_TYPE_GITHUB) {
		ghcli, err := provifv1.As[provifv1.GitHub](i.prov)
		if err != nil {
			return nil, fmt.Errorf("unable to get github provider from provider configuration")
		}
		verifieropts = append(verifieropts, container.WithGitHubClient(ghcli))
	} else if i.prov.CanImplement(pb.ProviderType_PROVIDER_TYPE_OCI) {
		ocicli, err := provifv1.As[provifv1.OCI](i.prov)
		if err != nil {
			return nil, fmt.Errorf("unable to get oci provider from provider configuration")
		}
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
// ones that apply to the rule. Note that this returns the checksums of the
// applicable artifact versions.
func getAndFilterArtifactVersions(
	ctx context.Context,
	cfg *ingesterConfig,
	vers provifv1.ArtifactProvider,
	artifact *pb.Artifact,
) ([]string, error) {
	var res []string

	// Build a tag filter based on the configuration
	filter, err := artif.BuildFilter(cfg.Tags, cfg.TagRegex)
	if err != nil {
		return nil, fmt.Errorf("error building filter from artifact ingester config: %w", err)
	}

	// Fetch all available versions of the artifact
	upstreamVersions, err := vers.GetArtifactVersions(ctx, artifact, filter)
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	for _, version := range upstreamVersions {
		res = append(res, version.Sha)
	}

	// If no applicable artifact versions were found for this rule, we can go ahead and fail the rule evaluation here
	if len(res) == 0 {
		return nil, evalerrors.NewErrEvaluationFailed("no applicable artifact versions found")
	}

	// Return the list of applicable artifact versions, i.e. []string{"digest1", "digest2", ...}
	return res, nil
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
	if c.Extensions.Issuer != githubTokenIssuer {
		return builderURL, nil
	}

	// When handling a cert issued through GitHub actions tokens, break the identity
	// into its components. The verifier captures the git reference and the
	// the repository URI.
	if c.Extensions.SourceRepositoryURI == "" {
		return "", fmt.Errorf(
			"certificate extension dont have a SourceRepositoryURI set (oid 1.3.6.1.4.1.57264.1.5)",
		)
	}

	builderURL, _, _ = strings.Cut(builderURL, "@")
	builderURL = strings.TrimPrefix(builderURL, c.Extensions.SourceRepositoryURI)

	return builderURL, nil
}

func getVersioner(prov provifv1.Provider) (provifv1.ArtifactProvider, error) {
	ap, ok := prov.(provifv1.ArtifactProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not implement ArtifactProvider")
	}

	return ap, nil
}
