// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package api provides utility functions for working with the Minder APIs.
// Currently, this provides "upsert" methods for a few API calls that support
// Create and Update methods -- Upsert attempts a create, and falls back to
// update if the create fails with AlreadyExists.
package api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// We could try to do this with generics, but there are a lot of distinct types and names,
// and I think we'd end up being sad.  Instead, we repeat the same code three times.

// UpsertProfile creates or updates a profile using the supplied client.
func UpsertProfile(
	ctx context.Context, client minderv1.ProfileServiceClient, profile *minderv1.Profile,
) error {
	_, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
		Profile: profile,
	})
	if err == nil {
		return nil
	}
	rpcStatus, _ := status.FromError(err)
	if rpcStatus.Code() == codes.AlreadyExists {
		_, err = client.UpdateProfile(ctx, &minderv1.UpdateProfileRequest{
			Profile: profile,
		})
	}
	return err
}

// UpsertRuleType creates or updates a ruleType using the supplied client.
func UpsertRuleType(
	ctx context.Context, client minderv1.RuleTypeServiceClient, ruleType *minderv1.RuleType,
) error {
	_, err := client.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
		RuleType: ruleType,
	})
	if err == nil {
		return nil
	}
	// If not a grpc error, this will become grpc.Unknown with the original error message
	rpcStatus, _ := status.FromError(err)
	if rpcStatus.Code() == codes.AlreadyExists {
		_, err = client.UpdateRuleType(ctx, &minderv1.UpdateRuleTypeRequest{
			RuleType: ruleType,
		})
	}
	return err
}

// UpsertDataSource creates or updates a dataSource using the supplied client.
func UpsertDataSource(
	ctx context.Context, client minderv1.DataSourceServiceClient, dataSource *minderv1.DataSource,
) error {
	_, err := client.CreateDataSource(ctx, &minderv1.CreateDataSourceRequest{
		DataSource: dataSource,
	})
	if err == nil {
		return nil
	}
	// If not a grpc error, this will become grpc.Unknown with the original error message
	rpcStatus, _ := status.FromError(err)
	if rpcStatus.Code() == codes.AlreadyExists {
		_, err = client.UpdateDataSource(ctx, &minderv1.UpdateDataSourceRequest{
			DataSource: dataSource,
		})
	}
	return err
}
