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

// Package security_advisory provides necessary interfaces and implementations for
// creating alerts of type security advisory.
package security_advisory

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/db"
	enginerr "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

const (
	// SecurityAdvisoryType is the type of the security advisory alert engine
	SecurityAdvisoryType = "security-advisory"
)

// Alert is the structure backing the security-advisory alert action
type Alert struct {
	actionType string
	cli        provifv1.REST
}

// NewSecurityAdvisoryAlert creates a new security-advisory alert action
func NewSecurityAdvisoryAlert(
	actionType string,
	saCfg *pb.RuleType_Definition_Alert_AlertTypeSA,
	pbuild *providers.ProviderBuilder,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}
	cli, err := pbuild.GetHTTP(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot get http client: %w", err)
	}
	_ = saCfg
	return &Alert{
		actionType: actionType,
		cli:        cli,
	}, nil
}

// Type returns the action type of the security-advisory engine
func (alert *Alert) Type() string {
	return alert.actionType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Alert) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Alert)
}

// IsSkippable returns true if the alert is skippable
func (_ *Alert) IsSkippable(ctx context.Context, actionState interfaces.ActionOpt, evalErr error) bool {
	// TODO: Implement this
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msgf("Alert action: evaluating if %s should be skipped: %v, %d", SecurityAdvisoryType, evalErr, actionState)
	return false
}

// Do alerts through security advisory
func (alert *Alert) Do(
	ctx context.Context,
	setting interfaces.ActionOpt,
	entity protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
	dbEvalStatus db.ListRuleEvaluationsByProfileIdRow,
) error {
	// TODO: Implement this
	_ = setting
	_ = entity
	_ = ruleDef
	_ = ruleParams
	_ = dbEvalStatus
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msgf("Alert action: processing %s action for %s", SecurityAdvisoryType, entity)

	// 1. Prepare the current alert state (from db)
	// 2. Prepare the new alert state from rule evaluation parameters
	// 3. If the new rule evaluation state is failing AND we have an alert triggered, return
	// 4. If the new rule evaluation state is failing AND we don't have an alert, create an alert
	// 5. If the new rule evaluation state is passing AND we have an alert triggered, close the alert
	// 6. If the new rule evaluation state is passing AND we don't have an alert triggered, return
	// 7. Process the alert
	// 8. If the alert is created, save the alert ID in the alert details database table
	// 9. If the alert is closed, remove the alert ID from the alert details database table
	return fmt.Errorf("%s:%w", alert.Type(), enginerr.ErrActionNotAvailable)
}
