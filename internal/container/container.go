// // Copyright 2023 Stacklok, Inc
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //	http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

// Package container provides a client for interacting with container images
package container

import (
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerregistry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/rs/zerolog"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	cosign "github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	oci "github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// REGISTRY is the default registry
var REGISTRY = "ghcr.io"

var errManifestNotFound = errors.New("no such manifest found")

type githubAuthenticator struct{ username, password string }

func (g githubAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: g.username,
		Password: g.password,
	}, nil
}

func artifactImageRef(registry, owner, artifactName, versionName string) string {
	if registry == "" {
		registry = REGISTRY
	}

	// NOTE(jaosorior): The owner can't be upper-cased.
	owner = strings.ToLower(owner)

	return fmt.Sprintf("%s/%s/%s@%s", registry, owner, artifactName, versionName)
}

var (
	// ErrSigValidation is returned when signature validation fails
	ErrSigValidation = errors.New("error validating signature")
	// ErrProtoParse is returned when parsing the protobuf representation of signature or workflow fails
	ErrProtoParse = errors.New("error getting bytes from proto")
)

// GetArtifactSignatureAndWorkflowInfo returns the signature and workflow information as raw JSON for a given artifact
func GetArtifactSignatureAndWorkflowInfo(
	ctx context.Context,
	cli provifv1.Provider,
	ownerLogin, artifactName, versionName string,
) (sigInfo json.RawMessage, workflowInfo json.RawMessage, err error) {
	imageRef := artifactImageRef("", ownerLogin, artifactName, versionName)
	signatureVerification, githubWorkflow, validateErr := ValidateSignature(ctx,
		cli.GetToken(), ownerLogin, imageRef)
	if validateErr != nil {
		err = fmt.Errorf("%w: errorvalidating image-ref %s: %s", ErrSigValidation,
			imageRef, validateErr.Error())
		return
	}

	sig, parseErr := util.GetBytesFromProto(signatureVerification)
	if parseErr != nil {
		sigInfo = json.RawMessage("{}")
		err = ErrProtoParse
		// don't return early, let the caller handle the error after parsing the rest
	} else {
		sigInfo = json.RawMessage(sig)
	}

	work, parseErr := util.GetBytesFromProto(githubWorkflow)
	if parseErr != nil {
		err = ErrProtoParse
		workflowInfo = json.RawMessage("{}")
	} else {
		workflowInfo = json.RawMessage(work)
	}

	return
}

func extractAndValidateSignature(
	ctx context.Context,
	package_url, package_owner, accessToken string,
	manifest containerregistry.Manifest,
	signatureVerification *pb.SignatureVerification,
	githubWorkflow *pb.GithubWorkflow,
) {
	signatureVerification.IsSigned = true
	identity, issuer, err := ExtractIdentityFromCertificate(manifest)
	if err == nil && identity != "" && issuer != "" {
		signatureVerification.CertIdentity = &identity
		signatureVerification.CertIssuer = &issuer

		// we have issuer and identity, we can verify the image
		verified, bundleVerified, imageKeys, err := VerifyFromIdentity(ctx, package_url, package_owner, accessToken, identity, issuer)
		if err == nil {
			// we can add information for the image
			signatureVerification.IsVerified = verified
			signatureVerification.IsBundleVerified = bundleVerified
			rekorLogID, err := readValueAs[string](imageKeys, "RekorLogID")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				signatureVerification.RekorLogId = proto.String(rekorLogID)
			}

			rekorLogIndex, err := readValueAs[int64](imageKeys, "RekorLogIndex")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				log_index := int32(rekorLogIndex)
				signatureVerification.RekorLogIndex = &log_index
			}

			signatureTime, err := readValueAs[int64](imageKeys, "SignatureTime")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				signatureVerification.SignatureTime = timestamppb.New(time.Unix(signatureTime, 0))
			}

			workflowName, err := readValueAs[string](imageKeys, "WorkflowName")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				githubWorkflow.Name = workflowName
			}

			workflowRepository, err := readValueAs[string](imageKeys, "WorkflowRepository")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				githubWorkflow.Repository = workflowRepository
			}

			workflowSha, err := readValueAs[string](imageKeys, "WorkflowSha")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				githubWorkflow.CommitSha = workflowSha
			}

			workflowTrigger, err := readValueAs[string](imageKeys, "WorkflowTrigger")
			if err != nil {
				log.Printf("error parsing value from imageKeys: %v", err)
			} else {
				githubWorkflow.Trigger = workflowTrigger
			}
		} else {
			log.Printf("error verifying image: %v", err)
		}
	} else {
		log.Printf("error extracting identity from certificate: %v", err)
	}
}

