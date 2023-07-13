package oci

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// GetImageManifest returns the manifest for the given image
func GetImageManifest(imageRef string, token string) (v1.Manifest, error) {

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error parsing reference url: %w", err)
	}

	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error parsing image url: %w", err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error getting image: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return v1.Manifest{}, fmt.Errorf("error getting manifest: %w", err)
	}

	layers := manifest.Layers
	for _, layer := range layers {
		digest := layer.Digest.String()
		fmt.Println("Layer Digest:", digest)
	}

	return *manifest, nil
}

// GetImageManifest returns the manifest for the given image
func GetImageDigest(imageRef string, token string) (v1.Hash, error) {

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return v1.Hash{}, fmt.Errorf("error parsing reference url: %w", err)
	}

	if err != nil {
		return v1.Hash{}, fmt.Errorf("error parsing image url: %w", err)
	}

	// auth := &authn.Bearer{Token: token}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return v1.Hash{}, fmt.Errorf("error getting image: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		fmt.Printf("Error retrieving image digest: %v", err)
	}
	return digest, nil
}

// GetImageConfig returns the config for the given image
func GetImageConfig(image name.Reference) (v1.ConfigFile, error) {
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
func GetImageLayers(image name.Reference) ([]v1.Layer, error) {
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
func GetImageLayerByDigest(image name.Reference, digest string) (v1.Layer, error) {
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
