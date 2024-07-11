// Copyright 2024 Stacklok, Inc.
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

// PgTime/PgTimeArray is used to work around the pq driver's inability to map
// TIMESTAMPZ[] to []time.Time without some hand holding.
// Code taken mostly as-is from: https://github.com/lib/pq/issues/536#issuecomment-397849980

package db

import (
	"database/sql/driver"
	"errors"
	"time"

	"github.com/lib/pq"
)

// ErrParseData signifies that an error occurred while parsing SQL data
var ErrParseData = errors.New("unable to parse SQL data")

// PgTime wraps a time.Time
type PgTime struct{ time.Time }

// Scan implements the sql.Scanner interface
func (t *PgTime) Scan(val interface{}) error {
	switch v := val.(type) {
	case time.Time:
		t.Time = v
		return nil
	case []uint8: // byte is the same as uint8: https://golang.org/pkg/builtin/#byte
		_t, err := pq.ParseTimestamp(nil, string(v))
		if err != nil {
			return ErrParseData
		}
		t.Time = _t
		return nil
	case string:
		_t, err := pq.ParseTimestamp(nil, v)
		if err != nil {
			return ErrParseData
		}
		t.Time = _t
		return nil
	}
	return ErrParseData
}

// Value implements the driver.Valuer interface
func (t *PgTime) Value() (driver.Value, error) { return pq.FormatTimestamp(t.Time), nil }

// PgTimeArray wraps a time.Time slice to be used as a Postgres array
// type PgTimeArray []time.Time
type PgTimeArray []PgTime

//type PgTimeArray []pq.NullTime

// Scan implements the sql.Scanner interface
func (a *PgTimeArray) Scan(src interface{}) error {
	return pq.GenericArray{A: a}.Scan(src)
}

// Value implements the driver.Valuer interface
func (a *PgTimeArray) Value() (driver.Value, error) {
	return pq.GenericArray{A: a}.Value()
}
