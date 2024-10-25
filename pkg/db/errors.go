// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"errors"

	"github.com/lib/pq"
)

// ErrIsUniqueViolation returns true if the error is a unique violation
func ErrIsUniqueViolation(err error) bool {
	return isPostgresError(err, "23505")
}

func isPostgresError(err error, code string) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code == pq.ErrorCode(code)
	}
	return false
}
