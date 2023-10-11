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
	"errors"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/interfaces"
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

	projectID := inf.ProjectID

	// get project info
	project, err := e.querier.GetProjectByID(ctx, *projectID)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	provider, err := e.querier.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      inf.Provider,
		ProjectID: *projectID,
	})

	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	cli, err := providers.GetProviderBuilder(ctx, provider, *projectID, e.querier, e.crypteng)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Project: Project{
			ID:   project.ID,
			Name: project.Name,
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
	// Get profiles relevant to group
	dbpols, err := e.querier.ListProfilesByProjectID(ctx, *inf.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting profiles: %w", err)
	}

	for _, pol := range MergeDatabaseListIntoProfiles(dbpols, ectx) {
		profileID, err := uuid.Parse(*pol.Id)
		if err != nil {
			return fmt.Errorf("error parsing profile ID: %w", err)
		}
		remAction := interfaces.ActionOptFromString(pol.Remediate)

		// Get only these rules that are relevant for this entity type
		relevant, err := GetRulesForEntity(pol, inf.Type)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this profile
		err = TraverseRules(relevant, func(rule *pb.Profile_Rule) error {
			rt, rte, err := e.getEvaluator(ctx, profileID, ectx.Provider.Name, cli, ectx, rule)
			if err != nil {
				return err
			}

			ruleTypeID, err := uuid.Parse(*rt.Id)
			if err != nil {
				return fmt.Errorf("error parsing rule type ID: %w", err)
			}

			evalResult, remediateResult := rte.Eval(ctx, inf.Entity, rule.Def.AsMap(), rule.Params.AsMap(), remAction)

			logEval(ctx, pol, rule, inf, evalResult, remediateResult)

			return e.createOrUpdateEvalStatus(ctx, inf.evalStatusParams(
				profileID, ruleTypeID, evalResult, remediateResult))
		})

		if err != nil {
			p := pol.Name
			if pol.Id != nil {
				p = *pol.Id
			}
			return fmt.Errorf("error traversing rules for profile %s: %w", p, err)
		}
	}
	return nil
}

func (e *Executor) getEvaluator(
	ctx context.Context,
	profileID uuid.UUID,
	prov string,
	cli *providers.ProviderBuilder,
	ectx *EntityContext,
	rule *pb.Profile_Rule,
) (*pb.RuleType, *RuleTypeEngine, error) {
	log.Printf("Evaluating rule: %s for profile %s", rule.Type, profileID)

	dbrt, err := e.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  prov,
		ProjectID: ectx.Project.ID,
		Name:      rule.Type,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error getting rule type when traversing profile %s: %w", profileID, err)
	}

	rt, err := RuleTypePBFromDB(&dbrt, ectx)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing rule type when traversing profile %s: %w", profileID, err)
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
	pol *pb.Profile,
	rule *pb.Profile_Rule,
	inf *EntityInfoWrapper,
	evalResult error,
	remediateResult error,
) {
	logger := zerolog.Ctx(ctx).Debug().
		Str("profile", pol.Name).
		Str("ruleType", rule.Type).
		Str("projectId", inf.ProjectID.String()).
		Str("repositoryId", inf.OwnershipData[RepositoryIDEventKey])

	if aID, ok := inf.OwnershipData[ArtifactIDEventKey]; ok {
		logger = logger.Str("artifactId", aID)
	}

	logger.Err(evalResult).Msg("evaluated rule")

	if errors.Is(remediateResult, evalerrors.ErrRemediationSkipped) {
		logger.Msg("remediation skipped")
	} else if errors.Is(remediateResult, evalerrors.ErrRemediationNotAvailable) {
		logger.Msg("remediation not supported")
	} else {
		logger.Err(remediateResult).Msg("remediated rule")
	}
}
