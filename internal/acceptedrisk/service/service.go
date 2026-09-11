// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service contains the business logic for accepted risks.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// CreateRequest contains the information required to create an accepted risk.
type CreateRequest struct {
	ProjectID  uuid.UUID
	ProviderID uuid.UUID
	EntityName string
	RuleTypeID uuid.UUID
	ExpiresAt  time.Time
}

// Service encapsulates logic related to accepted risks.
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*db.AcceptedRisk, error)
	List(ctx context.Context, projectID uuid.UUID) ([]db.AcceptedRisk, error)
	Delete(ctx context.Context, id uuid.UUID, projectID uuid.UUID) error
}

// Ensure that service implements Service.
var _ Service = (*acceptedRiskService)(nil)

type acceptedRiskService struct {
	store db.Store
}

// NewService creates a new accepted risk service.
func NewService(store db.Store) Service {
	return &acceptedRiskService{
		store: store,
	}
}

// Create creates an accepted risk.
func (s *acceptedRiskService) Create(
	ctx context.Context,
	req CreateRequest,
) (*db.AcceptedRisk, error) {
	risk, err := s.store.CreateAcceptedRisk(ctx, db.CreateAcceptedRiskParams{
		ProjectID:  req.ProjectID,
		ProviderID: req.ProviderID,
		EntityName: req.EntityName,
		RuleTypeID: req.RuleTypeID,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create accepted risk: %w", err)
	}

	return &risk, nil
}

// List lists active accepted risks for a project.
func (s *acceptedRiskService) List(
	ctx context.Context,
	projectID uuid.UUID,
) ([]db.AcceptedRisk, error) {
	risks, err := s.store.ListAcceptedRisks(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accepted risks: %w", err)
	}

	return risks, nil
}

// Delete deletes an accepted risk from a project.
func (s *acceptedRiskService) Delete(
	ctx context.Context,
	id uuid.UUID,
	projectID uuid.UUID,
) error {
	if err := s.store.DeleteAcceptedRisk(ctx, db.DeleteAcceptedRiskParams{
		ID:        id,
		ProjectID: projectID,
	}); err != nil {
		return fmt.Errorf("failed to delete accepted risk: %w", err)
	}

	return nil
}
