// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cli provides internal CLI helpers shared by command implementations.
package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"
	billyutil "github.com/go-git/go-billy/v5/util"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
	"github.com/mindersec/minder/pkg/profiles"
)

const (
	catalogRuleTypesDir = "rule-types"
	catalogProfilesDir  = "profiles"
)

// Catalog represents a loaded collection of rule types and profiles from a filesystem.
type Catalog struct {
	RuleTypes []*minderv1.RuleType
	Profiles  []*minderv1.Profile
}

// LoadCatalogFromFS loads and validates all resources under the catalog directories.
// Invalid resources are skipped and reported through warnf.
func LoadCatalogFromFS(vfs billy.Filesystem, warnf func(string, ...any)) (*Catalog, error) {
	if warnf == nil {
		warnf = func(string, ...any) {}
	}

	ruleTypesByName, err := loadRuleTypesFromFS(vfs, warnf)
	if err != nil {
		return nil, err
	}

	loadedProfiles, err := loadProfilesFromFS(vfs, warnf)
	if err != nil {
		return nil, err
	}

	validProfiles := validateCatalog(ruleTypesByName, loadedProfiles, warnf)
	if len(validProfiles) == 0 {
		return nil, fmt.Errorf("no valid profiles found under %s", catalogProfilesDir)
	}

	ruleTypeNames := make([]string, 0, len(ruleTypesByName))
	for name := range ruleTypesByName {
		ruleTypeNames = append(ruleTypeNames, name)
	}
	sort.Strings(ruleTypeNames)

	ruleTypes := make([]*minderv1.RuleType, 0, len(ruleTypeNames))
	for _, name := range ruleTypeNames {
		ruleTypes = append(ruleTypes, ruleTypesByName[name])
	}

	return &Catalog{RuleTypes: ruleTypes, Profiles: validProfiles}, nil
}

func loadRuleTypesFromFS(vfs billy.Filesystem, warnf func(string, ...any)) (map[string]*minderv1.RuleType, error) {
	paths, err := collectYAMLFiles(vfs, catalogRuleTypesDir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no rule type YAML files found under %s", catalogRuleTypesDir)
	}

	ruleTypesByName := make(map[string]*minderv1.RuleType, len(paths))
	for _, path := range paths {
		ruleType, err := readRuleTypeFromPath(vfs, path)
		if err != nil {
			warnf("Skipping invalid rule type %s: %v\n", path, err)
			continue
		}
		if ruleType.GetName() == "" {
			warnf("Skipping invalid rule type %s: missing name\n", path)
			continue
		}
		if _, exists := ruleTypesByName[ruleType.GetName()]; exists {
			warnf("Skipping duplicate rule type %q from %s\n", ruleType.GetName(), path)
			continue
		}
		ruleTypesByName[ruleType.GetName()] = ruleType
	}

	if len(ruleTypesByName) == 0 {
		return nil, fmt.Errorf("no valid rule types found under %s", catalogRuleTypesDir)
	}

	return ruleTypesByName, nil
}

func loadProfilesFromFS(vfs billy.Filesystem, warnf func(string, ...any)) ([]*minderv1.Profile, error) {
	paths, err := collectYAMLFiles(vfs, catalogProfilesDir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no profile YAML files found under %s", catalogProfilesDir)
	}

	loadedProfiles := make([]*minderv1.Profile, 0, len(paths))
	profileNames := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		profile, err := profiles.ReadProfileFromPath(vfs, path)
		if err != nil {
			warnf("Skipping invalid profile %s: %v\n", path, err)
			continue
		}
		if profile.GetName() == "" {
			warnf("Skipping invalid profile %s: missing name\n", path)
			continue
		}
		if _, exists := profileNames[profile.GetName()]; exists {
			warnf("Skipping duplicate profile %q from %s\n", profile.GetName(), path)
			continue
		}

		profileNames[profile.GetName()] = struct{}{}
		loadedProfiles = append(loadedProfiles, profile)
	}

	if len(loadedProfiles) == 0 {
		return nil, fmt.Errorf("no valid profiles found under %s", catalogProfilesDir)
	}

	return loadedProfiles, nil
}

func validateCatalog(
	ruleTypesByName map[string]*minderv1.RuleType,
	loadedProfiles []*minderv1.Profile,
	warnf func(string, ...any),
) []*minderv1.Profile {
	validProfiles := make([]*minderv1.Profile, 0, len(loadedProfiles))

	for _, loadedProfile := range loadedProfiles {
		referencedRuleTypes := make(map[string]struct{})
		if err := profiles.TraverseRuleTypesForEntities(loadedProfile, func(_ minderv1.Entity, rule *minderv1.Profile_Rule) error {
			if rule.GetType() != "" {
				referencedRuleTypes[rule.GetType()] = struct{}{}
			}
			return nil
		}); err != nil {
			warnf("Skipping invalid profile %q: failed to inspect rules: %v\n", loadedProfile.GetName(), err)
			continue
		}

		missingRuleTypes := make([]string, 0)
		for ruleType := range referencedRuleTypes {
			if _, exists := ruleTypesByName[ruleType]; !exists {
				missingRuleTypes = append(missingRuleTypes, ruleType)
			}
		}

		if len(missingRuleTypes) > 0 {
			sort.Strings(missingRuleTypes)
			warnf(
				"Skipping profile %q: references missing rule type(s): %s\n",
				loadedProfile.GetName(),
				strings.Join(missingRuleTypes, ", "),
			)
			continue
		}

		validProfiles = append(validProfiles, loadedProfile)
	}

	return validProfiles
}

func collectYAMLFiles(vfs billy.Filesystem, root string) ([]string, error) {
	paths := make([]string, 0)
	err := billyutil.Walk(vfs, root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("catalog directory %s not found", root)
		}
		return nil, fmt.Errorf("failed to read catalog directory %s: %w", root, err)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	sort.Strings(paths)
	return paths, nil
}

func readRuleTypeFromPath(vfs billy.Filesystem, path string) (*minderv1.RuleType, error) {
	reader, err := vfs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rule type file %s: %w", path, err)
	}
	defer reader.Close()

	ext := filepath.Ext(path)
	if ext == "" {
		ext = ".yaml"
	}
	tmpFile, err := os.CreateTemp("", "minder-quickstart-ruletype-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for %s: %w", path, err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to copy rule type file %s: %w", path, err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file for %s: %w", path, err)
	}

	decoder, closer := fileconvert.DecoderForFile(tmpName)
	if decoder == nil {
		return nil, fmt.Errorf("unsupported rule type format for %s", path)
	}
	defer closer.Close()

	ruleType, err := fileconvert.ReadResourceTyped[*minderv1.RuleType](decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rule type file %s: %w", path, err)
	}

	return ruleType, nil
}
