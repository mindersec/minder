// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type Querier interface {
	CountProfilesByEntityType(ctx context.Context) ([]CountProfilesByEntityTypeRow, error)
	CountProfilesByName(ctx context.Context, name string) (int64, error)
	CountRepositories(ctx context.Context) (int64, error)
	CountUsers(ctx context.Context) (int64, error)
	CreateAccessToken(ctx context.Context, arg CreateAccessTokenParams) (ProviderAccessToken, error)
	CreateArtifact(ctx context.Context, arg CreateArtifactParams) (Artifact, error)
	CreateOrganization(ctx context.Context, arg CreateOrganizationParams) (Project, error)
	CreateProfile(ctx context.Context, arg CreateProfileParams) (Profile, error)
	CreateProfileForEntity(ctx context.Context, arg CreateProfileForEntityParams) (EntityProfile, error)
	CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error)
	CreateProjectWithID(ctx context.Context, arg CreateProjectWithIDParams) (Project, error)
	CreateProvider(ctx context.Context, arg CreateProviderParams) (Provider, error)
	CreatePullRequest(ctx context.Context, arg CreatePullRequestParams) (PullRequest, error)
	CreateRepository(ctx context.Context, arg CreateRepositoryParams) (Repository, error)
	CreateRuleType(ctx context.Context, arg CreateRuleTypeParams) (RuleType, error)
	CreateSessionState(ctx context.Context, arg CreateSessionStateParams) (SessionStore, error)
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	DeleteAccessToken(ctx context.Context, arg DeleteAccessTokenParams) error
	DeleteArtifact(ctx context.Context, id uuid.UUID) error
	DeleteExpiredSessionStates(ctx context.Context) error
	DeleteOrganization(ctx context.Context, id uuid.UUID) error
	DeleteProfile(ctx context.Context, id uuid.UUID) error
	DeleteProfileForEntity(ctx context.Context, arg DeleteProfileForEntityParams) error
	DeleteProject(ctx context.Context, id uuid.UUID) ([]DeleteProjectRow, error)
	DeleteProvider(ctx context.Context, arg DeleteProviderParams) error
	DeletePullRequest(ctx context.Context, arg DeletePullRequestParams) error
	DeleteRepository(ctx context.Context, id uuid.UUID) error
	DeleteRuleInstantiation(ctx context.Context, arg DeleteRuleInstantiationParams) error
	// DeleteRuleStatusesForProfileAndRuleType deletes a rule evaluation
	// but locks the table before doing so.
	DeleteRuleStatusesForProfileAndRuleType(ctx context.Context, arg DeleteRuleStatusesForProfileAndRuleTypeParams) error
	DeleteRuleType(ctx context.Context, id uuid.UUID) error
	DeleteSessionState(ctx context.Context, id int32) error
	DeleteSessionStateByProjectID(ctx context.Context, arg DeleteSessionStateByProjectIDParams) error
	DeleteUser(ctx context.Context, id int32) error
	EnqueueFlush(ctx context.Context, arg EnqueueFlushParams) (FlushCache, error)
	FlushCache(ctx context.Context, arg FlushCacheParams) (FlushCache, error)
	GetAccessTokenByProjectID(ctx context.Context, arg GetAccessTokenByProjectIDParams) (ProviderAccessToken, error)
	GetAccessTokenByProvider(ctx context.Context, provider string) ([]ProviderAccessToken, error)
	GetAccessTokenSinceDate(ctx context.Context, arg GetAccessTokenSinceDateParams) (ProviderAccessToken, error)
	GetArtifactByID(ctx context.Context, id uuid.UUID) (GetArtifactByIDRow, error)
	GetArtifactByName(ctx context.Context, arg GetArtifactByNameParams) (GetArtifactByNameRow, error)
	GetChildrenProjects(ctx context.Context, id uuid.UUID) ([]GetChildrenProjectsRow, error)
	GetEntityProfileByProjectAndName(ctx context.Context, arg GetEntityProfileByProjectAndNameParams) ([]GetEntityProfileByProjectAndNameRow, error)
	// GetFeatureInProject verifies if a feature is available for a specific project.
	// It returns the settings for the feature if it is available.
	GetFeatureInProject(ctx context.Context, arg GetFeatureInProjectParams) (json.RawMessage, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (Project, error)
	GetOrganizationByName(ctx context.Context, name string) (Project, error)
	GetOrganizationForUpdate(ctx context.Context, name string) (Project, error)
	GetParentProjects(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	GetParentProjectsUntil(ctx context.Context, arg GetParentProjectsUntilParams) ([]uuid.UUID, error)
	GetProfileByID(ctx context.Context, id uuid.UUID) (Profile, error)
	GetProfileByIDAndLock(ctx context.Context, id uuid.UUID) (Profile, error)
	GetProfileByNameAndLock(ctx context.Context, arg GetProfileByNameAndLockParams) (Profile, error)
	GetProfileByProjectAndID(ctx context.Context, arg GetProfileByProjectAndIDParams) ([]GetProfileByProjectAndIDRow, error)
	GetProfileForEntity(ctx context.Context, arg GetProfileForEntityParams) (EntityProfile, error)
	GetProfileStatusByIdAndProject(ctx context.Context, arg GetProfileStatusByIdAndProjectParams) (GetProfileStatusByIdAndProjectRow, error)
	GetProfileStatusByNameAndProject(ctx context.Context, arg GetProfileStatusByNameAndProjectParams) (GetProfileStatusByNameAndProjectRow, error)
	GetProfileStatusByProject(ctx context.Context, projectID uuid.UUID) ([]GetProfileStatusByProjectRow, error)
	GetProjectByID(ctx context.Context, id uuid.UUID) (Project, error)
	GetProjectByName(ctx context.Context, name string) (Project, error)
	GetProjectIDPortBySessionState(ctx context.Context, sessionState string) (GetProjectIDPortBySessionStateRow, error)
	GetProviderByID(ctx context.Context, arg GetProviderByIDParams) (Provider, error)
	GetProviderByName(ctx context.Context, arg GetProviderByNameParams) (Provider, error)
	GetPullRequest(ctx context.Context, arg GetPullRequestParams) (PullRequest, error)
	GetPullRequestByID(ctx context.Context, id uuid.UUID) (PullRequest, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (Repository, error)
	GetRepositoryByIDAndProject(ctx context.Context, arg GetRepositoryByIDAndProjectParams) (Repository, error)
	GetRepositoryByRepoID(ctx context.Context, repoID int64) (Repository, error)
	GetRepositoryByRepoName(ctx context.Context, arg GetRepositoryByRepoNameParams) (Repository, error)
	GetRootProjects(ctx context.Context) ([]Project, error)
	GetRuleTypeByID(ctx context.Context, id uuid.UUID) (RuleType, error)
	GetRuleTypeByName(ctx context.Context, arg GetRuleTypeByNameParams) (RuleType, error)
	GetSessionState(ctx context.Context, id int32) (SessionStore, error)
	GetSessionStateByProjectID(ctx context.Context, projectID uuid.UUID) (SessionStore, error)
	GetUserByID(ctx context.Context, id int32) (User, error)
	GetUserBySubject(ctx context.Context, identitySubject string) (User, error)
	GlobalListProviders(ctx context.Context) ([]Provider, error)
	ListArtifactsByRepoID(ctx context.Context, repositoryID uuid.UUID) ([]Artifact, error)
	ListFlushCache(ctx context.Context) ([]FlushCache, error)
	ListOrganizations(ctx context.Context, arg ListOrganizationsParams) ([]Project, error)
	ListProfilesByProjectID(ctx context.Context, projectID uuid.UUID) ([]ListProfilesByProjectIDRow, error)
	// get profile information that instantiate a rule. This is done by joining the profiles with entity_profiles, then correlating those
	// with entity_profile_rules. The rule_type_id is used to filter the results. Note that we only really care about the overal profile,
	// so we only return the profile information. We also should group the profiles so that we don't get duplicates.
	ListProfilesInstantiatingRuleType(ctx context.Context, ruleTypeID uuid.UUID) ([]ListProfilesInstantiatingRuleTypeRow, error)
	// ListProvidersByProjectID allows us to lits all providers for a given project.
	ListProvidersByProjectID(ctx context.Context, projectID uuid.UUID) ([]Provider, error)
	// ListProvidersByProjectIDPaginated allows us to lits all providers for a given project
	// with pagination taken into account. In this case, the cursor is the creation date.
	ListProvidersByProjectIDPaginated(ctx context.Context, arg ListProvidersByProjectIDPaginatedParams) ([]Provider, error)
	ListRegisteredRepositoriesByProjectIDAndProvider(ctx context.Context, arg ListRegisteredRepositoriesByProjectIDAndProviderParams) ([]Repository, error)
	ListRepositoriesByProjectID(ctx context.Context, arg ListRepositoriesByProjectIDParams) ([]Repository, error)
	ListRuleEvaluationsByProfileId(ctx context.Context, arg ListRuleEvaluationsByProfileIdParams) ([]ListRuleEvaluationsByProfileIdRow, error)
	ListRuleTypesByProviderAndProject(ctx context.Context, arg ListRuleTypesByProviderAndProjectParams) ([]RuleType, error)
	ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error)
	ListUsersByOrganization(ctx context.Context, arg ListUsersByOrganizationParams) ([]User, error)
	// LockIfThresholdNotExceeded is used to lock an entity for execution. It will
	// attempt to insert or update the entity_execution_lock table only if the
	// last_lock_time is older than the threshold. If the lock is successful, it
	// will return the lock record. If the lock is unsuccessful, it will return
	// NULL.
	LockIfThresholdNotExceeded(ctx context.Context, arg LockIfThresholdNotExceededParams) (EntityExecutionLock, error)
	// ReleaseLock is used to release a lock on an entity. It will delete the
	// entity_execution_lock record if the lock is held by the given locked_by
	// value.
	ReleaseLock(ctx context.Context, arg ReleaseLockParams) error
	UpdateAccessToken(ctx context.Context, arg UpdateAccessTokenParams) (ProviderAccessToken, error)
	UpdateLease(ctx context.Context, arg UpdateLeaseParams) error
	UpdateOrganization(ctx context.Context, arg UpdateOrganizationParams) (Project, error)
	UpdateProfile(ctx context.Context, arg UpdateProfileParams) (Profile, error)
	// set clone_url if the value is not an empty string
	UpdateRepository(ctx context.Context, arg UpdateRepositoryParams) (Repository, error)
	UpdateRuleType(ctx context.Context, arg UpdateRuleTypeParams) error
	UpsertArtifact(ctx context.Context, arg UpsertArtifactParams) (Artifact, error)
	UpsertProfileForEntity(ctx context.Context, arg UpsertProfileForEntityParams) (EntityProfile, error)
	UpsertPullRequest(ctx context.Context, arg UpsertPullRequestParams) (PullRequest, error)
	UpsertRuleDetailsAlert(ctx context.Context, arg UpsertRuleDetailsAlertParams) (uuid.UUID, error)
	UpsertRuleDetailsEval(ctx context.Context, arg UpsertRuleDetailsEvalParams) (uuid.UUID, error)
	UpsertRuleDetailsRemediate(ctx context.Context, arg UpsertRuleDetailsRemediateParams) (uuid.UUID, error)
	UpsertRuleEvaluations(ctx context.Context, arg UpsertRuleEvaluationsParams) (uuid.UUID, error)
	UpsertRuleInstantiation(ctx context.Context, arg UpsertRuleInstantiationParams) (EntityProfileRule, error)
}

var _ Querier = (*Queries)(nil)
