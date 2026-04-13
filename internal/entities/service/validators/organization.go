// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package validators

import (
"context"

"github.com/google/uuid"

"github.com/mindersec/minder/pkg/entities/properties"
)

// OrganizationValidator validates organization entity creation
type OrganizationValidator struct{}

// NewOrganizationValidator creates a new OrganizationValidator
func NewOrganizationValidator() *OrganizationValidator {
return &OrganizationValidator{}
}

// Validate checks if an organization entity can be created
func (v *OrganizationValidator) Validate(
_ context.Context,
_ *properties.Properties,
_ uuid.UUID,
) error {
// For now, any organization properties that make it this far are valid
return nil
}
