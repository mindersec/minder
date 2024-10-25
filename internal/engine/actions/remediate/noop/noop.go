// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides a fallback remediation engine for cases where
// no remediation is set.
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

// Remediator is the structure backing the noop remediator
type Remediator struct {
	actionType interfaces.ActionType
}

// NewNoopRemediate creates a new noop remediation engine
func NewNoopRemediate(actionType interfaces.ActionType) (*Remediator, error) {
	return &Remediator{actionType: actionType}, nil
}

// Class returns the action type of the noop engine
func (r *Remediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *Remediator) Type() string {
	return "noop"
}

// GetOnOffState returns the off state of the noop engine
func (_ *Remediator) GetOnOffState(_ models.ActionOpt) models.ActionOpt {
	return models.ActionOptOff
}

// Do perform the remediation
func (r *Remediator) Do(
	_ context.Context,
	_ interfaces.ActionCmd,
	_ models.ActionOpt,
	_ protoreflect.ProtoMessage,
	_ interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
	return nil, fmt.Errorf("%s:%w", r.Class(), enginerr.ErrActionNotAvailable)
}
