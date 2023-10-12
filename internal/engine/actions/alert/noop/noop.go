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
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// Alert is the structure backing the noop alert
type Alert struct {
	actionType string
}

// NewNoopAlert creates a new noop alert engine
func NewNoopAlert(actionType string) (*Alert, error) {
	return &Alert{actionType: actionType}, nil
}

// Type returns the action type of the noop engine
func (a *Alert) Type() string {
	return a.actionType
}

// GetState returns the off state of the noop engine
func (_ *Alert) GetState(_ *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptOff
}

// IsSkippable returns true if the alert is skippable
func (_ *Alert) IsSkippable(_ interfaces.ActionOpt, _ error) bool {
	return true
}

// Do perform the noop alert
func (a *Alert) Do(
	_ context.Context,
	_ interfaces.ActionOpt,
	_ protoreflect.ProtoMessage,
	_ map[string]any,
	_ map[string]any,
) error {
	return fmt.Errorf("%s:%w", a.Type(), enginerr.ErrActionNotAvailable)
}
