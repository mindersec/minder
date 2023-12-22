// Copyright 2023 Stacklok, Inc.
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

package auth

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func userRegistered(ctx context.Context, client minderv1.UserServiceClient) (bool, *minderv1.GetUserResponse, error) {
	res, err := client.GetUser(ctx, &minderv1.GetUserRequest{})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return false, nil, nil
			}
		}
		return false, nil, fmt.Errorf("error retrieving user %w", err)
	}
	return true, res, nil
}

func renderNewUser(conn string, newUser *minderv1.CreateUserResponse) {
	t := table.New(table.Simple, layouts.KeyValue, nil)
	t.AddRow("Project ID", newUser.ProjectId)
	t.AddRow("Project Name", newUser.ProjectName)
	t.AddRow("Minder Server", conn)
	t.Render()
}

func renderUserInfo(conn string, user *minderv1.GetUserResponse) {
	t := table.New(table.Simple, layouts.KeyValue, nil)
	t.AddRow("Minder Server", conn)
	for _, project := range getProjectTableRows(user.Projects) {
		t.AddRow(project...)
	}
	t.Render()
}

func renderUserInfoWhoami(conn string, user *minderv1.GetUserResponse) {
	t := table.New(table.Simple, layouts.KeyValue, nil)
	t.AddRow("Subject", user.GetUser().GetIdentitySubject())
	t.AddRow("Created At", user.GetUser().GetCreatedAt().AsTime().String())
	t.AddRow("Updated At", user.GetUser().GetUpdatedAt().AsTime().String())
	t.AddRow("Minder Server", conn)
	for _, project := range getProjectTableRows(user.Projects) {
		t.AddRow(project...)
	}
	t.Render()
}

func getProjectTableRows(projects []*minderv1.Project) [][]string {
	var rows [][]string
	projectKey := "Project"
	for idx, project := range projects {
		if len(projects) > 1 {
			projectKey = fmt.Sprintf("Project #%d", idx+1)
		}
		projectVal := fmt.Sprintf("%s / %s", project.GetName(), project.GetProjectId())
		rows = append(rows, []string{projectKey, projectVal})
	}
	return rows
}
