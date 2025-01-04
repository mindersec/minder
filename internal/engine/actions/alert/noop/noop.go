// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides a fallback alert engine for cases where
// no alert is set.
package noop

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
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
func (_ *Alert) GetOnOffState() models.ActionOpt {
	return models.ActionOptOff
}

// Do perform the noop alert
func (a *Alert) Do(
	_ context.Context,
	_ interfaces.ActionCmd,
	_ protoreflect.ProtoMessage,
	_ interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
	return nil, fmt.Errorf("%s:%w", a.Class(), enginerr.ErrActionNotAvailable)
}
