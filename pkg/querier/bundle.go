// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// BundleHandlers interface provides functions to interact with bundles and subscriptions
type BundleHandlers interface {
	GetSubscriptionByProjectBundle(
		ctx context.Context,
		projectID uuid.UUID,
		bundleNamespace, bundleName string,
	) (*pb.BundleSubscription, error)
	SetCurrentVersion(ctx context.Context, projectID uuid.UUID, currentVersion string) error
}

// SetCurrentVersion sets the current version of the bundle for a project
func (t *Type) SetCurrentVersion(ctx context.Context, projectID uuid.UUID, currentVersion string) error {
	return t.db.querier.SetCurrentVersion(ctx, db.SetCurrentVersionParams{
		ProjectID:      projectID,
		CurrentVersion: currentVersion,
	})
}

// GetSubscriptionByProjectBundle gets a subscription by project bundle
func (t *Type) GetSubscriptionByProjectBundle(
	ctx context.Context,
	projectID uuid.UUID,
	bundleNamespace, bundleName string,
) (*pb.BundleSubscription, error) {
	ret, err := t.db.querier.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: bundleNamespace,
		Name:      bundleName,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	return &pb.BundleSubscription{
		Id:             ret.ID.String(),
		ProjectId:      ret.ProjectID.String(),
		BundleId:       ret.BundleID.String(),
		CurrentVersion: ret.CurrentVersion,
	}, nil
}