// readValueAs gets the typed value from the given accessor. Returns an error when the accessor
// doesn't find anything or when the type assertion fails.
func readValueAs[T any](data map[string]any, key string) (T, error) {
	var out T

	value, ok := data[key]
	if !ok {
		return out, fmt.Errorf("key %s not found in map", key)
	}

	if value == nil {
		return out, fmt.Errorf("no value for key %s in map", key)
	}

	out, ok = value.(T)
	if !ok {
		return out, fmt.Errorf("could not type assert %v (value of key %v) to %v", value, key, reflect.TypeOf(out))
	}

	return out, nil
}

// ValidateSignature returns information about signature validation of a package
func ValidateSignature(ctx context.Context, accessToken string, package_owner string,
	package_url string) (*pb.SignatureVerification, *pb.GithubWorkflow, error) {
	baseRef, err := name.ParseReference(package_url)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing image path: %w", err)
	}

	// need to retrieve package by name
	signature_verification := &pb.SignatureVerification{
		IsVerified:       false,
		IsSigned:         false,
		IsBundleVerified: false,
	}
	github_workflow := &pb.GithubWorkflow{}

	// get information about signature
	signature, err := GetSignatureTag(baseRef, package_owner, accessToken)

	// if there is a signature, we can move forward and retrieve details
	if err == nil && signature != nil {
		// we need to extract manifest from the signature
		manifest, err := GetImageManifest(signature, package_owner, accessToken)
		if errors.Is(err, errManifestNotFound) {
			zerolog.Ctx(ctx).Info().
				Str("packageUrl", package_url).
				Msg("no manifest found")
		} else if err != nil {
			log.Printf("error getting manifest: %v", err)
		} else if manifest.Layers != nil {
			extractAndValidateSignature(
				ctx,
				package_url,
				package_owner,
				accessToken,
				manifest,
				signature_verification,
				github_workflow)
		}
	} else {
		log.Printf("error getting signature tag: %v", err)
	}
	return signature_verification, github_workflow, nil

}

// GetSignatureTag returns the signature tag for a given image if exists
func GetSignatureTag(imageRef name.Reference, username string, token string) (name.Reference, error) {
	auth := githubAuthenticator{username, token}
	dstRef, err := ociremote.SignatureTag(imageRef, ociremote.WithRemoteOptions(remote.WithAuth(auth)))
	if err != nil {
		return nil, fmt.Errorf("error getting signature tag: %w", err)
	}
	return dstRef, nil
}

// GetImageManifest returns the manifest for the given image
func GetImageManifest(imageRef name.Reference, username string, token string) (containerregistry.Manifest, error) {
	auth := githubAuthenticator{username, token}
	img, err := remote.Image(imageRef, remote.WithAuth(auth))
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) {
			for _, err := range transportErr.Errors {
				if err.Code == transport.ManifestUnknownErrorCode {
					return containerregistry.Manifest{}, errManifestNotFound
				}
			}
		}
		return containerregistry.Manifest{}, fmt.Errorf("error getting image: %w", err)
	}
	manifest, err := img.Manifest()
	if err != nil {
		return containerregistry.Manifest{}, fmt.Errorf("error getting manifest: %w", err)
	}
	return *manifest, nil
}

