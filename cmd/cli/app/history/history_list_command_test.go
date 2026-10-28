// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest
func TestListCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "table output with data",
			Args: []string{"history", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEvalResultsServiceClient(ctrl)
				mockResponse := &minderv1.ListEvaluationHistoryResponse{}
				cli.LoadFixture(t, "mock_history_response.json", mockResponse)
				client.EXPECT().
					ListEvaluationHistory(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EvalResultsServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.table",
		},
		{
			Name: "table output empty",
			Args: []string{"history", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEvalResultsServiceClient(ctrl)
				client.EXPECT().
					ListEvaluationHistory(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListEvaluationHistoryResponse{
						Data: []*minderv1.EvaluationHistory{},
					}, nil)
				return cli.WithRPCClient[minderv1.EvalResultsServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_empty.table",
		},
		{
			Name: "grpc error handling",
			Args: []string{"history", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEvalResultsServiceClient(ctrl)
				client.EXPECT().
					ListEvaluationHistory(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
				return cli.WithRPCClient[minderv1.EvalResultsServiceClient](context.Background(), client)
			},
			ExpectedError: "request timed out",
		},
		{
			Name:          "invalid output format",
			Args:          []string{"history", "list", "-o", "csv"},
			ExpectedError: "invalid argument",
		},
	}

	cli.RunCmdTests(t, tests, historyCmd)
}
