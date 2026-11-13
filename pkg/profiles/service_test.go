// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mocksel "github.com/mindersec/minder/pkg/engine/selectors/mock"
	"github.com/mindersec/minder/pkg/eventer/constants"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
	"github.com/mindersec/minder/pkg/profiles"
)

type (
	dbMockBuilder func(store *mockdb.MockStore, pub *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker)
)

func newTestEnv(t *testing.T) (*mockdb.MockStore, *mockevents.MockPublisher, *mocksel.MockSelectionChecker, profiles.ProfileService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	pub := mockevents.NewMockPublisher(ctrl)
	sel := mocksel.NewMockSelectionChecker(ctrl)
	svc := profiles.NewProfileService(pub, sel)
	return store, pub, sel, svc
}

func TestDeleteProfile(t *testing.T) {
	t.Parallel()

	projID := uuid.New()
	targetProfID := uuid.New()

	scenarios := []struct {
		Name          string
		Target        string
		DBSetup       dbMockBuilder
		ExpectedError string
		ValidateRes   func(t *testing.T, res *db.Profile)
	}{
		{
			Name:   "successfully deletes profile by UUID string",
			Target: targetProfID.String(),
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(db.Profile{
					ID:             targetProfID,
					ProjectID:      projID,
					Name:           "profile-to-delete",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().DeleteProfile(gomock.Any(), db.DeleteProfileParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(nil)
			},
			ValidateRes: func(t *testing.T, res *db.Profile) {
				t.Helper()
				require.NotNil(t, res)
				assert.Equal(t, targetProfID, res.ID)
				assert.Equal(t, "profile-to-delete", res.Name)
			},
		},
		{
			Name:   "successfully deletes profile by name when profile is not a valid UUID",
			Target: "named-profile",
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByNameAndLock(gomock.Any(), db.GetProfileByNameAndLockParams{
					ProjectID: projID,
					Name:      "named-profile",
				}).Return(db.Profile{
					ID:             targetProfID,
					ProjectID:      projID,
					Name:           "named-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().DeleteProfile(gomock.Any(), db.DeleteProfileParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(nil)
			},
			ValidateRes: func(t *testing.T, res *db.Profile) {
				t.Helper()
				require.NotNil(t, res)
				assert.Equal(t, targetProfID, res.ID)
				assert.Equal(t, "named-profile", res.Name)
			},
		},
		{
			Name:   "returns NotFound when profile ID does not exist in DB",
			Target: targetProfID.String(),
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(db.Profile{}, sql.ErrNoRows)
			},
			ExpectedError: fmt.Sprintf("profile %q not found", targetProfID.String()),
		},
		{
			Name:   "returns NotFound when profile name does not exist in DB",
			Target: "nonexistent-name",
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByNameAndLock(gomock.Any(), db.GetProfileByNameAndLockParams{
					ProjectID: projID,
					Name:      "nonexistent-name",
				}).Return(db.Profile{}, sql.ErrNoRows)
			},
			ExpectedError: `profile "nonexistent-name" not found`,
		},
		{
			Name:   "returns error when fetching profile encounters internal database error",
			Target: targetProfID.String(),
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(db.Profile{}, errors.New("connection refused"))
			},
			ExpectedError: "error fetching profile to be deleted",
		},
		{
			Name:   "rejects deletion of bundle subscription profiles",
			Target: targetProfID.String(),
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(db.Profile{
					ID:             targetProfID,
					ProjectID:      projID,
					Name:           "bundle-profile",
					SubscriptionID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
				}, nil)
			},
			ExpectedError: "cannot delete profile from bundle",
		},
		{
			Name:   "returns error when DeleteProfile query execution fails in DB",
			Target: targetProfID.String(),
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(db.Profile{
					ID:             targetProfID,
					ProjectID:      projID,
					Name:           "profile-to-delete",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().DeleteProfile(gomock.Any(), db.DeleteProfileParams{
					ProjectID: projID,
					ID:        targetProfID,
				}).Return(errors.New("deadlock detected"))
			},
			ExpectedError: "failed to delete profile: deadlock detected",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			store, pub, sel, svc := newTestEnv(t)
			if sc.DBSetup != nil {
				sc.DBSetup(store, pub, sel)
			}

			res, err := svc.DeleteProfile(context.Background(), projID, sc.Target, store)
			if sc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), sc.ExpectedError)
			} else {
				require.NoError(t, err)
				if sc.ValidateRes != nil {
					sc.ValidateRes(t, res)
				}
			}
		})
	}
}