// ExtractIdentityFromCertificate returns the identity and issuer from the certificate
func ExtractIdentityFromCertificate(manifest containerregistry.Manifest) (string, string, error) {
	identity := ""
	issuer := ""
	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.dev.cosign.simplesigning.v1+json" {
			//signature_digest := layer.Digest.String()
			//signature := layer.Annotations["dev.cosignproject.cosign/signature"]
			cert := layer.Annotations["dev.sigstore.cosign/certificate"]
			// Decode the PEM-encoded certificate
			pemBlock, _ := pem.Decode([]byte(cert))
			if pemBlock == nil || pemBlock.Type != "CERTIFICATE" {
				return "", "", fmt.Errorf("failed to decode PEM certificate")
			}

			// Parse the X.509 certificate
			certObj, err := x509.ParseCertificate(pemBlock.Bytes)
			if err != nil {
				return "", "", fmt.Errorf("error parsing certificate: %w", err)
			}
			for _, uri := range certObj.URIs {
				identity = uri.String()
				break
			}

			// now parse the issuer
			customOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

			// Search for the custom OID in the certificate extensions
			var customExtensionValue []byte
			for _, ext := range certObj.Extensions {
				if ext.Id.Equal(customOID) {
					customExtensionValue = ext.Value
					issuer = string(customExtensionValue)
					return identity, issuer, nil
				}
			}
			break
		}
	}

	return identity, "", nil
}

// GetKeysFromVerified returns the keys from the verified signatures
// nolint: gocyclo
func GetKeysFromVerified(verified []oci.Signature) ([]payload.SimpleContainerImage, error) {
	var outputKeys []payload.SimpleContainerImage
	for _, sig := range verified {
		p, err := sig.Payload()
		if err != nil {
			return nil, fmt.Errorf("error fetching payload: %w", err)
		}
		ss := payload.SimpleContainerImage{}
		if err := json.Unmarshal(p, &ss); err != nil {
			return nil, fmt.Errorf("error decoding the payload: %w", err)
		}
		if cert, err := sig.Cert(); err == nil && cert != nil {
			ce := cosign.CertExtensions{Cert: cert}
			if ss.Optional == nil {
				ss.Optional = make(map[string]interface{})
			}
			ss.Optional["Subject"] = sigs.CertSubject(cert)
			if issuerURL := ce.GetIssuer(); issuerURL != "" {
				ss.Optional["Issuer"] = issuerURL
				ss.Optional[cosign.CertExtensionOIDCIssuer] = issuerURL
			}
			if githubWorkflowTrigger := ce.GetCertExtensionGithubWorkflowTrigger(); githubWorkflowTrigger != "" {
				ss.Optional[cosign.CertExtensionMap[cosign.CertExtensionGithubWorkflowTrigger]] = githubWorkflowTrigger
				ss.Optional[cosign.CertExtensionGithubWorkflowTrigger] = githubWorkflowTrigger
			}

			if githubWorkflowSha := ce.GetExtensionGithubWorkflowSha(); githubWorkflowSha != "" {
				ss.Optional[cosign.CertExtensionMap[cosign.CertExtensionGithubWorkflowSha]] = githubWorkflowSha
				ss.Optional[cosign.CertExtensionGithubWorkflowSha] = githubWorkflowSha
			}
			if githubWorkflowName := ce.GetCertExtensionGithubWorkflowName(); githubWorkflowName != "" {
				ss.Optional[cosign.CertExtensionMap[cosign.CertExtensionGithubWorkflowName]] = githubWorkflowName
				ss.Optional[cosign.CertExtensionGithubWorkflowName] = githubWorkflowName
			}

			if githubWorkflowRepository := ce.GetCertExtensionGithubWorkflowRepository(); githubWorkflowRepository != "" {
				ss.Optional[cosign.CertExtensionMap[cosign.CertExtensionGithubWorkflowRepository]] = githubWorkflowRepository
				ss.Optional[cosign.CertExtensionGithubWorkflowRepository] = githubWorkflowRepository
			}

			if githubWorkflowRef := ce.GetCertExtensionGithubWorkflowRef(); githubWorkflowRef != "" {
				ss.Optional[cosign.CertExtensionMap[cosign.CertExtensionGithubWorkflowRef]] = githubWorkflowRef
				ss.Optional[cosign.CertExtensionGithubWorkflowRef] = githubWorkflowRef
			}
		}
		if container_bundle, err := sig.Bundle(); err == nil && container_bundle != nil {
			if ss.Optional == nil {
				ss.Optional = make(map[string]interface{})
			}
			ss.Optional["Bundle"] = container_bundle
		}
		if rfc3161Timestamp, err := sig.RFC3161Timestamp(); err == nil && rfc3161Timestamp != nil {
			if ss.Optional == nil {
				ss.Optional = make(map[string]interface{})
			}
			ss.Optional["RFC3161Timestamp"] = rfc3161Timestamp
		}

		outputKeys = append(outputKeys, ss)
	}
	return outputKeys, nil
}

