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
// Package rule provides the CLI subcommand for managing rules

package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleValidationError is used to report errors from evaluating a rule, including
// attribution of the particular error encountered.
type RuleValidationError struct {
	Err string
	// RuleType is a rule name
	RuleType string
}

// String implements fmt.Stringer
func (e *RuleValidationError) String() string {
	return fmt.Sprintf("error in rule %q: %s", e.RuleType, e.Err)
}

// Error implements error.Error
func (e *RuleValidationError) Error() string {
	return e.String()
}

// ParseYAML parses a YAML pipeline profile and validates it
func ParseYAML(r io.Reader) (*pb.Profile, error) {
	w := &bytes.Buffer{}
	if err := jsonyaml.TranscodeYAMLToJSON(r, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}
	return ParseJSON(w)
}

// ParseJSON parses a JSON pipeline profile and validates it
func ParseJSON(r io.Reader) (*pb.Profile, error) {
	var out pb.Profile

	dec := json.NewDecoder(r)
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("error validating profile: %w", err)
	}

	return &out, nil
}

// ReadProfileFromFile reads a pipeline profile from a file and returns it as a protobuf
func ReadProfileFromFile(fpath string) (*pb.Profile, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	defer f.Close()
	var out *pb.Profile

	if filepath.Ext(fpath) == ".json" {
		out, err = ParseJSON(f)
	} else {
		// parse yaml by default
		out, err = ParseYAML(f)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing profile: %w", err)
	}

	return out, nil
}

