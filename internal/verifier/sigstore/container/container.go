// Copyright 2023 Stacklok, Inc
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

// Package container provides the tools to verify a container artifact using sigstore
package container

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	protorekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Verify verifies a container artifact using sigstore
func Verify(_ context.Context, sev *verify.SignedEntityVerifier, accessToken, registry, owner, artifact, version string) (
	[]byte, []byte, error) {
	// create a default verification result
	params := newVerifyResult(BuildImageRef(registry, owner, artifact, version))

	// construct the bundle
	err := bundleFromOCIImage(params, newGithubAuthenticator(owner, accessToken))
	if err != nil {
		return nil, nil, err
	}

	// verify the artifact
	verificationResult, err := sev.Verify(params.bundle, verify.NewPolicy(verify.WithArtifactDigest("sha512", params.digest),
		verify.WithCertificateIdentity(*params.certID)))
	if err != nil {
		return nil, nil, err
	}

	// at the point, we have verified the artifact
	params.si.IsVerified = true
	params.si.IsBundleVerified = true

	// parse the verification result, populate signature and workflow info and return their bytes
	return parseVerificationResult(params, verificationResult)
}

func parseVerificationResult(params *verifyResult, res *verify.VerificationResult) ([]byte, []byte, error) {
	// this is not the workflow name, i.e. "Release", instead it is the workflow's URI
	// example: https://github.com/stacklok/minder/.github/workflows/chart-publish.yml@refs/heads/main
	params.wi.Name = res.Signature.Certificate.BuildSignerURI

	// repository name, example: https://github.com/stacklok/minder
	params.wi.Repository = res.Signature.Certificate.SourceRepositoryURI

	// source commit sha, example: 6dc6c6c27184f919ba6ca258e064bd7dd18e9200
	params.wi.CommitSha = res.Signature.Certificate.SourceRepositoryDigest

	// build trigger event, example: push
	params.wi.Trigger = res.Signature.Certificate.BuildTrigger

	// convert the signature and workflow info to bytes
	sig, err := util.GetBytesFromProto(params.si)
	if err != nil {
		return nil, nil, err
	}
	work, err := util.GetBytesFromProto(params.wi)
	if err != nil {
		return sig, nil, err
	}
	return sig, work, err
}

// bundleFromOCIImage returns a ProtobufBundle based on OCI image reference.
func bundleFromOCIImage(params *verifyResult, auth githubAuthenticator) error {
	// 1. Get the signature manifest from the OCI image reference
	manifest, err := getSignatureManifestFromOCIImage(params, auth)
	if err != nil {
		return fmt.Errorf("error getting signature manifest: %w", err)
	}

	// 2. Parse the manifest and return the simple signing layer
	simpleSigningLayer, err := parseSignatureManifest(params, manifest)
	if err != nil {
		return fmt.Errorf("error parsing the signature manifest: %w", err)
	}

	// 3. Build the verification material for the bundle
	verificationMaterial, err := getBundleVerificationMaterial(params, simpleSigningLayer)
	if err != nil {
		return fmt.Errorf("error getting verification material: %w", err)
	}
	// 4. Build the message signature for the bundle
	msgSignature, err := getBundleMsgSignature(simpleSigningLayer)
	if err != nil {
		return fmt.Errorf("error getting message signature: %w", err)
	}
	// 5. Construct and verify the bundle
	pbb := protobundle.Bundle{
		MediaType:            bundle.SigstoreBundleMediaType01,
		VerificationMaterial: verificationMaterial,
		Content:              msgSignature,
	}
	bun, err := bundle.NewProtobufBundle(&pbb)
	if err != nil {
		return fmt.Errorf("error creating bundle: %w", err)
	}
	// 6. Return the bundle and the digest of the simple signing layer (this is what is signed)

	// get the artifact digest
	params.digest, err = hex.DecodeString(simpleSigningLayer.Digest.Hex)
	if err != nil {
		return err
	}

	// get the certificate identity
	certID, err := verify.NewShortCertificateIdentity(*params.si.CertIssuer, "", "", *params.si.CertIdentity)
	if err != nil {
		return err
	}
	// store the bundle and the certificate identity
	params.bundle = bun
	params.certID = &certID
	return nil
}

