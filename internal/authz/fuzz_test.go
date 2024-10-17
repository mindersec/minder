// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package authz_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz"
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
