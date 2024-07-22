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

// Package noop provides a fallback alert engine for cases where
// no alert is set.
package noop

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/profiles/models"
)

// Alert is the structure backing the noop alert
type Alert struct {
	actionType interfaces.ActionType
}

// NewNoopAlert creates a new noop alert engine
func NewNoopAlert(actionType interfaces.ActionType) (*Alert, error) {
	return &Alert{actionType: actionType}, nil
}

// Class returns the action type of the noop engine
func (a *Alert) Class() interfaces.ActionType {
	return a.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *Alert) Type() string {
	return "noop"
}

// GetOnOffState returns the off state of the noop engine
func (_ *Alert) GetOnOffState(_ models.ActionOpt) models.ActionOpt {
	return models.ActionOptOff
}

// Do perform the noop alert
func (a *Alert) Do(
	_ context.Context,
	_ interfaces.ActionCmd,
	_ models.ActionOpt,
	_ protoreflect.ProtoMessage,
	_ interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
	return nil, fmt.Errorf("%s:%w", a.Class(), enginerr.ErrActionNotAvailable)
}