// GetRulesForEntity returns the rules for the given entity
func GetRulesForEntity(p *pb.Profile, entity pb.Entity) ([]*pb.Profile_Rule, error) {
	switch entity {
	case pb.Entity_ENTITY_REPOSITORIES:
		return p.Repository, nil
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return p.BuildEnvironment, nil
	case pb.Entity_ENTITY_ARTIFACTS:
		return p.Artifact, nil
	case pb.Entity_ENTITY_PULL_REQUESTS:
		return p.PullRequest, nil
	case pb.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}

// TraverseAllRulesForPipeline traverses all rules for the given pipeline profile
func TraverseAllRulesForPipeline(p *pb.Profile, fn func(*pb.Profile_Rule) error) error {
	if err := TraverseRules(p.Repository, fn); err != nil {
		return fmt.Errorf("error traversing repository rules: %w", err)
	}

	if err := TraverseRules(p.BuildEnvironment, fn); err != nil {
		return fmt.Errorf("error traversing build environment rules: %w", err)
	}

	if err := TraverseRules(p.PullRequest, fn); err != nil {
		return fmt.Errorf("error traversing pull_request rules: %w", err)
	}

	if err := TraverseRules(p.Artifact, fn); err != nil {
		return fmt.Errorf("error traversing artifact rules: %w", err)
	}

	return nil
}

// TraverseRules traverses the rules and calls the given function for each rule
// TODO: do we want to collect and return _all_ errors, rather than just the first,
// to prevent whack-a-mole fixing?
func TraverseRules(rules []*pb.Profile_Rule, fn func(*pb.Profile_Rule) error) error {
	for _, rule := range rules {
		if err := fn(rule); err != nil {
			return &RuleValidationError{err.Error(), rule.GetType()}
		}
	}

	return nil
}

// MergeDatabaseListIntoProfiles merges the database list profiles into the given
// profiles map. This assumes that the profiles belong to the same project.
//
// TODO(jaosorior): This will have to consider the project tree once we	migrate to that
func MergeDatabaseListIntoProfiles(ppl []db.ListProfilesByProjectIDRow) map[string]*pb.Profile {
	profiles := map[string]*pb.Profile{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Project ID (Unique index),
		// so we don't need to worry about collisions.
		// first we check if profile already exists, if not we create a new one
		if _, ok := profiles[p.Name]; !ok {
			profileID := p.ID.String()
			project := p.ProjectID.String()

			displayName := p.DisplayName
			if displayName == "" {
				displayName = p.Name
			}

			profiles[p.Name] = &pb.Profile{
				Id:          &profileID,
				Name:        p.Name,
				DisplayName: displayName,
				Context: &pb.Context{
					Provider: &p.Provider,
					Project:  &project,
				},
			}

			if p.Remediate.Valid {
				profiles[p.Name].Remediate = proto.String(string(p.Remediate.ActionType))
			} else {
				profiles[p.Name].Remediate = proto.String(string(db.ActionTypeOff))
			}

			if p.Alert.Valid {
				profiles[p.Name].Alert = proto.String(string(p.Alert.ActionType))
			} else {
				profiles[p.Name].Alert = proto.String(string(db.ActionTypeOn))
			}
		}
		if pm := rowInfoToProfileMap(profiles[p.Name], p.Entity, p.ContextualRules); pm != nil {
			profiles[p.Name] = pm
		}
	}

	return profiles
}

// MergeDatabaseGetIntoProfiles merges the database get profiles into the given
// profiles map. This assumes that the profiles belong to the same project.
//
// TODO(jaosorior): This will have to consider the project tree once we migrate to that
func MergeDatabaseGetIntoProfiles(ppl []db.GetProfileByProjectAndIDRow) map[string]*pb.Profile {
	profiles := map[string]*pb.Profile{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Project ID (Unique index),
		// so we don't need to worry about collisions.

		// first we check if profile already exists, if not we create a new one
		if _, ok := profiles[p.Name]; !ok {
			profileID := p.ID.String()
			project := p.ProjectID.String()

			displayName := p.DisplayName
			if displayName == "" {
				displayName = p.Name
			}

			profiles[p.Name] = &pb.Profile{
				Id:          &profileID,
				Name:        p.Name,
				DisplayName: displayName,
				Context: &pb.Context{
					Provider: &p.Provider,
					Project:  &project,
				},
			}

			if p.Remediate.Valid {
				profiles[p.Name].Remediate = proto.String(string(p.Remediate.ActionType))
			} else {
				profiles[p.Name].Remediate = proto.String(string(db.ActionTypeOff))
			}

			if p.Alert.Valid {
				profiles[p.Name].Alert = proto.String(string(p.Alert.ActionType))
			} else {
				profiles[p.Name].Alert = proto.String(string(db.ActionTypeOn))
			}
		}
		if pm := rowInfoToProfileMap(profiles[p.Name], p.Entity, p.ContextualRules); pm != nil {
			profiles[p.Name] = pm
		}
	}

	return profiles
}

// rowInfoToProfileMap adds the database row information to the given map of
// profiles. This assumes that the profiles belong to the same project.
// Note that this function is thought to be called from scpecific Merge functions
// and thus the logic is targetted to that.
func rowInfoToProfileMap(
	profile *pb.Profile,
	entity db.Entities,
	contextualRules json.RawMessage,
) *pb.Profile {
	if !entities.EntityTypeFromDB(entity).IsValid() {
		log.Printf("unknown entity found in database: %s", entity)
		return nil
	}

	var ruleset []*pb.Profile_Rule

	if err := json.Unmarshal(contextualRules, &ruleset); err != nil {
		// We merely print the error and continue. This is because the user
		// can't do anything about it and it's not a critical error.
		log.Printf("error unmarshalling contextual rules; there is corruption in the database: %s", err)
		return nil
	}

	switch entities.EntityTypeFromDB(entity) {
	case pb.Entity_ENTITY_REPOSITORIES:
		profile.Repository = ruleset
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		profile.BuildEnvironment = ruleset
	case pb.Entity_ENTITY_ARTIFACTS:
		profile.Artifact = ruleset
	case pb.Entity_ENTITY_PULL_REQUESTS:
		profile.PullRequest = ruleset
	case pb.Entity_ENTITY_UNSPECIFIED:
		// This shouldn't happen
		log.Printf("unknown entity found in database: %s", entity)
	}

	return profile
}
