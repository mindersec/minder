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
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()
	ruleTypeID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	req := CreateRequest{
		ProjectID:  projectID,
		EntityID:   entityID,
		EntityName: "owner/repo",
		RuleTypeID: ruleTypeID,
		ExpiresAt:  expiresAt,
	}

	expected := db.Exception{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityID:   entityID,
		EntityName: "owner/repo",
		RuleTypeID: ruleTypeID,
		ExpiresAt:  expiresAt,
	}

	store.EXPECT().
		CreateException(gomock.Any(), db.CreateExceptionParams{
			ProjectID:  projectID,
			EntityID:   entityID,
			EntityName: "owner/repo",
			RuleTypeID: ruleTypeID,
			ExpiresAt:  expiresAt,
		}).
		Return(expected, nil)

	got, err := svc.Create(context.Background(), req)

	require.NoError(t, err)
	require.Equal(t, expected.ID, got.ID)
	require.Equal(t, expected.ProjectID, got.ProjectID)
	require.Equal(t, expected.EntityID, got.EntityID)
	require.Equal(t, expected.EntityName, got.EntityName)
	require.Equal(t, expected.RuleTypeID, got.RuleTypeID)
	require.Equal(t, expected.ExpiresAt, got.ExpiresAt)
	require.Equal(t, expected.CreatedAt, got.CreatedAt)
}

func TestCreateError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")

	req := CreateRequest{
		ProjectID:  uuid.New(),
		EntityID:   uuid.New(),
		EntityName: "owner/repo",
		RuleTypeID: uuid.New(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	store.EXPECT().
		CreateException(gomock.Any(), gomock.Any()).
		Return(db.Exception{}, dbErr)

	got, err := svc.Create(context.Background(), req)

	require.ErrorIs(t, err, dbErr)
	require.Nil(t, got)
}

func TestList(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	expected := []db.Exception{
		{
			ID:         uuid.New(),
			ProjectID:  projectID,
			EntityID:   uuid.New(),
			EntityName: "owner/repo",
			RuleTypeID: uuid.New(),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		},
	}

	store.EXPECT().
		ListExceptions(gomock.Any(), projectID).
		Return(expected, nil)

	got, err := svc.List(context.Background(), projectID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, expected[0].ID, got[0].ID)
	require.Equal(t, expected[0].ProjectID, got[0].ProjectID)
	require.Equal(t, expected[0].EntityID, got[0].EntityID)
	require.Equal(t, expected[0].EntityName, got[0].EntityName)
	require.Equal(t, expected[0].RuleTypeID, got[0].RuleTypeID)
	require.Equal(t, expected[0].ExpiresAt, got[0].ExpiresAt)
	require.Equal(t, expected[0].CreatedAt, got[0].CreatedAt)
}

func TestListError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")
	projectID := uuid.New()

	store.EXPECT().
		ListExceptions(gomock.Any(), projectID).
		Return(nil, dbErr)

	got, err := svc.List(context.Background(), projectID)

	require.ErrorIs(t, err, dbErr)
	require.Nil(t, got)
}

func TestDelete(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	id := uuid.New()
	projectID := uuid.New()

	store.EXPECT().
		DeleteException(gomock.Any(), db.DeleteExceptionParams{
			ID:        id,
			ProjectID: projectID,
		}).
		Return(nil)

	err := svc.Delete(context.Background(), id, projectID)

	require.NoError(t, err)
}

func TestDeleteError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	dbErr := errors.New("database error")
	id := uuid.New()
	projectID := uuid.New()

	store.EXPECT().
		DeleteException(gomock.Any(), db.DeleteExceptionParams{
			ID:        id,
			ProjectID: projectID,
		}).
		Return(dbErr)

	err := svc.Delete(context.Background(), id, projectID)

	require.ErrorIs(t, err, dbErr)
}
