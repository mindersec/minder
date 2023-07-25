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

// Package oci provides a client for interacting with the OCI API
package oci

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/go-containerregistry/pkg/authn"
	registry_name "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type githubAuthenticator struct{ username, password string }

func (g githubAuthenticator) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{
		Username: g.username,
		Password: g.password,
	}, nil
}

// REGISTRY is the default registry to use
const REGISTRY = "ghcr.io"

// GetImageManifest returns the manifest for the given image
func GetImageManifest(owner string, name string, tag string, username string, token string) (v1.Manifest, error) {
	imageRef := fmt.Sprintf("%s/%s/%s:%s", REGISTRY, owner, name, tag)
	ref, err := registry_name.ParseReference(imageRef)
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error parsing reference url: %w", err)
	}

	auth := githubAuthenticator{username, token}

	img, err := remote.Image(ref, remote.WithAuth(auth))
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error getting image: %w", err)
	}
	manifest, err := img.Manifest()
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error getting manifest: %w", err)
	}
	return *manifest, nil
}

func CheckSignatureVerification(owner string, name string, tag string, cert string, chain string) (bool, error) {
	// first save cert and chain to disk
	imageRef := fmt.Sprintf("%s/%s/%s:%s", REGISTRY, owner, name, tag)

	// extract public key from certificate
	pemBlock, _ := pem.Decode([]byte(cert))
	if pemBlock == nil || pemBlock.Type != "CERTIFICATE" {
		return false, fmt.Errorf("failed to decode PEM block containing public key")
	}

	// Parse the X.509 certificate
	cert_parsed, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate")
	}

	// Get the public key from the certificate
	publicKey := cert_parsed.PublicKey
	if publicKey == nil {
		return false, fmt.Errorf("error getting public key from certificate")
	}

	// Create a temporary .pub file
	tempFile, err := os.CreateTemp("", "public_key_*.pub")
	if err != nil {
		return false, fmt.Errorf("error creating temporary file")
	}
	defer tempFile.Close()

	// Convert the public key to a byte slice
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return false, fmt.Errorf("error marshaling public key")
	}

	pubKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	fmt.Println(string(pubKeyPem))
	// Write the public key data to the temporary .pub file
	_, err = tempFile.Write(pubKeyPem)
	if err != nil {
		return false, fmt.Errorf("error writing to temporary file")
	}

	// call cosign to verify
	command := "cosign"
	args := []string{
		"verify",
		"--key",
		tempFile.Name(),
		imageRef,
	}
	fmt.Println(args)

	// Run the command
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd_err := cmd.Run()
	if cmd_err != nil {
		return false, cmd_err
	}
	fmt.Println(cmd.Stdout)
	return true, nil
}
