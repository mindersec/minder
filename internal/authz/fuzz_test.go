//
// Copyright 2024 Stacklok, Inc.
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

package authz_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
)

//nolint:gosec // This test does not validate return values
func FuzzAllAuthzApis(f *testing.F) {
	f.Fuzz(func(t *testing.T, str1, str2, str3, str4, str5, str6 string) {
		c, stopFunc := newOpenFGAServerAndClient(t)
		defer stopFunc()
		ctx := context.Background()
		err := c.MigrateUp(ctx)
		if err != nil {
			panic(err.Error())
		}

		err = c.PrepareForRun(ctx)
		if err != nil {
			panic(err.Error())
		}
		prj := uuid.New()

		c.Write(ctx, str1, authz.RoleAdmin, prj)

		userJWT := openid.New()

		err = userJWT.Set(str2, str1)
		if err != nil {
			return
		}

		userctx := jwt.WithAuthTokenContext(ctx, userJWT)

		c.Check(userctx, str3, prj)

		c.ProjectsForUser(userctx, str4)
		c.AssignmentsToProject(userctx, prj)
		c.Delete(ctx, str5, authz.RoleAdmin, prj)
		c.Check(userctx, "get", prj)
		c.ProjectsForUser(userctx, str6)
		c.AssignmentsToProject(userctx, prj)
	})
}
