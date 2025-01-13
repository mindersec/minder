// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reader contains logic for accessing the contents of a bundle
package reader

import (
	"fmt"
	"io/fs"
	"strings"

	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/profiles"
)

// BundleReader provides a high-level interface for accessing the contents of
// a Bundle
type BundleReader interface {
	// GetMetadata returns the bundle information as-is
	GetMetadata() *mindpak.Metadata
	// GetProfile takes the name of a profile in the bundle and attempts to
	// read it from the bundle, parse it and return an instance of the profile
	// struct
	GetProfile(string) (*v1.Profile, error)
	// ForEachRuleType walks each rule type in the bundle, attempts to read
	// and parse the rule type, and then applies the specified anonymous
	// function to the rule type
	ForEachRuleType(func(*v1.RuleType) error) error
	// ForEachDataSource walks each data source in the bundle, attempts to read
	// and parse the data source, and then applies the specified anonymous
	// function to the rule type
	ForEachDataSource(func(source *v1.DataSource) error) error
}

type profileSetType = map[string]struct{}
type bundleReader struct {
	original *mindpak.Bundle
	profiles profileSetType
}

// NewBundleReader creates an instance of BundleReader from mindpak.Bundle
func NewBundleReader(bundle *mindpak.Bundle) BundleReader {
	bundleProfiles := bundle.Files.Profiles
	profileSet := make(profileSetType, len(bundleProfiles))
	// build a set of profile names for `GetProfile`
	// this saves us from searching the manifest each time this method is used
	for _, profile := range bundleProfiles {
		profileSet[profile.Name] = struct{}{}
	}

	return &bundleReader{
		original: bundle,
		profiles: profileSet,
	}
}

func (b *bundleReader) GetMetadata() *mindpak.Metadata {
	return b.original.Manifest.Metadata
}

func (b *bundleReader) GetProfile(name string) (*v1.Profile, error) {
	// if called with a namespace prefix, remove it
	name, err := b.stripNamespace(name)
	if err != nil {
		return nil, err
	}
	// ensure name has file extension
	name = ensureYamlSuffix(name)
	// validate that profile exists
	_, ok := b.profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile does not exist in bundle: %s", name)
	}

	// read from bundle
	path := fmt.Sprintf("%s/%s", mindpak.PathProfiles, name)
	file, err := b.original.Source.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error reading profile from bundle: %w", err)
	}
	defer file.Close()

	// parse profile from YAML
	profile, err := profiles.ParseYAML(file)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile yaml: %w", err)
	}

	return profile, nil
}

func (b *bundleReader) ForEachRuleType(fn func(*v1.RuleType) error) error {
	var err error
	var file fs.File
	// used for error handling if we return during the loop
	defer func() {
		// Add precaution to close file only if it was assigned
		if file != nil {
			_ = file.Close()
		}
	}()

	for _, ruleType := range b.original.Files.RuleTypes {
		// read from bundle
		path := fmt.Sprintf("%s/%s", mindpak.PathRuleTypes, ruleType.Name)
		file, err = b.original.Source.Open(path)
		if err != nil {
			return fmt.Errorf("error reading rule type from bundle: %w", err)
		}

		// parse rule type from YAML
		parsedRuleType := &v1.RuleType{}
		if err := v1.ParseResource(file, parsedRuleType); err != nil {
			return fmt.Errorf("error parsing rule type yaml: %w", err)
		}
		if err = file.Close(); err != nil {
			return fmt.Errorf("error closing file: %w", err)
		}

		// apply operation from caller
		err = fn(parsedRuleType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *bundleReader) ForEachDataSource(fn func(source *v1.DataSource) error) error {
	var err error
	var file fs.File
	// used for error handling if we return during the loop
	defer func() {
		// Add precaution to close file only if it was assigned
		if file != nil {
			_ = file.Close()
		}
	}()

	for _, dataSource := range b.original.Files.DataSources {
		// read from bundle
		path := fmt.Sprintf("%s/%s", mindpak.PathDataSources, dataSource.Name)
		file, err = b.original.Source.Open(path)
		if err != nil {
			return fmt.Errorf("error reading data source from bundle: %w", err)
		}

		// parse data source from YAML
		parsedDataSource := &v1.DataSource{}
		if err := v1.ParseResourceProto(file, parsedDataSource); err != nil {
			return fmt.Errorf("error parsing data source yaml: %w", err)
		}
		if err = file.Close(); err != nil {
			return fmt.Errorf("error closing file: %w", err)
		}

		// apply operation from caller
		err = fn(parsedDataSource)
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureYamlSuffix(name string) string {
	if strings.HasSuffix(name, fileSuffix) {
		return name
	}
	return fmt.Sprintf("%s%s", name, fileSuffix)
}

func (b *bundleReader) stripNamespace(name string) (string, error) {
	components := strings.Split(name, "/")
	switch len(components) {
	// non namespaced name
	case 1:
		return name, nil
	case 2:
		// sanity check that the namespace relates to this bundle
		if components[0] != b.original.Manifest.Metadata.Namespace {
			return "", fmt.Errorf("invalid namespace: %s", components[0])
		}
		return components[1], nil
	default:
		return "", fmt.Errorf("malformed profile name: %s", name)
	}
}

const (
	fileSuffix = ".yaml"
)
