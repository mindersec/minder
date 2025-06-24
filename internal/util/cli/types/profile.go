// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/mindersec/minder/internal/util/cli/table"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type profileToDisplay struct {
	profile *minderv1.ProfileStatus
}

type profileStatusDetails struct{}

// GetDetails implements table.StatusDetails.
func (*profileStatusDetails) GetDetails() string {
	return ""
}

// GetStatus implements table.StatusDetails.
func (*profileStatusDetails) GetStatus() string {
	return ""
}

var _ table.StatusDetails = (*profileStatusDetails)(nil)

// GetStatus implements table.EvalStatus.
func (p *profileToDisplay) GetStatus() string {
	return p.profile.GetProfileStatus()
}

// GetStatusDetail implements table.EvalStatus.
func (*profileToDisplay) GetStatusDetail() string {
	return ""
}

// GetRemediationStatus implements table.EvalStatus.
func (*profileToDisplay) GetRemediationStatus() string {
	return ""
}

// GetRemediationDetail implements table.EvalStatus.
func (*profileToDisplay) GetRemediationDetail() string {
	return ""
}

// GetAlert implements table.StatusDetails.
func (*profileToDisplay) GetAlert() table.StatusDetails {
	return &profileStatusDetails{}
}

var _ table.EvalStatus = (*profileToDisplay)(nil)

// ProfileStatus converts a ProfileStatus for status display.
func ProfileStatus(p *minderv1.ProfileStatus) table.EvalStatus {
	return &profileToDisplay{profile: p}
}
