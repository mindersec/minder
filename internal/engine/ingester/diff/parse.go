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
	"strings"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

type ecosystemParser func(string) ([]*pb.Dependency, error)

func newEcosystemParser(eco DependencyEcosystem) ecosystemParser {
	switch eco {
	case DepEcosystemNPM:
		return npmParse
	case DepEcosystemGo:
		return goParse
	case DepEcosystemNone:
		return nil
	default:
		return nil
	}
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
