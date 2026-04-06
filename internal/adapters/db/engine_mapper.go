// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package dbadapter provides adapter utilities for converting database models
// into engine domain types.
package dbadapter

import (
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/engine"
)

// MapDbToEngine converts a database evaluation row into an engine domain type.
func MapDbToEngine(row *db.ListRuleEvaluationsByProfileIdRow) *engine.EvaluationSnapshot {
	if row == nil {
		return nil
	}

	return &engine.EvaluationSnapshot{
		EvalStatus:        engine.EvalStatus(row.EvalStatus),
		RemediationStatus: string(row.RemStatus),
		AlertStatus:       string(row.AlertStatus),
	}
}
