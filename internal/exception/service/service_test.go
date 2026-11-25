// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	input := Exception{
		EntityID:       entityID,
		ProfileName:    "default",
		RuleName:       "vulnerability",
		Reason:         "Accepted risk",
		ReasonProvided: true,
		ExpiresAt:      expiresAt,
	}

	expected := db.Exception{
		ID:          uuid.New(),
		ProjectID:   projectID,
		EntityID:    entityID,
		ProfileName: "default",
		RuleName:    "vulnerability",
		Reason:      "Accepted risk",
		ExpiresAt:   expiresAt,
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{
			ID:        entityID,
			ProjectID: projectID,
		}, nil)

	store.EXPECT().
		CreateException(gomock.Any(), db.CreateExceptionParams{
			ProjectID:      projectID,
			EntityID:       entityID,
			ProfileName:    "default",
			RuleName:       "vulnerability",
			Reason:         "Accepted risk",
			ReasonProvided: true,
			ExpiresAt:      expiresAt,
		}).
		Return(expected, nil)

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

	require.NoError(t, err)
	require.Equal(t, expected.ID, got.ID)
	require.Equal(t, expected.EntityID, got.EntityID)
	require.Equal(t, expected.ProfileName, got.ProfileName)
	require.Equal(t, expected.RuleName, got.RuleName)
	require.Equal(t, expected.Reason, got.Reason)
	require.Equal(t, expected.ExpiresAt, got.ExpiresAt)
	require.Equal(t, expected.CreatedAt, got.CreatedAt)
}

func TestCreateReasonMaxLength(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)
	reason := strings.Repeat("a", MaxReasonLength)

	input := Exception{
		EntityID:       entityID,
		ProfileName:    "default",
		RuleName:       "vulnerability",
		Reason:         reason,
		ReasonProvided: true,
		ExpiresAt:      expiresAt,
	}

	expected := db.Exception{
		ID:          uuid.New(),
		ProjectID:   projectID,
		EntityID:    entityID,
		ProfileName: "default",
		RuleName:    "vulnerability",
		Reason:      reason,
		ExpiresAt:   expiresAt,
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{
			ID:        entityID,
			ProjectID: projectID,
		}, nil)

	store.EXPECT().
		CreateException(gomock.Any(), db.CreateExceptionParams{
			ProjectID:      projectID,
			EntityID:       entityID,
			ProfileName:    "default",
			RuleName:       "vulnerability",
			Reason:         reason,
			ReasonProvided: true,
			ExpiresAt:      expiresAt,
		}).
		Return(expected, nil)

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

	require.NoError(t, err)
	require.Equal(t, reason, got.Reason)
}

func TestCreateReasonTooLong(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	input := Exception{
		Reason: strings.Repeat("a", MaxReasonLength+1),
	}

	got, err := svc.Create(context.Background(), input)

	require.ErrorContains(t, err, "reason must not exceed")
	require.Nil(t, got)
}

func TestCreateReasonOmitted(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	input := Exception{
		EntityID:    entityID,
		ProfileName: "default",
		RuleName:    "vulnerability",
		ExpiresAt:   expiresAt,
	}

	expected := db.Exception{
		ID:          uuid.New(),
		ProjectID:   projectID,
		EntityID:    entityID,
		ProfileName: "default",
		RuleName:    "vulnerability",
		Reason:      "Previously accepted risk",
		ExpiresAt:   expiresAt,
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{
			ID:        entityID,
			ProjectID: projectID,
		}, nil)

	store.EXPECT().
		CreateException(gomock.Any(), db.CreateExceptionParams{
			ProjectID:      projectID,
			EntityID:       entityID,
			ProfileName:    "default",
			RuleName:       "vulnerability",
			Reason:         "",
			ReasonProvided: false,
			ExpiresAt:      expiresAt,
		}).
		Return(expected, nil)

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

	require.NoError(t, err)
	require.Equal(t, "Previously accepted risk", got.Reason)
}

func TestCreateEntityWrongProject(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()

	input := Exception{
		EntityID:    entityID,
		ProfileName: "default",
		RuleName:    "vulnerability",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{
			ID:        entityID,
			ProjectID: uuid.New(),
		}, nil)

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

	require.Error(t, err)
	require.Nil(t, got)
}

func TestCreateEntityLookupError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()

	input := Exception{
		EntityID:  entityID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{}, errors.New("entity not found"))

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

	require.Error(t, err)
	require.Nil(t, got)
}

func TestCreateError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	svc := NewService(store)

	projectID := uuid.New()
	entityID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)
	dbErr := errors.New("database error")

	input := Exception{
		EntityID:       entityID,
		ProfileName:    "default",
		RuleName:       "vulnerability",
		Reason:         "Accepted risk",
		ReasonProvided: true,
		ExpiresAt:      expiresAt,
	}

	store.EXPECT().
		GetEntityByID(gomock.Any(), entityID).
		Return(db.EntityInstance{
			ID:        entityID,
			ProjectID: projectID,
		}, nil)

	store.EXPECT().
		CreateException(gomock.Any(), gomock.Any()).
		Return(db.Exception{}, dbErr)

	ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{
			ID: projectID,
		},
	})

	got, err := svc.Create(ctx, input)

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
			ID:          uuid.New(),
			ProjectID:   projectID,
			EntityID:    uuid.New(),
			ProfileName: "default",
			RuleName:    "vulnerability",
			Reason:      "Accepted risk",
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		},
	}

	store.EXPECT().
		ListExceptions(gomock.Any(), projectID).
		Return(expected, nil)

	got, err := svc.List(context.Background(), projectID)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, expected[0].ID, got[0].ID)
	require.Equal(t, expected[0].EntityID, got[0].EntityID)
	require.Equal(t, expected[0].ProfileName, got[0].ProfileName)
	require.Equal(t, expected[0].RuleName, got[0].RuleName)
	require.Equal(t, expected[0].Reason, got[0].Reason)
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
		Return(int64(1), nil)

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
		Return(int64(0), dbErr)

	err := svc.Delete(context.Background(), id, projectID)

	require.ErrorIs(t, err, dbErr)
}

func TestDeleteNotFound(t *testing.T) {
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
		Return(int64(0), nil)

	err := svc.Delete(context.Background(), id, projectID)

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}