// getSignatureManifestFromOCIImage returns the simple signing layer from the OCI image reference
func getSignatureManifestFromOCIImage(ret *verifyResult, auth githubAuthenticator) (*v1.Manifest, error) {
	// 0. Get the auth options
	opts := []remote.Option{remote.WithAuth(auth)}
	craneOpts := []crane.Option{crane.WithAuth(auth)}

	// 1. Get the image reference
	ref, err := name.ParseReference(ret.imageRef)
	if err != nil {
		return nil, fmt.Errorf("error parsing image reference: %w", err)
	}

	// 2. Get the image descriptor
	desc, err := remote.Get(ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("error getting image descriptor: %w", err)
	}

	// 3. Get the digest
	digest := ref.Context().Digest(desc.Digest.String())
	h, err := v1.NewHash(digest.Identifier())
	if err != nil {
		return nil, fmt.Errorf("error getting hash: %w", err)
	}

	// 4. Construct the signature reference - sha256-<hash>.sig
	sigTag := digest.Context().Tag(fmt.Sprint(h.Algorithm, "-", h.Hex, ".sig"))

	// 5. Get the manifest of the signature
	mf, err := crane.Manifest(sigTag.Name(), craneOpts...)
	if err != nil {
		return nil, fmt.Errorf("error getting signature manifest: %w", err)
	}
	sigManifest, err := v1.ParseManifest(bytes.NewReader(mf))
	if err != nil {
		return nil, fmt.Errorf("error parsing signature manifest: %w", err)
	}

	// 6. If successful, set the IsSigned flag to true
	ret.si.IsSigned = true

	// 7. Return the manifest
	return sigManifest, nil
}

// parseSignatureManifest returns the identity and issuer from the certificate
func parseSignatureManifest(ret *verifyResult, manifest *v1.Manifest) (*v1.Descriptor, error) {
	res := v1.Descriptor{}
	for _, layer := range manifest.Layers {
		if layer.MediaType == "application/vnd.dev.cosign.simplesigning.v1+json" {
			//signature_digest := layer.Digest.String()
			//signature := layer.Annotations["dev.cosignproject.cosign/signature"]
			cert := layer.Annotations["dev.sigstore.cosign/certificate"]
			// Decode the PEM-encoded certificate
			pemBlock, _ := pem.Decode([]byte(cert))
			if pemBlock == nil || pemBlock.Type != "CERTIFICATE" {
				return nil, fmt.Errorf("failed to decode PEM certificate")
			}

			// Parse the X.509 certificate
			certObj, err := x509.ParseCertificate(pemBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("error parsing certificate: %w", err)
			}
			for _, uri := range certObj.URIs {
				identity := uri.String()
				ret.si.CertIdentity = &identity
				res = layer
				break
			}

			// now parse the issuer
			customOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}

			// Search for the custom OID in the certificate extensions
			var customExtensionValue []byte
			for _, ext := range certObj.Extensions {
				if ext.Id.Equal(customOID) {
					customExtensionValue = ext.Value
					issuer := string(customExtensionValue)
					ret.si.CertIssuer = &issuer
					return &layer, nil
				}
			}
			break
		}
	}
	return &res, nil
}

// getBundleVerificationMaterial returns the bundle verification material from the simple signing layer
func getBundleVerificationMaterial(params *verifyResult, manifestLayer *v1.Descriptor) (
	*protobundle.VerificationMaterial, error) {
	// 1. Get the signing certificate chain
	signingCert, err := getVerificationMaterialX509CertificateChain(manifestLayer)
	if err != nil {
		return nil, fmt.Errorf("error getting signing certificate: %w", err)
	}
	// 2. Get the transparency log entries
	tlogEntries, err := getVerificationMaterialTlogEntries(params, manifestLayer)
	if err != nil {
		return nil, fmt.Errorf("error getting tlog entries: %w", err)
	}
	// 3. Construct the verification material
	return &protobundle.VerificationMaterial{
		Content:     signingCert,
		TlogEntries: tlogEntries,
		TimestampVerificationData: &protobundle.TimestampVerificationData{
			Rfc3161Timestamps: []*protocommon.RFC3161SignedTimestamp{
				{
					SignedTimestamp: tlogEntries[0].InclusionPromise.SignedEntryTimestamp,
				},
			},
		},
	}, nil
}

// getVerificationMaterialX509CertificateChain returns the verification material X509 certificate chain from the
// simple signing layer
func getVerificationMaterialX509CertificateChain(manifestLayer *v1.Descriptor) (
	*protobundle.VerificationMaterial_X509CertificateChain, error) {
	// 1. Get the PEM certificate from the simple signing layer
	pemCert := manifestLayer.Annotations["dev.sigstore.cosign/certificate"]
	// 2. Construct the DER encoded version of the PEM certificate
	block, _ := pem.Decode([]byte(pemCert))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	signingCert := protocommon.X509Certificate{
		RawBytes: block.Bytes,
	}
	// 3. Construct the X509 certificate chain
	return &protobundle.VerificationMaterial_X509CertificateChain{
		X509CertificateChain: &protocommon.X509CertificateChain{
			Certificates: []*protocommon.X509Certificate{&signingCert},
		},
	}, nil
}

