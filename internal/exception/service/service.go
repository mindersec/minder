// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service contains the business logic for exceptions.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// CreateRequest contains the information required to create an exception.
type CreateRequest struct {
	ProjectID  uuid.UUID
	EntityID   uuid.UUID
	EntityName string
	RuleTypeID uuid.UUID
	ExpiresAt  time.Time
}

// Exception represents an exception.
type Exception struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	EntityID   uuid.UUID
	EntityName string
	RuleTypeID uuid.UUID
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// Service encapsulates logic related to exceptions.
type Service interface {
	Create(ctx context.Context, req CreateRequest) (*Exception, error)
	List(ctx context.Context, projectID uuid.UUID) ([]Exception, error)
	Delete(ctx context.Context, id uuid.UUID, projectID uuid.UUID) error
}

// Ensure that service implements Service.
var _ Service = (*exceptionService)(nil)

type exceptionService struct {
	store db.Store
}

// NewService creates a new exception service.
func NewService(store db.Store) Service {
	return &exceptionService{
		store: store,
	}
}

// Create creates an exception.
func (s *exceptionService) Create(
	ctx context.Context,
	req CreateRequest,
) (*Exception, error) {
	exception, err := s.store.CreateException(ctx, db.CreateExceptionParams{
		ProjectID:  req.ProjectID,
		EntityID:   req.EntityID,
		EntityName: req.EntityName,
		RuleTypeID: req.RuleTypeID,
		ExpiresAt:  req.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create exception: %w", err)
	}

	return &Exception{
		ID:         exception.ID,
		ProjectID:  exception.ProjectID,
		EntityID:   exception.EntityID,
		EntityName: exception.EntityName,
		RuleTypeID: exception.RuleTypeID,
		ExpiresAt:  exception.ExpiresAt,
		CreatedAt:  exception.CreatedAt,
	}, nil
}

// List lists active exceptions for a project.
func (s *exceptionService) List(
	ctx context.Context,
	projectID uuid.UUID,
) ([]Exception, error) {
	exceptions, err := s.store.ListExceptions(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list exceptions: %w", err)
	}

	result := make([]Exception, 0, len(exceptions))
	for _, exception := range exceptions {
		result = append(result, Exception{
			ID:         exception.ID,
			ProjectID:  exception.ProjectID,
			EntityID:   exception.EntityID,
			EntityName: exception.EntityName,
			RuleTypeID: exception.RuleTypeID,
			ExpiresAt:  exception.ExpiresAt,
			CreatedAt:  exception.CreatedAt,
		})
	}

	return result, nil
}

// Delete deletes an exception from a project.
func (s *exceptionService) Delete(
	ctx context.Context,
	id uuid.UUID,
	projectID uuid.UUID,
) error {
	if err := s.store.DeleteException(ctx, db.DeleteExceptionParams{
		ID:        id,
		ProjectID: projectID,
	}); err != nil {
		return fmt.Errorf("failed to delete exception: %w", err)
	}

	return nil
}
