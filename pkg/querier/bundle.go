// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
)

// BundleSubscription represents a bundle subscription
type BundleSubscription struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	BundleID       uuid.UUID
	CurrentVersion string
}

// BundleHandlers interface provides functions to interact with bundles and subscriptions
type BundleHandlers interface {
	GetSubscriptionByProjectBundle(
		ctx context.Context,
		projectID uuid.UUID,
		bundleNamespace, bundleName string,
	) (*BundleSubscription, error)
	SetCurrentVersion(ctx context.Context, projectID uuid.UUID, currentVersion string) error
}

// SetCurrentVersion sets the current version of the bundle for a project
func (q *querierType) SetCurrentVersion(ctx context.Context, projectID uuid.UUID, currentVersion string) error {
	if q.querier == nil {
		return ErrQuerierMissing
	}
	return q.querier.SetSubscriptionBundleVersion(ctx, db.SetSubscriptionBundleVersionParams{
		ProjectID:      projectID,
		CurrentVersion: currentVersion,
	})
}

// GetSubscriptionByProjectBundle gets a subscription by project bundle
func (q *querierType) GetSubscriptionByProjectBundle(
	ctx context.Context,
	projectID uuid.UUID,
	bundleNamespace, bundleName string,
) (*BundleSubscription, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	ret, err := q.querier.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: bundleNamespace,
		Name:      bundleName,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	return &BundleSubscription{
		ID:             ret.ID,
		ProjectID:      ret.ProjectID,
		BundleID:       ret.BundleID,
		CurrentVersion: ret.CurrentVersion,
	}, nil
}
