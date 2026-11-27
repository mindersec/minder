// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package diff provides the diff rule data ingest engine
package diff

import (
	"bufio"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/internal/util"
)

var (
	versionRegex        = regexp.MustCompile(`^\+\s*"version"\s*:\s*"([^"\n]*)"\s*(?:,|$)`)
	dependencyNameRegex = regexp.MustCompile(`\s*"([^"]+)"\s*:\s*{\s*`)
	// packageNameRegex matches the "name" field of a package-lock.json entry,
	// which npm writes for an aliased dependency ("my-lodash": "npm:lodash@4").
	packageNameRegex = regexp.MustCompile(`^[+ ]?\s*"name"\s*:\s*"([^"]+)"`)
)

type ecosystemParser func(string) ([]*pbinternal.Dependency, error)

func newEcosystemParser(eco DependencyEcosystem) ecosystemParser {
	switch strings.ToLower(string(eco)) {
	case string(DepEcosystemNPM):
		return npmParse
	case string(DepEcosystemGo):
		return goParse
	case string(DepEcosystemPyPI):
		// currently we only support requirements.txt
		// (the name comes from the rule config, so e.g. requirements-dev.txt would be supported, too)
		return requirementsParse
	case string(DepEcosystemNone):
		return nil
	default:
		return nil
	}
}

func requirementsParse(patch string) ([]*pbinternal.Dependency, error) {
	var deps []*pbinternal.Dependency

	scanner := bufio.NewScanner(strings.NewReader(patch))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		line = pyReqNormalizeLine(line)
		if line == "" {
			continue
		}

		// Lines starting with - are pip options, such as -r other.txt, -e <url> or
		// --hash=..., not requirements.
		if strings.HasPrefix(line, "-") {
			continue
		}

		matches := util.PyRequestsVersionRegexp.FindAllStringSubmatch(line, -1)
		// For now, if no version is set, we just don't set the version. The evaluator
		// will skip the dependency, but later we can query the package repository for the
		// latest version
		if matches == nil {
			deps = pyReqAddPkgName(deps, pyReqStripExtras(line), "")
			continue
		}

		version := pyReqLowestVersion(matches)

		// Extract the name by grabbing everything up to the first operator,
		// without any extras such as [security]
		if name, ok := util.PyRequestsName(line); ok {
			deps = pyReqAddPkgName(deps, name, version)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return deps, nil
}

func pyReqNormalizeLine(line string) string {
	if !strings.HasPrefix(line, "+") {
		return ""
	}
	line = strings.TrimPrefix(line, "+")

	// Remove inline comments
	if idx := strings.Index(line, "#"); idx != -1 {
		line = line[:idx]
	}

	return strings.TrimSpace(line)
}

// pyReqLowestVersion returns the lowest version a requirement's specifiers
// allow. This is probably a bit confusing. If there is a lower bound (>, >=,
// ~=) or an exact version (==, ===), we use that, because then the lower
// version is set explicitly. Otherwise we just use the first version we find.
func pyReqLowestVersion(matches [][]string) string {
	version := ""
	var lowestVersion string
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		if version == "" {
			version = match[2]
		}
		switch match[1] {
		// ~= is a compatible release: ~=1.26.0 means >=1.26.0, ==1.26.*
		case ">", ">=", "~=", "==", "===":
			lowestVersion = match[2]
		}
	}
	if lowestVersion != "" {
		return lowestVersion
	}
	return version
}

// pyReqStripExtras removes the extras from a requirement without a version
// specifier: "requests[security]" is the requests package.
func pyReqStripExtras(line string) string {
	if i := strings.Index(line, "["); i != -1 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}

func pyReqAddPkgName(depList []*pbinternal.Dependency, pkgName, version string) []*pbinternal.Dependency {
	dep := &pbinternal.Dependency{
		Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
		Name:      pkgName,
		Version:   version,
	}
	return append(depList, dep)
}

