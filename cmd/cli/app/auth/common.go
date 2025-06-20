// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	t := table.New(table.Simple, layouts.Default, []string{"Key", "Value"})
	t.AddRow("Subject", newUser.GetIdentitySubject())
	t.AddRow("Project ID", newUser.ProjectId)
	t.AddRow("Project Name", newUser.ProjectName)
	t.AddRow("Minder Server", conn)
	t.Render()
}

func renderUserInfo(conn string, user *minderv1.GetUserResponse) {
	t := table.New(table.Simple, layouts.Default, []string{"Key", "Value"})
	t.AddRow("Minder Server", conn)
	t.AddRow("Subject", user.GetUser().GetIdentitySubject())
	for _, project := range getProjectTableRows(user.GetProjectRoles()) {
		t.AddRow(project...)
	}
	t.Render()
}

func renderUserInfoWhoami(conn string, outWriter io.Writer, format string, user *minderv1.GetUserResponse) {
	switch format {
	case app.Table:
		fmt.Fprintln(outWriter, cli.Header.Render("Here are your details:"))
		t := table.New(table.Simple, layouts.Default, []string{"Key", "Value"})
		t.AddRow("Subject", user.GetUser().GetIdentitySubject())
		t.AddRow("Created At", user.GetUser().GetCreatedAt().AsTime().String())
		t.AddRow("Updated At", user.GetUser().GetUpdatedAt().AsTime().String())
		t.AddRow("Minder Server", conn)
		for _, project := range getProjectTableRows(user.GetProjectRoles()) {
			t.AddRow(project...)
		}
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to JSON: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	case app.YAML:
		out, err := util.GetYamlFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to YAML: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	}
}

func getProjectTableRows(projects []*minderv1.ProjectRole) [][]string {
	var rows [][]string
	projectKey := "Project"
	for idx, project := range projects {
		if len(projects) > 1 {
			projectKey = fmt.Sprintf("Project #%d", idx+1)
		}
		projectVal := fmt.Sprintf("%s / %s", project.GetProject().GetName(), project.GetProject().GetProjectId())
		rows = append(rows, []string{fmt.Sprintf("%s (role: %s)", projectKey, project.GetRole().GetName()), projectVal})
	}
	return rows
}
