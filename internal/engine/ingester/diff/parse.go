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

	// block is the go.mod directive block ("require", "replace", ...) the
	// current line is in, or "" when it isn't known. Block headers and their
	// closing parentheses appear as context or added lines; a hunk can start
	// inside a block whose header isn't part of the patch.
	block := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "@@") {
			block = ""
			continue
		}
		// Removed lines are not in the new go.mod.
		if strings.HasPrefix(line, "-") {
			continue
		}
		added := strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++")
		content := line
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, " ") {
			content = line[1:]
		}
		if b, ok := goModBlockStart(content); ok {
			block = b
			continue
		}
		if strings.TrimSpace(content) == ")" {
			block = ""
			continue
		}
		if !added {
			continue
		}

		// Parse the line and extract a dependency
		dep := extractGoDepFromLine(content, block)

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

// goModBlockStart reports whether line opens a go.mod directive block, such
// as "require (", and returns the directive.
func goModBlockStart(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[1] != "(" {
		return "", false
	}
	return fields[0], true
}

// extractGoDepFromLine returns the dependency a go.mod line adds, or nil.
// line is the line without its diff marker, and block is the directive block
// it is in, or "" when unknown. Only require and replace directives name
// dependencies; exclude, retract, tool and godebug lines don't.
func extractGoDepFromLine(line, block string) *pbinternal.Dependency {
	// We ignore lines that contain "// indirect" because they are transitive dependencies, and therefore
	// not actionable.
	if strings.Contains(line, "// indirect") {
		return nil
	}
	if i := strings.Index(line, "//"); i != -1 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}

	directive, args := fields[0], fields[1:]
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		// An indented line is inside a block. When the block header isn't in
		// the patch, only a replacement has "=>" in it.
		directive, args = block, fields
		if directive == "" {
			directive = "require"
			if slices.Contains(fields, "=>") {
				directive = "replace"
			}
		}
	}

	switch directive {
	case "require":
		if len(args) < 2 {
			return nil
		}
		return &pbinternal.Dependency{Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO, Name: args[0], Version: args[1]}
	case "replace":
		// module [version] => replacement [version]. The replacement is what
		// is built; without a version it is a local directory, not a module.
		arrow := slices.Index(args, "=>")
		if arrow == -1 || len(args[arrow+1:]) != 2 {
			return nil
		}
		return &pbinternal.Dependency{Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO, Name: args[arrow+1], Version: args[arrow+2]}
	}
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
// If the parent key does not correspond to a dependency (i.e. it could be the root project), then an empty string is
// returned.
func findDependencyName(i int, lines []string) string {
	closingBraces := 0
	for j := i - 1; j >= 0; j-- {
		if strings.Contains(lines[j], "}") {
			closingBraces = closingBraces + 1
		}
		if strings.Contains(lines[j], "{") {
			if closingBraces == 0 {
				matches := dependencyNameRegex.FindStringSubmatch(lines[j])
				if len(matches) > 1 {
					// extract the dependency name from the key, which is the dependency path
					dependencyPath := matches[1]
					return getDependencyName(dependencyPath)
				}
				return ""
			}
			closingBraces = closingBraces - 1
		}
	}
	return ""
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
