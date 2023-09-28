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

package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

const (
	// InternalEntityEventTopic is the topic for internal webhook events
	InternalEntityEventTopic = "internal.entity.event"
)

// Executor is the engine that executes the rules for a given event
type Executor struct {
	querier  db.Store
	crypteng *crypto.Engine
}

// NewExecutor creates a new executor
func NewExecutor(querier db.Store, authCfg *config.AuthConfig) (*Executor, error) {
	crypteng, err := crypto.EngineFromAuthConfig(authCfg)
	if err != nil {
		return nil, err
	}

	return &Executor{
		querier:  querier,
		crypteng: crypteng,
	}, nil
}

// Register implements the Consumer interface.
func (e *Executor) Register(r events.Registrar) {
	r.Register(InternalEntityEventTopic, e.HandleEntityEvent)
}

// HandleEntityEvent handles events coming from webhooks/signals
// as well as the init event.
func (e *Executor) HandleEntityEvent(msg *message.Message) error {
	inf, err := parseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	ctx := msg.Context()

	// get group info
	group, err := e.querier.GetGroupByID(ctx, inf.GroupID)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	provider, err := e.querier.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    inf.Provider,
		GroupID: inf.GroupID,
	})

	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	cli, err := providers.GetProviderBuilder(ctx, provider, inf.GroupID, e.querier, e.crypteng)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Group: Group{
			ID:   group.ID,
			Name: group.Name,
		},
		Provider: Provider{
			Name: inf.Provider,
			ID:   provider.ID,
		},
	}

	return e.evalEntityEvent(ctx, inf, ectx, cli)
}

