// Copyright 2023 Stacklok, Inc.
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

// Package diff provides the diff rule data ingest engine
package diff

import (
	"bufio"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	versionRegex        = regexp.MustCompile(`^\+\s*"version"\s*:\s*"([^"\n]*)"\s*(?:,|$)`)
	dependencyNameRegex = regexp.MustCompile(`\s*"([^"]+)"\s*:\s*{\s*`)
)

type ecosystemParser func(string) ([]*pb.Dependency, error)

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

func requirementsParse(patch string) ([]*pb.Dependency, error) {
	var deps []*pb.Dependency

	scanner := bufio.NewScanner(strings.NewReader(patch))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		line = pyReqNormalizeLine(line)
		if line == "" {
			continue
		}

		matches := util.PyRequestsVersionRegexp.FindAllStringSubmatch(line, -1)
		// For now, if no version is set, we just don't set the version. The evaluator
		// will skip the dependency, but later we can query the package repository for the
		// latest version
		if matches == nil {
			deps = pyReqAddPkgName(deps, line, "")
			continue
		}

		// this is probably a bit confusing. What we're trying to do here is to find the
		// lowest version in the line. If there is a > or >= operator, we use that, because
		// then the lower version is set explicitly. If there is no > or >= operator, we
		// just use the first version we find.
		version := ""
		var lowestVersion string
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			if version == "" {
				version = match[2]
			}
			if match[1] == ">" || match[1] == ">=" || match[1] == "==" {
				lowestVersion = match[2]
			}
		}
		if lowestVersion != "" {
			version = lowestVersion
		}

		// Extract the name by grabbing everything up to the first operator
		nameMatch := util.PyRequestsNameRegexp.FindStringIndex(line)
		if nameMatch != nil {
			// requests   ==2.19.0 is apparently a valid line, so we need to trim the whitespace
			name := strings.TrimSpace(line[:nameMatch[0]])
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

func pyReqAddPkgName(depList []*pb.Dependency, pkgName, version string) []*pb.Dependency {
	dep := &pb.Dependency{
		Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_PYPI,
		Name:      pyNormalizeName(pkgName),
		Version:   version,
	}
	return append(depList, dep)
}

func pyNormalizeName(pkgName string) string {
	regex := regexp.MustCompile(`[-_.]+`)
	result := regex.ReplaceAllString(pkgName, "-")
	return strings.ToLower(result)
}

func goParse(patch string) ([]*pb.Dependency, error) {
	var deps []*pb.Dependency
	scanner := bufio.NewScanner(strings.NewReader(patch))

	// Iterate over the lines of the go.mod patch and parse the dependencies
	for scanner.Scan() {
		// Parse the line and extract a dependency
		dep := extractGoDepFromPatchLine(scanner.Text())

		// If we failed to extract a dependency, or if it's already in the slice, skip it
		if dep == nil || slices.ContainsFunc(deps, func(n *pb.Dependency) bool {
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

func extractGoDepFromPatchLine(line string) *pb.Dependency {
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

		dep := &pb.Dependency{
			Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_GO,
		}
		if fields[0] == "require" && fields[1] != "(" {
			if len(fields) < 3 {
				return nil
			}
			dep.Name = fields[1]
			dep.Version = fields[2]
		} else if strings.HasPrefix(lineContent, "\t") {
			dep.Name = fields[0]
			dep.Version = fields[1]
		} else if fields[0] == "replace" && strings.Contains(lineContent, "=>") && len(fields) >= 5 {
			if len(fields) < 5 {
				return nil
			}
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

func npmParse(patch string) ([]*pb.Dependency, error) {
	lines := strings.Split(patch, "\n")

	var deps []*pb.Dependency

	for i, line := range lines {
		// Check if the line contains a version
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]
			name := findDependencyName(i, lines)
			// The version is not always a dependency version. It may also be the version of the package in this repo,
			// or the version of the root project. See https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
			if name != "" {
				deps = append(deps, &pb.Dependency{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
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