func TestCreateProfile(t *testing.T) {
	t.Parallel()

	rawRuleDef, err := loadRawRuleTypeDef()
	require.NoError(t, err)

	projID := uuid.New()
	ruleTypeID := uuid.New()
	ruleTypeParam := dbMockWithRuleTypeID(rawRuleDef, ruleTypeID)

	scenarios := []struct {
		Name           string
		SubscriptionID uuid.UUID
		ProfileArg     func() *minderv1.Profile
		DBSetup        dbMockBuilder
		ExpectedError  string
		ValidateRes    func(t *testing.T, created *minderv1.Profile, createdID uuid.UUID)
	}{
		{
			Name: "successfully creates profile with rules and publishes reconciler event",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withDisplayName("My Security Profile"), withRules(makeValidRule()))
				p.Name = "my-security-profile"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), db.ListProfilesByProjectIDAndLabelParams{
					ProjectID: projID,
				}).Return([]db.ListProfilesByProjectIDAndLabelRow{}, nil)

				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, arg db.CreateProfileParams) (db.Profile, error) {
						assert.Equal(t, projID, arg.ProjectID)
						assert.Equal(t, "my-security-profile", arg.Name)
						assert.Equal(t, "My Security Profile", arg.DisplayName)
						return db.Profile{
							ID:          ruleTypeID,
							ProjectID:   projID,
							Name:        arg.Name,
							DisplayName: arg.DisplayName,
							Remediate:   db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
							Alert:       db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
							CreatedAt:   time.Now(),
							UpdatedAt:   time.Now(),
						}, nil
					},
				)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().CreateProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, created *minderv1.Profile, _ uuid.UUID) {
				t.Helper()
				require.NotNil(t, created)
				assert.Equal(t, ruleTypeID.String(), created.GetId())
				assert.Equal(t, projID.String(), created.GetContext().GetProject())
			},
		},
		{
			Name:           "fails when profile name validation fails for namespace rules",
			SubscriptionID: uuid.New(),
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Name = "invalid-unnamespaced-name"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
			},
			ExpectedError: "name failed namespace validation",
		},
		{
			Name: "fails when profile validation fails due to missing rule type",
			ProfileArg: func() *minderv1.Profile {
				return makeProfile(withBasicProfileData, withRules(makeValidRule()))
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetParentProjects(gomock.Any(), projID).Return([]uuid.UUID{}, nil).AnyTimes()
				store.EXPECT().GetRuleTypeByName(gomock.Any(), gomock.Any()).Return(db.RuleType{}, sql.ErrNoRows).AnyTimes()
			},
			ExpectedError: "cannot find rule type",
		},
		{
			Name: "fails when ListProfilesByProjectIDAndLabel returns error",
			ProfileArg: func() *minderv1.Profile {
				return makeProfile(withBasicProfileData, withRules(makeValidRule()))
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, errors.New("db disk failure"))
			},
			ExpectedError: "failed to get profiles",
		},
		{
			Name: "returns user visible AlreadyExists error on unique constraint violation",
			ProfileArg: func() *minderv1.Profile {
				return makeProfile(withBasicProfileData, withRules(makeValidRule()))
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{}, &pq.Error{
					Code: pqerror.UniqueViolation,
				})
			},
			ExpectedError: "profile already exists",
		},
		{
			Name: "fails when CreateProfile database call fails with generic error",
			ProfileArg: func() *minderv1.Profile {
				return makeProfile(withBasicProfileData, withRules(makeValidRule()))
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{}, errors.New("connection lost"))
			},
			ExpectedError: "error creating profile",
		},
		{
			Name: "fails when creating profile rules encounters upsert error",
			ProfileArg: func() *minderv1.Profile {
				return makeProfile(withBasicProfileData, withRules(makeValidRule()))
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          ruleTypeID,
					ProjectID:   projID,
					Name:        "test-profile",
					DisplayName: "test-profile",
				}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.Nil, errors.New("foreign key violation"))
			},
			ExpectedError: "unable to insert new rule instance",
		},
		{
			Name: "successfully creates profile with selectors",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Name = "selector-profile"
				p.Selection = []*minderv1.Profile_Selector{
					{
						Entity:      "repository",
						Selector:    "repository.name == 'minder'",
						Description: "Target minder repo",
					},
				}
				return p
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				createdID := uuid.New()
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          createdID,
					ProjectID:   projID,
					Name:        "selector-profile",
					DisplayName: "selector-profile",
				}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().CreateProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				sel.EXPECT().CheckSelector(gomock.Any()).Return(nil)
				store.EXPECT().CreateSelector(gomock.Any(), gomock.Any()).Return(db.ProfileSelector{
					ID:        uuid.New(),
					ProfileID: createdID,
				}, nil)
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, created *minderv1.Profile, _ uuid.UUID) {
				t.Helper()
				require.NotNil(t, created)
			},
		},
		{
			Name: "fails when selector validation fails",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Name = "bad-selector-profile"
				p.Selection = []*minderv1.Profile_Selector{
					{
						Entity:   "repository",
						Selector: "invalid == cel",
					},
				}
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          uuid.New(),
					ProjectID:   projID,
					Name:        "bad-selector-profile",
					DisplayName: "bad-selector-profile",
				}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().CreateProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				sel.EXPECT().CheckSelector(gomock.Any()).Return(errors.New("syntax error in CEL expression"))
			},
			ExpectedError: "syntax error in CEL expression",
		},
		{
			Name: "fails when selector creation DB call fails",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Name = "db-fail-selector-profile"
				p.Selection = []*minderv1.Profile_Selector{
					{
						Entity:      "repository",
						Selector:    "repository.is_fork == false",
						Description: "Only upstream repos",
					},
				}
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          uuid.New(),
					ProjectID:   projID,
					Name:        "db-fail-selector-profile",
					DisplayName: "db-fail-selector-profile",
				}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().CreateProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				sel.EXPECT().CheckSelector(gomock.Any()).Return(nil)
				store.EXPECT().CreateSelector(gomock.Any(), gomock.Any()).Return(db.ProfileSelector{}, errors.New("foreign key violation"))
			},
			ExpectedError: "error creating selector",
		},
		{
			Name: "successfully creates profile with unspecified entity selector",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Name = "unspecified-entity-sel-profile"
				p.Selection = []*minderv1.Profile_Selector{
					{
						Entity:      "",
						Selector:    "true",
						Description: "matches all",
					},
				}
				return p
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().ListProfilesByProjectIDAndLabel(gomock.Any(), gomock.Any()).Return(nil, nil)
				createdID := uuid.New()
				store.EXPECT().CreateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          createdID,
					ProjectID:   projID,
					Name:        "unspecified-entity-sel-profile",
					DisplayName: "unspecified-entity-sel-profile",
				}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().CreateProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				sel.EXPECT().CheckSelector(gomock.Any()).Return(nil)
				store.EXPECT().CreateSelector(gomock.Any(), db.CreateSelectorParams{
					ProfileID: createdID,
					Entity:    db.NullEntities{Valid: false},
					Selector:  "true",
					Comment:   "matches all",
				}).Return(db.ProfileSelector{
					ID:        uuid.New(),
					ProfileID: createdID,
					Selector:  "true",
				}, nil)
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, created *minderv1.Profile, _ uuid.UUID) {
				t.Helper()
				require.NotNil(t, created)
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			store, pub, sel, svc := newTestEnv(t)
			if sc.DBSetup != nil {
				sc.DBSetup(store, pub, sel)
			}

			inProfile := sc.ProfileArg()
			created, err := svc.CreateProfile(context.Background(), projID, sc.SubscriptionID, inProfile, store)
			if sc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), sc.ExpectedError)
			} else {
				require.NoError(t, err)
				if sc.ValidateRes != nil {
					sc.ValidateRes(t, created, ruleTypeID)
				}
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	t.Parallel()

	rawRuleDef, err := loadRawRuleTypeDef()
	require.NoError(t, err)

	projID := uuid.New()
	ruleTypeID := uuid.New()
	ruleTypeParam := dbMockWithRuleTypeID(rawRuleDef, ruleTypeID)

	existingProfID := uuid.New()

	scenarios := []struct {
		Name           string
		SubscriptionID uuid.UUID
		ProfileArg     func() *minderv1.Profile
		DBSetup        dbMockBuilder
		ExpectedError  string
		ValidateRes    func(t *testing.T, updated *minderv1.Profile)
	}{
		{
			Name: "successfully updates profile and cleans up non-updated rules",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withDisplayName("Updated Display Name"), withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				p.Name = "existing-profile"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      projID,
					Name:           "existing-profile",
					DisplayName:    "Existing Profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          existingProfID,
					ProjectID:   projID,
					Name:        "existing-profile",
					DisplayName: "Updated Display Name",
					Remediate:   db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
					Alert:       db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
				}, nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().UpsertProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteSelectorsByProfileID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, updated *minderv1.Profile) {
				t.Helper()
				require.NotNil(t, updated)
				assert.Equal(t, existingProfID.String(), updated.GetId())
			},
		},
		{
			Name: "returns NotFound when profile to update does not exist",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{}, sql.ErrNoRows)
			},
			ExpectedError: "profile not found",
		},
		{
			Name: "rejects update when profile name is changed",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				p.Name = "changed-name"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      projID,
					Name:           "original-name",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)
			},
			ExpectedError: "cannot change profile name",
		},
		{
			Name: "rejects update when project does not match",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				p.Name = "profile-x"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      uuid.New(),
					Name:           "profile-x",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)
			},
			ExpectedError: "cannot change profile project",
		},
		{
			Name: "cleans up entities when entity rules are cleared to empty",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData)
				p.Id = proto.String(existingProfID.String())
				p.Name = "cleared-rules-profile"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      projID,
					Name:           "cleared-rules-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)
				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:        existingProfID,
					ProjectID: projID,
					Name:      "cleared-rules-profile",
				}, nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteSelectorsByProfileID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, updated *minderv1.Profile) {
				t.Helper()
				require.NotNil(t, updated)
			},
		},
		{
			Name: "fails when cleaning up non updated rules returns database error",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				p.Name = "cleanup-error-profile"
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      projID,
					Name:           "cleanup-error-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)
				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:        existingProfID,
					ProjectID: projID,
					Name:      "cleanup-error-profile",
				}, nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().UpsertProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil).AnyTimes()
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil).AnyTimes()
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(errors.New("lock timeout")).AnyTimes()
			},
			ExpectedError: "error while cleaning up rule instances",
		},
		{
			Name: "fails when updateSelectors fails in UpdateProfile",
			ProfileArg: func() *minderv1.Profile {
				p := makeProfile(withBasicProfileData, withRules(makeValidRule()))
				p.Id = proto.String(existingProfID.String())
				p.Name = "bad-sel-update-profile"
				p.Selection = []*minderv1.Profile_Selector{
					{
						Entity:   "repository",
						Selector: "syntax error",
					},
				}
				return p
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, sel *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        existingProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             existingProfID,
					ProjectID:      projID,
					Name:           "bad-sel-update-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)
				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:        existingProfID,
					ProjectID: projID,
					Name:      "bad-sel-update-profile",
				}, nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().UpsertProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil).AnyTimes()
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil).AnyTimes()
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				sel.EXPECT().CheckSelector(gomock.Any()).Return(errors.New("invalid CEL syntax in selector"))
			},
			ExpectedError: "invalid CEL syntax in selector",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			store, pub, sel, svc := newTestEnv(t)
			if sc.DBSetup != nil {
				sc.DBSetup(store, pub, sel)
			}

			inProfile := sc.ProfileArg()
			updated, err := svc.UpdateProfile(context.Background(), projID, sc.SubscriptionID, inProfile, store)
			if sc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), sc.ExpectedError)
			} else {
				require.NoError(t, err)
				if sc.ValidateRes != nil {
					sc.ValidateRes(t, updated)
				}
			}
		})
	}
}

