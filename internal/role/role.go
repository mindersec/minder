//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package role

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
)

type CreateRoleValidation struct {
	GroupId int32  `db:"group_id" validate:"required"`
	Name    string `db:"name" validate:"required"`
}

func CreateRole(ctx context.Context, store db.Store, group_id int32, name string,
	is_admin *bool, is_protected *bool) (*db.Role, error) {
	// validate role
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{GroupId: group_id, Name: name})
	if err != nil {
		return nil, err
	}

	if is_admin == nil {
		is_admin = new(bool)
		*is_admin = false
	}

	if is_protected == nil {
		is_protected = new(bool)
		*is_protected = false
	}
	org, err := store.CreateRole(ctx, db.CreateRoleParams{GroupID: group_id, Name: name,
		IsAdmin: *is_admin, IsProtected: *is_protected})
	if err != nil {
		return nil, err
	}
	return &org, nil
}
