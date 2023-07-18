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
	"fmt"

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
func GetImageManifest(owner string, name string, tags []string, username string, token string) (v1.Manifest, error) {
	imageRef := fmt.Sprintf("%s/%s/%s:%s", REGISTRY, owner, name, tags[0])
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

// GetImageConfig returns the config for the given image
func GetImageConfig(image registry_name.Reference) (v1.ConfigFile, error) {
	img, err := remote.Image(image)
	if err != nil {
		return v1.ConfigFile{}, fmt.Errorf("error getting image: %w", err)
	}

	config, err := img.ConfigFile()
	if err != nil {
		return v1.ConfigFile{}, fmt.Errorf("error getting config: %w", err)
	}

	return *config, nil
}

// GetImageLayers returns the layers for the given image
func GetImageLayers(image registry_name.Reference) ([]v1.Layer, error) {
	img, err := remote.Image(image)
	if err != nil {
		return nil, fmt.Errorf("error getting image: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("error getting layers: %w", err)
	}

	return layers, nil
}

// GetImageLayerByDigest returns the layer for the given image and digest
func GetImageLayerByDigest(image registry_name.Reference, digest string) (v1.Layer, error) {
	img, err := remote.Image(image)
	if err != nil {
		return nil, fmt.Errorf("error getting image: %w", err)
	}

	layer, err := img.LayerByDigest(v1.Hash{
		Algorithm: "sha256",
		Hex:       digest,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting layer: %w", err)
	}

	return layer, nil
}
