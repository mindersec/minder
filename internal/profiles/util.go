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

package profiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/sqlc-dev/pqtype"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)

var multipleSpacesRegex = regexp.MustCompile(`\s{2,}`)

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
	case pb.Entity_ENTITY_RELEASE:
		return p.Release, nil
	case pb.Entity_ENTITY_PIPELINE_RUN:
		return p.PipelineRun, nil
	case pb.Entity_ENTITY_TASK_RUN:
		return p.TaskRun, nil
	case pb.Entity_ENTITY_BUILD:
		return p.Build, nil
	case pb.Entity_ENTITY_UNSPECIFIED:
		return nil, fmt.Errorf("entity type unspecified")
	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}

// TraverseRuleTypesForEntities traverses the rules for the given entities and calls the given function
func TraverseRuleTypesForEntities(p *pb.Profile, fn func(pb.Entity, *pb.Profile_Rule) error) error {
	pairs := map[pb.Entity][]*pb.Profile_Rule{
		pb.Entity_ENTITY_REPOSITORIES:       p.Repository,
		pb.Entity_ENTITY_BUILD_ENVIRONMENTS: p.BuildEnvironment,
		pb.Entity_ENTITY_ARTIFACTS:          p.Artifact,
		pb.Entity_ENTITY_PULL_REQUESTS:      p.PullRequest,
		pb.Entity_ENTITY_RELEASE:            p.Release,
		pb.Entity_ENTITY_PIPELINE_RUN:       p.PipelineRun,
		pb.Entity_ENTITY_TASK_RUN:           p.TaskRun,
		pb.Entity_ENTITY_BUILD:              p.Build,
	}

	for entity, rules := range pairs {
		for _, rule := range rules {
			if err := fn(entity, rule); err != nil {
				return err
			}
		}
	}

	return nil
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
func MergeDatabaseListIntoProfiles[T db.ProfileRow](ppl []T) map[string]*pb.Profile {
	profiles := map[string]*pb.Profile{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Project ID (Unique index),
		// so we don't need to worry about collisions.
		// first we check if profile already exists, if not we create a new one
		if _, ok := profiles[p.GetProfile().Name]; !ok {
			profileID := p.GetProfile().ID.String()
			project := p.GetProfile().ProjectID.String()

			displayName := p.GetProfile().DisplayName
			if displayName == "" {
				displayName = p.GetProfile().Name
			}

			profiles[p.GetProfile().Name] = &pb.Profile{
				Id:          &profileID,
				Name:        p.GetProfile().Name,
				DisplayName: displayName,
				Context: &pb.Context{
					Project: &project,
				},
			}

			if p.GetProfile().Remediate.Valid {
				profiles[p.GetProfile().Name].Remediate = proto.String(string(p.GetProfile().Remediate.ActionType))
			} else {
				profiles[p.GetProfile().Name].Remediate = proto.String(string(db.ActionTypeOff))
			}

			if p.GetProfile().Alert.Valid {
				profiles[p.GetProfile().Name].Alert = proto.String(string(p.GetProfile().Alert.ActionType))
			} else {
				profiles[p.GetProfile().Name].Alert = proto.String(string(db.ActionTypeOn))
			}

			selectorsToProfile(profiles[p.GetProfile().Name], p.GetSelectors())
		}
		if pm := rowInfoToProfileMap(
			profiles[p.GetProfile().Name], p.GetEntityProfile(),
			p.GetContextualRules()); pm != nil {
			profiles[p.GetProfile().Name] = pm
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
		if _, ok := profiles[p.Profile.Name]; !ok {
			profiles[p.Profile.Name] = dbProfileToPB(p.Profile)
			selectorsToProfile(profiles[p.Profile.Name], p.ProfilesWithSelectors)
		}

		if pm := rowInfoToProfileMap(
			profiles[p.Profile.Name],
			p.ProfilesWithEntityProfile.Entity,
			p.ProfilesWithEntityProfile.ContextualRules,
		); pm != nil {
			profiles[p.Profile.Name] = pm
		}
	}

	return profiles
}

// MergeDatabaseGetByNameIntoProfiles merges the database get profiles into the given
// profiles map. This assumes that the profiles belong to the same project.
//
// TODO: This will have to consider the project tree once we migrate to that
func MergeDatabaseGetByNameIntoProfiles(ppl []db.GetProfileByProjectAndNameRow) map[string]*pb.Profile {
	profiles := map[string]*pb.Profile{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Project ID (Unique index),
		// so we don't need to worry about collisions.

		// first we check if profile already exists, if not we create a new one
		if _, ok := profiles[p.Profile.Name]; !ok {
			profiles[p.Profile.Name] = dbProfileToPB(p.Profile)
			selectorsToProfile(profiles[p.Profile.Name], p.ProfilesWithSelectors)
		}

		if pm := rowInfoToProfileMap(
			profiles[p.Profile.Name],
			p.ProfilesWithEntityProfile.Entity,
			p.ProfilesWithEntityProfile.ContextualRules,
		); pm != nil {
			profiles[p.Profile.Name] = pm
		}
	}

	return profiles
}

// DeriveProfileNameFromDisplayName derives the profile name from the profile display name
func DeriveProfileNameFromDisplayName(
	profile *pb.Profile,
) (name string) {

	displayName := profile.GetDisplayName()
	name = profile.GetName()

	if displayName != "" && name == "" {
		// when a display name is provided, but no profile name
		// then the profile name is created and saved based on the profile display name
		return cleanDisplayName(displayName)
	}
	// when both a display name and a profile name are provided
	// then the profile name from the incoming request is used as the profile name
	// but the project already has a profile with that name

	return name

}

// The profile name should be derived from the profile display name given the following logic
func cleanDisplayName(displayName string) string {

	// Trim leading and trailing whitespace
	displayName = strings.TrimSpace(displayName)

	// Remove non-alphanumeric characters

	displayName = nonAlphanumericRegex.ReplaceAllString(displayName, "")

	// Replace multiple spaces with a single space
	displayName = multipleSpacesRegex.ReplaceAllString(displayName, " ")

	// Replace all whitespace with underscores
	displayName = strings.ReplaceAll(displayName, " ", "_")

	// Convert to lower-case
	displayName = strings.ToLower(displayName)

	// Trim to a maximum length of 63 characters
	if len(displayName) > 63 {
		displayName = displayName[:63]
	}

	return displayName
}

func dbProfileToPB(p db.Profile) *pb.Profile {
	profileID := p.ID.String()
	project := p.ProjectID.String()

	displayName := p.DisplayName
	if displayName == "" {
		displayName = p.Name
	}

	outprof := &pb.Profile{
		Id:          &profileID,
		Name:        p.Name,
		DisplayName: displayName,
		Context: &pb.Context{
			Project: &project,
		},
	}

	if p.Remediate.Valid {
		outprof.Remediate = proto.String(string(p.Remediate.ActionType))
	} else {
		outprof.Remediate = proto.String(string(db.ActionTypeOff))
	}

	if p.Alert.Valid {
		outprof.Alert = proto.String(string(p.Alert.ActionType))
	} else {
		outprof.Alert = proto.String(string(db.ActionTypeOn))
	}

	return outprof
}

// rowInfoToProfileMap adds the database row information to the given map of
// profiles. This assumes that the profiles belong to the same project.
// Note that this function is thought to be called from scpecific Merge functions
// and thus the logic is targetted to that.
func rowInfoToProfileMap(
	profile *pb.Profile,
	maybeEntity db.NullEntities,
	maybeContextualRules pqtype.NullRawMessage,
) *pb.Profile {
	if !maybeEntity.Valid || !maybeContextualRules.Valid {
		// empty profile. Just return without filling in the rules
		return profile
	}
	entity := maybeEntity.Entities
	contextualRules := maybeContextualRules.RawMessage

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
	case pb.Entity_ENTITY_RELEASE:
		profile.Release = ruleset
	case pb.Entity_ENTITY_PIPELINE_RUN:
		profile.PipelineRun = ruleset
	case pb.Entity_ENTITY_TASK_RUN:
		profile.TaskRun = ruleset
	case pb.Entity_ENTITY_BUILD:
		profile.Build = ruleset
	case pb.Entity_ENTITY_UNSPECIFIED:
		// This shouldn't happen
		log.Printf("unknown entity found in database: %s", entity)
	}

	return profile
}

func selectorsToProfile(
	profile *pb.Profile,
	selectors []db.ProfileSelector,
) {
	profile.Selection = make([]*pb.Profile_Selector, 0, len(selectors))
	for _, s := range selectors {
		profile.Selection = append(profile.Selection, &pb.Profile_Selector{
			Id:          s.ID.String(),
			Entity:      string(s.Entity.Entities),
			Selector:    s.Selector,
			Description: s.Comment,
		})
	}
}