func goParse(patch string) ([]*pbinternal.Dependency, error) {
	var deps []*pbinternal.Dependency
	scanner := bufio.NewScanner(strings.NewReader(patch))

	// Iterate over the lines of the go.mod patch and parse the dependencies
	for scanner.Scan() {
		// Parse the line and extract a dependency
		dep := extractGoDepFromPatchLine(scanner.Text())

		// If we failed to extract a dependency, or if it's already in the slice, skip it
		if dep == nil || slices.ContainsFunc(deps, func(n *pbinternal.Dependency) bool {
			if n.Name == dep.Name && n.Version == dep.Version {
				return true
			}
			return false
		}) {
			continue
		}

		// Add the dependency to the slice
		deps = append(deps, dep)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return deps, nil
}

func extractGoDepFromPatchLine(line string) *pbinternal.Dependency {
	// Look for lines that add dependencies.
	// We ignore lines that contain "// indirect" because they are transitive dependencies, and therefore
	// not actionable.
	if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") && !strings.Contains(line, "// indirect") {
		// Extract the part after the '+' sign.
		lineContent := line[1:]

		fields := strings.Fields(lineContent)
		if len(fields) < 2 {
			// No match
			return nil
		}

		dep := &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO,
		}
		if fields[0] == "require" && fields[1] != "(" && len(fields) >= 3 {
			dep.Name = fields[1]
			dep.Version = fields[2]
		} else if strings.HasPrefix(lineContent, "\t") {
			dep.Name = fields[0]
			dep.Version = fields[1]
		} else if fields[0] == "replace" && strings.Contains(lineContent, "=>") && len(fields) >= 5 {
			// For lines with version replacements, the new version is after the "=>"
			// Assuming format is module path version => newModulePath newVersion
			dep.Name = fields[3]
			dep.Version = fields[4]
		} else {
			// No match
			return nil
		}
		// Return the dependency
		return dep
	}
	// No match
	return nil
}

func npmParse(patch string) ([]*pbinternal.Dependency, error) {
	lines := strings.Split(patch, "\n")

	var deps []*pbinternal.Dependency

	for i, line := range lines {
		// Check if the line contains a version
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			name := findDependencyName(i, lines)
			// The version is not always a dependency version. It may also be the version of the package in this repo,
			// or the version of the root project. See https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
			if name != "" {
				deps = append(deps, &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      name,
					Version:   version,
				})
			}
		}
	}

	return deps, nil
}

// findDependencyName iterates over all the previous lines to find the JSON key containing the parent dependency name.
// If the parent key does not correspond to a dependency (i.e. it could be the root project or a workspace package),
// then an empty string is returned.
func findDependencyName(i int, lines []string) string {
	closingBraces := 0
	// aliasName is the entry's own "name" field, which npm writes for an
	// aliased dependency: the entry is installed under the alias, but the
	// package is the one named there.
	aliasName := ""
	for j := i - 1; j >= 0; j-- {
		if strings.Contains(lines[j], "}") {
			closingBraces = closingBraces + 1
		}
		if closingBraces == 0 && aliasName == "" {
			if m := packageNameRegex.FindStringSubmatch(lines[j]); m != nil {
				aliasName = m[1]
			}
		}
		if strings.Contains(lines[j], "{") {
			if closingBraces == 0 {
				matches := dependencyNameRegex.FindStringSubmatch(lines[j])
				if len(matches) > 1 {
					// extract the dependency name from the key, which is the dependency path
					dependencyPath := matches[1]
					if isLocalPackagePath(dependencyPath) {
						return ""
					}
					if aliasName != "" {
						return aliasName
					}
					return getDependencyName(dependencyPath)
				}
				return ""
			}
			closingBraces = closingBraces - 1
		}
	}
	return ""
}

// isLocalPackagePath reports whether a package-lock.json key is a package in
// the repository itself, such as a workspace ("packages/web"), rather than an
// installed dependency. Installed dependencies are under node_modules/ in
// lockfile v2 and v3, and are keyed by their name ("lodash", "@babel/core")
// in v1.
func isLocalPackagePath(dependencyPath string) bool {
	if strings.HasPrefix(dependencyPath, "node_modules/") || strings.Contains(dependencyPath, "/node_modules/") {
		return false
	}
	if strings.HasPrefix(dependencyPath, "@") {
		return strings.Count(dependencyPath, "/") != 1
	}
	return strings.Contains(dependencyPath, "/")
}

func getDependencyName(dependencyPath string) string {
	dependencyName := filepath.Base(dependencyPath)
	dir := filepath.Dir(dependencyPath)

	// Check if the parent directory starts with "@", meaning that the dependency has a scope.
	// See https://docs.npmjs.com/cli/v10/using-npm/scope
	if strings.HasPrefix(filepath.Base(dir), "@") {
		// Prefix the dependency name with the scope
		return filepath.Join(filepath.Base(dir), dependencyName)
	}

	return dependencyName
}