func (e *Executor) evalEntityEvent(
	ctx context.Context,
	inf *EntityInfoWrapper,
	ectx *EntityContext,
	cli *providers.ProviderBuilder,
) error {
	// Get policies relevant to group
	dbpols, err := e.querier.ListPoliciesByGroupID(ctx, inf.GroupID)
	if err != nil {
		return fmt.Errorf("error getting policies: %w", err)
	}

	for _, pol := range MergeDatabaseListIntoPolicies(dbpols, ectx) {
		policyID, err := uuid.Parse(*pol.Id)
		if err != nil {
			return fmt.Errorf("error parsing policy ID: %w", err)
		}

		// Given we're dealing with a repository event, we can assume that the
		// entity is a repository.
		relevant, err := GetRulesForEntity(pol, inf.Type)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}
		// Handle artifact entities separately than other entities like repositories and pull requests
		if inf.Type != pb.Entity_ENTITY_ARTIFACTS {
			// Evaluate rules for repository and pull request entities
			// Let's evaluate all the rules for this policy
			err = TraverseRules(relevant, func(rule *pb.Policy_Rule) error {
				rt, rte, err := e.getEvaluator(ctx, policyID, ectx.Provider.Name, cli, ectx, rule)
				if err != nil {
					return err
				}

				ruleTypeID, err := uuid.Parse(*rt.Id)
				if err != nil {
					return fmt.Errorf("error parsing rule type ID: %w", err)
				}

				result := rte.Eval(ctx, inf.Entity, rule.Def.AsMap(), rule.Params.AsMap())

				logEval(ctx, pol, rule, inf, result)

				return e.createOrUpdateEvalStatus(ctx, inf.evalStatusParams(
					policyID, ruleTypeID, result))
			})
		} else {
			// Evaluate rules for artifact entities
			repoID, err := uuid.Parse(inf.OwnershipData[RepositoryIDEventKey])
			if err != nil {
				return fmt.Errorf("error parsing rule type ID: %w", err)
			}
			// Get repository data - we need the owner and name
			dbrepo, err := e.querier.GetRepositoryByID(ctx, repoID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("repository not found")
			} else if err != nil {
				return fmt.Errorf("cannot read repository: %v", err)
			}
			artifactID, err := uuid.Parse(inf.OwnershipData[ArtifactIDEventKey])
			if err != nil {
				return fmt.Errorf("error parsing rule type ID: %w", err)
			}
			// Retrieve artifact details
			artifact, err := e.querier.GetArtifactByID(ctx, artifactID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("artifact not found")
			} else if err != nil {
				return fmt.Errorf("failed to get artifact: %v", err)
			}

			// Get all artifact versions that we want to evaluate for that given artifact
			dbArtifactVersions, err := e.querier.ListArtifactVersionsByArtifactID(ctx, db.ListArtifactVersionsByArtifactIDParams{
				ArtifactID: artifact.ID,
				Limit:      sql.NullInt32{Valid: false},
			})
			if err != nil {
				log.Printf("error getting artifact versions for artifact %d: %v", artifact.ID, err)
				return err
			}

			// 1. Traverse each rule within this policy
			err = TraverseRules(relevant, func(rule *pb.Policy_Rule) error {
				// this is used to flag we found a matching artifact and also completed processing the rule
				var ruleStatusReady = false
				// this is the result of the last rule evaluation
				var ruleResult error = nil
				// this is the rule type of the last rule evaluation
				var rt *pb.RuleType = nil

				// 2. Loop through all artifact versions and run the same rule against each version
				for _, dbVersion := range dbArtifactVersions {
					var tags []string
					if dbVersion.Tags.Valid {
						tags = strings.Split(dbVersion.Tags.String, ",")
					}
					sigVer := &pb.SignatureVerification{}
					if dbVersion.SignatureVerification.Valid {
						if err := protojson.Unmarshal(dbVersion.SignatureVerification.RawMessage, sigVer); err != nil {
							log.Printf("error unmarshalling signature verification: %v", err)
							continue
						}
					}
					ghWorkflow := &pb.GithubWorkflow{}
					if dbVersion.GithubWorkflow.Valid {
						if err := protojson.Unmarshal(dbVersion.GithubWorkflow.RawMessage, ghWorkflow); err != nil {
							log.Printf("error unmarshalling gh workflow: %v", err)
							continue
						}
					}
					versionedArtifact := &pb.VersionedArtifact{
						Artifact: &pb.Artifact{
							ArtifactPk: artifact.ID.String(),
							Owner:      dbrepo.RepoOwner,
							Name:       artifact.ArtifactName,
							Type:       artifact.ArtifactType,
							Visibility: artifact.ArtifactVisibility,
							Repository: dbrepo.RepoName,
							CreatedAt:  timestamppb.New(artifact.CreatedAt),
						},
						Version: &pb.ArtifactVersion{
							VersionId:             dbVersion.Version,
							Tags:                  tags,
							Sha:                   dbVersion.Sha,
							SignatureVerification: sigVer,
							GithubWorkflow:        ghWorkflow,
							CreatedAt:             timestamppb.New(dbVersion.CreatedAt),
						},
					}

					var rte *RuleTypeEngine
					rt, rte, err = e.getEvaluator(ctx, policyID, ectx.Provider.Name, cli, ectx, rule)
					if err != nil {
						return err
					}

					// 3. Evaluate the rule for this versioned artifact
					ruleResult = rte.Eval(ctx, versionedArtifact, rule.Def.AsMap(), rule.Params.AsMap())
					// 4. Process the result of the rule evaluation
					if ruleResult == nil {
						// We found a matching versioned artifact and rule evaluation passed for it.
						// No need to continue evaluating the rest of the artifacts and their versions
						ruleStatusReady = true
						break
					} else if errors.Is(ruleResult, evalerrors.ErrEvaluationFailed) {
						// We found a matching artifact version, but rule evaluation failed.
						// There's no need to continue evaluating this rule anymore
						ruleStatusReady = true
						break
					} else if errors.Is(ruleResult, evalerrors.ErrEvaluationSkipped) {
						// We found a matching versioned artifact, but rule evaluation was skipped.
						// There's no need to continue evaluating this rule anymore
						ruleStatusReady = true
						break
					} else if errors.Is(ruleResult, evalerrors.ErrEvaluationSkipSilently) {
						// No matching artifact version found.
						// Continue evaluating the rest of the versioned artifacts
						continue
					} else {
						// Rule evaluation failed for some other reason, no need to continue
						ruleStatusReady = true
						break
					}
				}
				// 5. Handle the case where there were no existing artifacts matching the rule. Default to rule failure.
				if !ruleStatusReady && errors.Is(ruleResult, evalerrors.ErrEvaluationSkipSilently) {
					ruleResult = evalerrors.NewErrEvaluationFailed("no matching artifact found, failing rule")
				}
				logEval(ctx, pol, rule, inf, ruleResult)

				ruleTypeID, err := uuid.Parse(*rt.Id)
				if err != nil {
					return fmt.Errorf("error parsing rule type ID: %w", err)
				}

				// 6. Update the rule status in the database
				return e.createOrUpdateEvalStatus(ctx, inf.evalStatusParams(policyID, ruleTypeID, ruleResult))
			})
		}

		if err != nil {
			p := pol.Name
			if pol.Id != nil {
				p = *pol.Id
			}

			return fmt.Errorf("error traversing rules for policy %s: %w", p, err)
		}

	}

	return nil
}

func (e *Executor) getEvaluator(
	ctx context.Context,
	policyID uuid.UUID,
	prov string,
	cli *providers.ProviderBuilder,
	ectx *EntityContext,
	rule *pb.Policy_Rule,
) (*pb.RuleType, *RuleTypeEngine, error) {
	log.Printf("Evaluating rule: %s for policy %s", rule.Type, policyID)

	dbrt, err := e.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: prov,
		GroupID:  ectx.Group.ID,
		Name:     rule.Type,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error getting rule type when traversing policy %s: %w", policyID, err)
	}

	rt, err := RuleTypePBFromDB(&dbrt, ectx)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing rule type when traversing policy %s: %w", policyID, err)
	}

	// TODO(jaosorior): Rule types should be cached in memory so
	// we don't have to query the database for each rule.
	rte, err := NewRuleTypeEngine(rt, cli)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	return rt, rte, nil
}

func logEval(
	ctx context.Context,
	pol *pb.Policy,
	rule *pb.Policy_Rule,
	inf *EntityInfoWrapper,
	result error,
) {
	logger := zerolog.Ctx(ctx).Debug().
		Str("policy", pol.Name).
		Str("ruleType", rule.Type).
		Int32("groupId", inf.GroupID).
		Str("repositoryId", inf.OwnershipData[RepositoryIDEventKey])

	if aID, ok := inf.OwnershipData[ArtifactIDEventKey]; ok {
		logger = logger.Str("artifactId", aID)
	}

	logger.Err(result).Msg("evaluated rule")
}
