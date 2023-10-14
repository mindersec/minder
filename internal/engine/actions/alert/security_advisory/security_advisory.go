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
	actionType interfaces.ActionType
	cli        provifv1.REST
}

// NewSecurityAdvisoryAlert creates a new security-advisory alert action
func NewSecurityAdvisoryAlert(
	actionType interfaces.ActionType,
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
func (alert *Alert) Type() interfaces.ActionType {
	return alert.actionType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Alert) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Alert)
}

// Do alerts through security advisory
func (alert *Alert) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	setting interfaces.ActionOpt,
	entity protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
) error {
	// TODO: Implement this
	_ = setting
	_ = entity
	_ = ruleDef
	_ = ruleParams
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("alert_type", SecurityAdvisoryType).
		Str("cmd", string(cmd)).
		Msg("alert action not implemented")
	return fmt.Errorf("%s:%w", alert.Type(), enginerr.ErrActionNotAvailable)
}
