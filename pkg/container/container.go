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
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	containerregistry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/fulcio"
	cosign "github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	oci "github.com/sigstore/cosign/v2/pkg/oci"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/payload"
)

type githubAuthenticator struct{ username, password string }

func (g githubAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: g.username,
		Password: g.password,
	}, nil
}

// GetSignatureTag returns the signature tag for a given image if exists
func GetSignatureTag(imageRef name.Reference) (name.Reference, error) {
	ociremoteOpts := []ociremote.Option{}
	dstRef, err := ociremote.SignatureTag(imageRef, ociremoteOpts...)
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