func TestPatchProfile(t *testing.T) {
	t.Parallel()

	rawRuleDef, err := loadRawRuleTypeDef()
	require.NoError(t, err)

	projID := uuid.New()
	ruleTypeID := uuid.New()
	ruleTypeParam := dbMockWithRuleTypeID(rawRuleDef, ruleTypeID)
	baseProfID := uuid.New()

	scenarios := []struct {
		Name          string
		Target        string
		PatchArg      func() (*minderv1.Profile, *fieldmaskpb.FieldMask)
		DBSetup       dbMockBuilder
		ExpectedError string
		ValidateRes   func(t *testing.T, patched *minderv1.Profile)
	}{
		{
			Name:   "successfully patches profile by UUID with field mask",
			Target: baseProfID.String(),
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				p := &minderv1.Profile{
					DisplayName: "Patched Display Name",
					Remediate:   proto.String("on"),
				}
				mask, err := fieldmaskpb.New(p, "display_name", "remediate")
				require.NoError(t, err)
				return p, mask
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return([]db.GetProfileByProjectAndIDRow{
					{
						Profile: db.Profile{
							ID:          baseProfID,
							ProjectID:   projID,
							Name:        "base-profile",
							DisplayName: "Base Profile",
							Remediate:   db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
							Alert:       db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
						},
						ProfilesWithEntityProfile: db.ProfilesWithEntityProfile{
							Entity: db.NullEntities{Entities: db.EntitiesRepository, Valid: true},
							ContextualRules: pqtype.NullRawMessage{
								RawMessage: []byte(fmt.Sprintf(`[{"type":%q,"name":%q,"def":{"allow_force_pushes":false},"params":{"branch":"main"}}]`,
									ruleTypeName, ruleTypeName)),
								Valid: true,
							},
						},
					},
				}, nil)

				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        baseProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             baseProfID,
					ProjectID:      projID,
					Name:           "base-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          baseProfID,
					ProjectID:   projID,
					Name:        "base-profile",
					DisplayName: "Patched Display Name",
					Remediate:   db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
					Alert:       db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
				}, nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().UpsertProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteSelectorsByProfileID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, patched *minderv1.Profile) {
				t.Helper()
				require.NotNil(t, patched)
				assert.Equal(t, "Patched Display Name", patched.GetDisplayName())
			},
		},
		{
			Name:   "successfully patches profile looked up by name",
			Target: "named-profile",
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				p := &minderv1.Profile{DisplayName: "New Name"}
				mask, err := fieldmaskpb.New(p, "display_name")
				require.NoError(t, err)
				return p, mask
			},
			DBSetup: func(store *mockdb.MockStore, pub *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByNameAndLock(gomock.Any(), db.GetProfileByNameAndLockParams{
					ProjectID: projID,
					Name:      "named-profile",
				}).Return(db.Profile{
					ID:        baseProfID,
					ProjectID: projID,
					Name:      "named-profile",
				}, nil)
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return([]db.GetProfileByProjectAndIDRow{
					{
						Profile: db.Profile{
							ID:          baseProfID,
							ProjectID:   projID,
							Name:        "named-profile",
							DisplayName: "Named Profile",
							Remediate:   db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
							Alert:       db.NullActionType{ActionType: db.ActionTypeOff, Valid: true},
						},
						ProfilesWithEntityProfile: db.ProfilesWithEntityProfile{
							Entity: db.NullEntities{Entities: db.EntitiesRepository, Valid: true},
							ContextualRules: pqtype.NullRawMessage{
								RawMessage: []byte(fmt.Sprintf(`[{"type":%q,"name":%q,"def":{"allow_force_pushes":false},"params":{"branch":"main"}}]`,
									ruleTypeName, ruleTypeName)),
								Valid: true,
							},
						},
					},
				}, nil)

				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        baseProfID,
					ProjectID: projID,
				}).Return(db.Profile{
					ID:             baseProfID,
					ProjectID:      projID,
					Name:           "named-profile",
					SubscriptionID: uuid.NullUUID{Valid: false},
				}, nil)

				store.EXPECT().UpdateProfile(gomock.Any(), gomock.Any()).Return(db.Profile{
					ID:          baseProfID,
					ProjectID:   projID,
					Name:        "named-profile",
					DisplayName: "New Name",
				}, nil)
				store.EXPECT().UpsertProfileForEntity(gomock.Any(), gomock.Any()).Return(db.EntityProfile{}, nil)
				store.EXPECT().UpsertRuleInstance(gomock.Any(), gomock.Any()).Return(uuid.New(), nil)
				store.EXPECT().DeleteProfileForEntity(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteNonUpdatedRules(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				store.EXPECT().DeleteSelectorsByProfileID(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				pub.EXPECT().Publish(constants.TopicQueueReconcileProfileInit, gomock.Any()).Return(nil)
			},
			ValidateRes: func(t *testing.T, patched *minderv1.Profile) {
				t.Helper()
				require.NotNil(t, patched)
			},
		},
		{
			Name:   "returns NotFound when patch lookup by name returns ErrNoRows",
			Target: "nonexistent-patch-target",
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				return &minderv1.Profile{}, nil
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByNameAndLock(gomock.Any(), db.GetProfileByNameAndLockParams{
					ProjectID: projID,
					Name:      "nonexistent-patch-target",
				}).Return(db.Profile{}, sql.ErrNoRows)
			},
			ExpectedError: `profile "nonexistent-patch-target" not found`,
		},
		{
			Name:   "returns error when GetProfileByProjectAndID fails in DB",
			Target: baseProfID.String(),
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				return &minderv1.Profile{}, nil
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return(nil, errors.New("read replica down"))
			},
			ExpectedError: "failed to get profile",
		},
		{
			Name:   "returns error when GetProfileByProjectAndID returns empty rows",
			Target: baseProfID.String(),
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				return &minderv1.Profile{}, nil
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return([]db.GetProfileByProjectAndIDRow{}, nil)
			},
			ExpectedError: "profile not found",
		},
		{
			Name:   "returns error when UpdateProfile fails during patch",
			Target: baseProfID.String(),
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				p := &minderv1.Profile{DisplayName: "Updated"}
				mask, err := fieldmaskpb.New(p, "display_name")
				require.NoError(t, err)
				return p, mask
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				ruleTypeParam(store)
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return([]db.GetProfileByProjectAndIDRow{
					{
						Profile: db.Profile{
							ID:        baseProfID,
							ProjectID: projID,
							Name:      "test-prof",
						},
					},
				}, nil)
				store.EXPECT().GetProfileByIDAndLock(gomock.Any(), db.GetProfileByIDAndLockParams{
					ID:        baseProfID,
					ProjectID: projID,
				}).Return(db.Profile{}, errors.New("cannot obtain lock on profile"))
			},
			ExpectedError: "cannot obtain lock on profile",
		},
		{
			Name:   "returns error when target profile ID is invalid string and name lookup fails",
			Target: "some-bad-name",
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				return &minderv1.Profile{}, nil
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByNameAndLock(gomock.Any(), db.GetProfileByNameAndLockParams{
					ProjectID: projID,
					Name:      "some-bad-name",
				}).Return(db.Profile{}, errors.New("connection reset by peer"))
			},
			ExpectedError: "error fetching profile to be patched",
		},
		{
			Name:   "returns error when database returns multiple profiles for single ID",
			Target: baseProfID.String(),
			PatchArg: func() (*minderv1.Profile, *fieldmaskpb.FieldMask) {
				return &minderv1.Profile{}, nil
			},
			DBSetup: func(store *mockdb.MockStore, _ *mockevents.MockPublisher, _ *mocksel.MockSelectionChecker) {
				store.EXPECT().GetProfileByProjectAndID(gomock.Any(), db.GetProfileByProjectAndIDParams{
					ProjectID: projID,
					ID:        baseProfID,
				}).Return([]db.GetProfileByProjectAndIDRow{
					{
						Profile: db.Profile{
							ID:        baseProfID,
							ProjectID: projID,
							Name:      "prof-1",
						},
					},
					{
						Profile: db.Profile{
							ID:        uuid.New(),
							ProjectID: projID,
							Name:      "prof-2",
						},
					},
				}, nil)
			},
			ExpectedError: "expected only one profile",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			t.Parallel()
			store, pub, sel, svc := newTestEnv(t)
			if sc.DBSetup != nil {
				sc.DBSetup(store, pub, sel)
			}

			patchPb, mask := sc.PatchArg()
			patched, err := svc.PatchProfile(context.Background(), projID, sc.Target, patchPb, mask, store)
			if sc.ExpectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), sc.ExpectedError)
			} else {
				require.NoError(t, err)
				if sc.ValidateRes != nil {
					sc.ValidateRes(t, patched)
				}
			}
		})
	}
}

// Helpers for test setup
func dbMockWithRuleTypeID(ruleTypeDef []byte, rtID uuid.UUID) func(store *mockdb.MockStore) {
	return func(store *mockdb.MockStore) {
		store.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).Return([]uuid.UUID{}, nil).AnyTimes()
		store.EXPECT().GetRuleTypeByName(gomock.Any(), gomock.Any()).Return(db.RuleType{
			ID:          rtID,
			Name:        ruleTypeName,
			DisplayName: ruleTypeDisplayName,
			Definition:  ruleTypeDef,
		}, nil).AnyTimes()
	}
}

func makeValidRule() *minderv1.Profile_Rule {
	return makeRule(withRuleDefs, withRuleParams)
}

func withDisplayName(name string) func(profile *minderv1.Profile) {
	return func(profile *minderv1.Profile) {
		profile.DisplayName = name
	}
}
