// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ruletypes

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// TODO: the implementation of this interface should contain logic from the
// controlplane methods for rule type create/update/delete
type RuleTypeService interface {
	CreateRule(ctx context.Context, newRule *v1.RuleType) error
	CreateSubscriptionRule(ctx context.Context, newRule *v1.RuleType, bundleID uuid.UUID) error
	UpdateRule(ctx context.Context, ruleID uuid.UUID, updatedRule *v1.RuleType) error
	UpdateSubscriptionRule(ctx context.Context, ruleID uuid.UUID, updatedRule *v1.RuleType, bundleID uuid.UUID) error
	DeleteRule(ctx context.Context, ruleID uuid.UUID) error
	DeleteSubscriptionRule(ctx context.Context, ruleID uuid.UUID, subscriptionID uuid.UUID)
}
