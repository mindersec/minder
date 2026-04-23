// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cli provides internal CLI helpers shared by command implementations.
package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-git/go-billy/v5"

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

// WarnFunc reports skipped catalog resources.
type WarnFunc = fileconvert.Printer

// LoadCatalogFromFS loads and validates all resources under the catalog directories.
// Invalid resources are skipped and reported through warnf.
func LoadCatalogFromFS(vfs billy.Filesystem, warnf WarnFunc) (*Catalog, error) {
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

	ruleTypeNames := make([]string, 0, len(ruleTypesByName))
	for name := range ruleTypesByName {
		ruleTypeNames = append(ruleTypeNames, name)
	}
	sort.Strings(ruleTypeNames)

	ruleTypes := make([]*minderv1.RuleType, 0, len(ruleTypeNames))
	for _, name := range ruleTypeNames {
		ruleTypes = append(ruleTypes, ruleTypesByName[name])
	}

	catalog := &Catalog{RuleTypes: ruleTypes, Profiles: loadedProfiles}
	if err := catalog.Validate(warnf); err != nil {
		return nil, err
	}

	return catalog, nil
}

// Validate validates the catalog contents and keeps only profiles whose rule types exist.
func (c *Catalog) Validate(warnf WarnFunc) error {
	if c == nil {
		return fmt.Errorf("catalog is nil")
	}
	if warnf == nil {
		warnf = func(string, ...any) {}
	}

	ruleTypesByName := make(map[string]struct{}, len(c.RuleTypes))
	for _, ruleType := range c.RuleTypes {
		if name := ruleType.GetName(); name != "" {
			ruleTypesByName[name] = struct{}{}
		}
	}

	validProfiles := make([]*minderv1.Profile, 0, len(c.Profiles))
	for _, loadedProfile := range c.Profiles {
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

	if len(validProfiles) == 0 {
		return fmt.Errorf("no valid profiles found under %s", catalogProfilesDir)
	}

	c.Profiles = validProfiles
	return nil
}

func loadRuleTypesFromFS(vfs billy.Filesystem, warnf WarnFunc) (map[string]*minderv1.RuleType, error) {
	ruleTypes, err := fileconvert.ResourcesFromFilesystem[*minderv1.RuleType](warnf, vfs, catalogRuleTypesDir)
	if err != nil {
		return nil, err
	}

	ruleTypesByName := make(map[string]*minderv1.RuleType, len(ruleTypes))
	for _, ruleType := range ruleTypes {
		if ruleType.GetName() == "" {
			warnf("Skipping invalid rule type: missing name\n")
			continue
		}
		if _, exists := ruleTypesByName[ruleType.GetName()]; exists {
			warnf("Skipping duplicate rule type %q\n", ruleType.GetName())
			continue
		}
		ruleTypesByName[ruleType.GetName()] = ruleType
	}

	if len(ruleTypesByName) == 0 {
		return nil, fmt.Errorf("no valid rule types found under %s", catalogRuleTypesDir)
	}

	return ruleTypesByName, nil
}

func loadProfilesFromFS(vfs billy.Filesystem, warnf WarnFunc) ([]*minderv1.Profile, error) {
	loadedProfilesFromFS, err := fileconvert.ResourcesFromFilesystem[*minderv1.Profile](warnf, vfs, catalogProfilesDir)
	if err != nil {
		return nil, err
	}

	loadedProfiles := make([]*minderv1.Profile, 0, len(loadedProfilesFromFS))
	profileNames := make(map[string]struct{}, len(loadedProfilesFromFS))

	for _, profile := range loadedProfilesFromFS {
		if profile.GetName() == "" {
			warnf("Skipping invalid profile: missing name\n")
			continue
		}
		if _, exists := profileNames[profile.GetName()]; exists {
			warnf("Skipping duplicate profile %q\n", profile.GetName())
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
