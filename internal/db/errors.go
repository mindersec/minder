// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"errors"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

// ErrIsUniqueViolation returns true if the error is a unique violation
func ErrIsUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pq.Error](err)
	return ok && pgErr.Code == pqerror.UniqueViolation
}