// VerifyFromIdentity verifies the image from the identity and extracts the keys
func VerifyFromIdentity(ctx context.Context, imageRef string, owner string, token string,
	identity string, issuer string) (bool, bool, map[string]interface{}, error) {
	imageKeys := make(map[string]interface{})

	options := []name.Option{}
	ref, err := name.ParseReference(imageRef, options...)
	if err != nil {
		return false, false, nil, fmt.Errorf("error parsing reference url: %w", err)
	}
	identityObj := []cosign.Identity{{Issuer: issuer, Subject: identity}}

	// get fulcio roots
	rootCerts, err := fulcio.GetRoots()
	if err != nil {
		return false, false, nil, fmt.Errorf("error getting fulcio roots: %w", err)
	}

	pubkeys, err := cosign.GetRekorPubs(ctx)
	if err != nil {
		return false, false, nil, fmt.Errorf("error getting rekor public keys: %w", err)
	}

	// need to authenticate in case artifact is private
	auth := githubAuthenticator{owner, token}
	registryClientOpts := []ociremote.Option{ociremote.WithRemoteOptions(remote.WithAuth(auth))}

	co := &cosign.CheckOpts{
		RegistryClientOpts: registryClientOpts,
		Identities:         identityObj,
		RootCerts:          rootCerts,
		RekorPubKeys:       pubkeys,
		IgnoreSCT:          true,
		ClaimVerifier:      cosign.SimpleClaimVerifier,
	}
	verified, bundleVerified, err := cosign.VerifyImageSignatures(ctx, ref, co)
	if err != nil {
		return false, false, nil, fmt.Errorf("error verifying image: %w", err)
	}
	is_verified := (len(verified) > 0)
	if is_verified {
		outputKeys, err := GetKeysFromVerified(verified)
		if err != nil {
			return false, bundleVerified, nil, fmt.Errorf("error getting keys from verified: %w", err)
		}

		if len(outputKeys) > 0 {
			imageKey := outputKeys[0]
			imageKeys["Issuer"] = imageKey.Optional["Issuer"]
			imageKeys["Identity"] = imageKey.Optional["Subject"]
			imageKeys["WorkflowName"] = imageKey.Optional["githubWorkflowName"]
			imageKeys["WorkflowRepository"] = imageKey.Optional["githubWorkflowRepository"]
			imageKeys["WorkflowSha"] = imageKey.Optional["githubWorkflowSha"]
			imageKeys["WorkflowTrigger"] = imageKey.Optional["githubWorkflowTrigger"]
			container_payload := imageKey.Optional["Bundle"].(*bundle.RekorBundle).Payload
			imageKeys["SignatureTime"] = container_payload.IntegratedTime
			imageKeys["RekorLogIndex"] = container_payload.LogIndex
			imageKeys["RekorLogID"] = container_payload.LogID
		}
	}

	return is_verified, bundleVerified, imageKeys, err
}

// TagsContainSignature if tag contains the .sig suffix it's a signature, as cosign
// stores signatures in that format
func TagsContainSignature(tags []string) bool {
	// if the artifact has a .sig tag it's a signature, skip it
	found := false
	for _, tag := range tags {
		if strings.HasSuffix(tag, ".sig") {
			found = true
			break
		}
	}
	return found
}

// FindSignatureTag returns the signature tag for a given image if exists
func FindSignatureTag(tags []string) string {
	// if the artifact has a .sig tag it's a signature, skip it
	for _, tag := range tags {
		if strings.HasSuffix(tag, ".sig") {
			return tag
		}
	}
	return ""
}
