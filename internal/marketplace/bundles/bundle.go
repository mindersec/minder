// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bundles contains a domain-level representation of the bundles
// created by the mindpak tool, and logic for working with that model
package bundles

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/stacklok/minder/internal/engine"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stacklok/minder/pkg/mindpak"
)

// Bundle encapsulates operations on the mindpak bundles
type Bundle interface {
	// GetMetadata returns the bundle information as-is
	GetMetadata() *mindpak.Metadata
	// GetProfile takes the name of a profile in the bundle and attempts to
	// read it from the bundle, parse it and return an instance of the profile
	// struct
	GetProfile(string) (*v1.Profile, error)
	// ForEachRuleType walks each rule type in the bundle, attempts to read
	// and parse the rule type, and then applies the specific anonymous
	// function to the rule type
	ForEachRuleType(func(*v1.RuleType) error) error
}

type profileSet = map[string]struct{}
type bundle struct {
	original *mindpak.Bundle
	profiles profileSet
}

// NewBundle creates an instance of the Bundle interface from a mindpak
// bundle
func NewBundle(mpBundle *mindpak.Bundle) Bundle {
	bundleProfiles := mpBundle.Files.Profiles
	profiles := make(profileSet, len(bundleProfiles))
	// build a set of profile names for `GetProfile` to ease lookups
	for _, profile := range bundleProfiles {
		profiles[profile.Name] = struct{}{}
	}

	return &bundle{
		original: mpBundle,
		profiles: profiles,
	}
}

func (b *bundle) GetMetadata() *mindpak.Metadata {
	return b.original.Manifest.Metadata
}

func (b *bundle) GetProfile(name string) (*v1.Profile, error) {
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
	profile, err := engine.ParseYAML(file)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile yaml: %w", err)
	}

	return profile, nil
}

func (b *bundle) ForEachRuleType(fn func(*v1.RuleType) error) error {
	var err error
	var file fs.File
	// used for error handling if we return during the loop
	defer func() {
		_ = file.Close()
	}()

	for _, ruleType := range b.original.Files.RuleTypes {
		// read from bundle
		path := fmt.Sprintf("%s/%s", mindpak.PathRuleTypes, ruleType.Name)
		file, err = b.original.Source.Open(path)
		if err != nil {
			return fmt.Errorf("error reading rule type from bundle: %w", err)
		}

		// parse rule type from YAML
		parsedRuleType, err := v1.ParseRuleType(file)
		if err != nil {
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

func ensureYamlSuffix(name string) string {
	if strings.HasSuffix(name, fileSuffix) {
		return name
	}
	return fmt.Sprintf("%s%s", name, fileSuffix)
}

const (
	fileSuffix = ".yaml"
)