// getVerificationMaterialTlogEntries returns the verification material transparency log entries from the simple signing layer
func getVerificationMaterialTlogEntries(params *verifyResult, manifestLayer *v1.Descriptor) (
	[]*protorekor.TransparencyLogEntry, error) {
	// 1. Get the bundle annotation
	bun := manifestLayer.Annotations["dev.sigstore.cosign/bundle"]
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(bun), &jsonData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %w", err)
	}
	// 2. Get the log index, log ID, integrated time, signed entry timestamp and body
	logIndex, ok := jsonData["Payload"].(map[string]interface{})["logIndex"].(float64)
	if !ok {
		return nil, fmt.Errorf("error getting logIndex")
	}
	li, ok := jsonData["Payload"].(map[string]interface{})["logID"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting logID")
	}
	logID, err := hex.DecodeString(li)
	if err != nil {
		return nil, fmt.Errorf("error decoding logID: %w", err)
	}
	integratedTime, ok := jsonData["Payload"].(map[string]interface{})["integratedTime"].(float64)
	if !ok {
		return nil, fmt.Errorf("error getting integratedTime")
	}
	set, ok := jsonData["SignedEntryTimestamp"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting SignedEntryTimestamp")
	}
	signedEntryTimestamp, err := base64.StdEncoding.DecodeString(set)
	if err != nil {
		return nil, fmt.Errorf("error decoding signedEntryTimestamp: %w", err)
	}
	// 3. Unmarshal the body and extract the rekor KindVersion details
	body, ok := jsonData["Payload"].(map[string]interface{})["body"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting body")
	}
	bodyBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("error decoding body: %w", err)
	}
	err = json.Unmarshal(bodyBytes, &jsonData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %w", err)
	}
	apiVersion := jsonData["apiVersion"].(string)
	kind := jsonData["kind"].(string)
	// 4. Store the log index and log ID from Rekor
	params.si.RekorLogId = &li
	params.si.RekorLogIndex = &logIndex
	// 5. Construct the transparency log entry list
	return []*protorekor.TransparencyLogEntry{
		{
			LogIndex: int64(logIndex),
			LogId: &protocommon.LogId{
				KeyId: logID,
			},
			KindVersion: &protorekor.KindVersion{
				Kind:    kind,
				Version: apiVersion,
			},
			IntegratedTime: int64(integratedTime),
			InclusionPromise: &protorekor.InclusionPromise{
				SignedEntryTimestamp: signedEntryTimestamp,
			},
			InclusionProof:    nil,
			CanonicalizedBody: bodyBytes,
		},
	}, nil
}

// getBundleMsgSignature returns the bundle message signature from the simple signing layer
func getBundleMsgSignature(simpleSigningLayer *v1.Descriptor) (*protobundle.Bundle_MessageSignature, error) {
	// 1. Get the message digest algorithm
	var msgHashAlg protocommon.HashAlgorithm
	switch simpleSigningLayer.Digest.Algorithm {
	case "sha256":
		msgHashAlg = protocommon.HashAlgorithm_SHA2_256
	default:
		return nil, fmt.Errorf("unknown digest algorithm: %s", simpleSigningLayer.Digest.Algorithm)
	}
	// 2. Get the message digest
	digest, err := hex.DecodeString(simpleSigningLayer.Digest.Hex)
	if err != nil {
		return nil, fmt.Errorf("error decoding digest: %w", err)
	}
	// 3. Get the signature
	s := simpleSigningLayer.Annotations["dev.cosignproject.cosign/signature"]
	sig, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("error decoding manSig: %w", err)
	}
	// Construct the bundle message signature
	return &protobundle.Bundle_MessageSignature{
		MessageSignature: &protocommon.MessageSignature{
			MessageDigest: &protocommon.HashOutput{
				Algorithm: msgHashAlg,
				Digest:    digest,
			},
			Signature: sig,
		},
	}, nil
}

// githubAuthenticator is an authenticator for GitHub
type githubAuthenticator struct{ username, password string }

// Authorization returns the username and password for the githubAuthenticator
func (g githubAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: g.username,
		Password: g.password,
	}, nil
}

// newGithubAuthenticator returns a new githubAuthenticator
func newGithubAuthenticator(username, password string) githubAuthenticator {
	return githubAuthenticator{username, password}
}

// BuildImageRef returns the OCI image reference
func BuildImageRef(registry, owner, artifact, version string) string {
	return fmt.Sprintf("%s/%s/%s@%s", registry, owner, artifact, version)
}

// verifyResult is the result of the verification
type verifyResult struct {
	// Params for the verification process
	imageRef string
	digest   []byte
	bundle   *bundle.ProtobufBundle
	certID   *verify.CertificateIdentity
	// Result of the verification
	si *pb.SignatureVerification
	wi *pb.GithubWorkflow
}

// newVerifyResult returns a new verifyResult
func newVerifyResult(imageRef string) *verifyResult {
	return &verifyResult{
		si: &pb.SignatureVerification{
			IsVerified:       false,
			IsSigned:         false,
			IsBundleVerified: false,
		},
		wi:       &pb.GithubWorkflow{},
		imageRef: imageRef,
	}
}
