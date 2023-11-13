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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	scanner := bufio.NewScanner(strings.NewReader(patch))
	var deps []*pb.Dependency

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "+") {
			fields := strings.Split(line, " ")
			if len(fields) > 2 && !strings.HasSuffix(fields[1], "/go.mod") {
				name, version := fields[0], fields[1]
				dep := &pb.Dependency{
					Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      name[1:],
					Version:   version,
				}

				deps = append(deps, dep)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return deps, nil
}

type npmDependency struct {
	Version      string                    `json:"version"`
	Resolved     string                    `json:"resolved"`
	Integrity    string                    `json:"integrity"`
	Requires     map[string]string         `json:"requires,omitempty"`
	Dependencies map[string]*npmDependency `json:"dependencies,omitempty"`
	Optional     bool                      `json:"optional,omitempty"`
}

type npmRoot struct {
	Deps map[string]*npmDependency
}

// TODO(jakub): this is a hacky way to parse the npm patch file
func npmParse(patch string) ([]*pb.Dependency, error) {
	lines := strings.Split(patch, "\n")
	var output strings.Builder

	// Write the start of the JSON object to the output
	output.WriteString("{\n")

	// Start of crazy code to parse the patch file
	// What we do here is first grab all the lines that start with "+"
	// Then we remove the "+" symbol and write the modified line to the output
	// We then add a { to the start of the output and a } to the end, so we have a valid JSON object
	// Then we convert the string builder to a string and unmarshal it into the Package struct
	for _, line := range lines {
		// Check if the line starts with "+"
		if strings.HasPrefix(line, "+") {
			// Remove the "+" symbol and write the modified line to the output
			line = strings.TrimPrefix(line, "+")
			output.WriteString(line + "\n")
		}
	}

	//if string ends with a comma, remove it
	outputString := strings.TrimSuffix(output.String(), ",\n")
	outputString = outputString + "\n}"

	// Convert the string builder to a string and unmarshal it into the Package struct
	root := &npmRoot{
		Deps: make(map[string]*npmDependency),
	}

	err := json.Unmarshal([]byte(outputString), &root.Deps)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal npm package: %w", err)
	}

	deps := make([]*pb.Dependency, 0, len(root.Deps))
	for name, dep := range root.Deps {
		deps = append(deps, &pb.Dependency{
			Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      name,
			Version:   dep.Version,
		})
	}
	return deps, nil
}
