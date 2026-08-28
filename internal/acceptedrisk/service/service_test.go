// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
)

func TestCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	providerID := uuid.New()
	ruleTypeID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	req := CreateRequest{
		ProjectID:  projectID,
		ProviderID: providerID,
		EntityName: "owner/repo",
		RuleTypeID: ruleTypeID,
		ExpiresAt:  expiresAt,
	}

	expected := db.AcceptedRisk{
		ID:         uuid.New(),
		ProjectID:  projectID,
		ProviderID: providerID,
		EntityName: "owner/repo",
		RuleTypeID: ruleTypeID,
		ExpiresAt:  expiresAt,
	}

	store.EXPECT().
		CreateAcceptedRisk(gomock.Any(), db.CreateAcceptedRiskParams{
			ProjectID:  projectID,
			ProviderID: providerID,
			EntityName: "owner/repo",
			RuleTypeID: ruleTypeID,
			ExpiresAt:  expiresAt,
		}).
		Return(expected, nil)

	got, err := svc.Create(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, &expected, got)
}

func TestCreateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")

	req := CreateRequest{
		ProjectID:  uuid.New(),
		ProviderID: uuid.New(),
		EntityName: "owner/repo",
		RuleTypeID: uuid.New(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	store.EXPECT().
		CreateAcceptedRisk(gomock.Any(), gomock.Any()).
		Return(db.AcceptedRisk{}, dbErr)

	got, err := svc.Create(context.Background(), req)

	require.ErrorIs(t, err, dbErr)
	require.Nil(t, got)
}

func TestList(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	expected := []db.AcceptedRisk{
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			ProviderID: uuid.New(),
			EntityName: "owner/repo",
			RuleTypeID: uuid.New(),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}

	store.EXPECT().
		ListAcceptedRisks(gomock.Any(), projectID).
		Return(expected, nil)

	got, err := svc.List(context.Background(), projectID)

	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")
	projectID := uuid.New()

	store.EXPECT().
		ListAcceptedRisks(gomock.Any(), projectID).
		Return(nil, dbErr)

	got, err := svc.List(context.Background(), projectID)

	require.ErrorIs(t, err, dbErr)
	require.Nil(t, got)
}

func TestDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	id := uuid.New()
	projectID := uuid.New()

	store.EXPECT().
		DeleteAcceptedRisk(gomock.Any(), db.DeleteAcceptedRiskParams{
			ID:        id,
			ProjectID: projectID,
		}).
		Return(nil)

	err := svc.Delete(context.Background(), id, projectID)

	require.NoError(t, err)
}

func TestDeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")
	id := uuid.New()
	projectID := uuid.New()

	store.EXPECT().
		DeleteAcceptedRisk(gomock.Any(), db.DeleteAcceptedRiskParams{
			ID:        id,
			ProjectID: projectID,
		}).
		Return(dbErr)

	err := svc.Delete(context.Background(), id, projectID)

	require.ErrorIs(t, err, dbErr)
}
