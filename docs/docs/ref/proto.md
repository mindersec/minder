---
sidebar_position: 60
toc_max_heading_level: 4
---
# Protocol Documentation
<a name="top"></a>



<a name="minder_v1_minder-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## minder/v1/minder.proto

### Services

<a name="minder-v1-ArtifactService"></a>

#### ArtifactService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListArtifacts | [ListArtifactsRequest](#minder-v1-ListArtifactsRequest) | [ListArtifactsResponse](#minder-v1-ListArtifactsResponse) |  |
| GetArtifactById | [GetArtifactByIdRequest](#minder-v1-GetArtifactByIdRequest) | [GetArtifactByIdResponse](#minder-v1-GetArtifactByIdResponse) |  |
| GetArtifactByName | [GetArtifactByNameRequest](#minder-v1-GetArtifactByNameRequest) | [GetArtifactByNameResponse](#minder-v1-GetArtifactByNameResponse) |  |


<a name="minder-v1-EvalResultsService"></a>

#### EvalResultsService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListEvaluationResults | [ListEvaluationResultsRequest](#minder-v1-ListEvaluationResultsRequest) | [ListEvaluationResultsResponse](#minder-v1-ListEvaluationResultsResponse) |  |


<a name="minder-v1-HealthService"></a>

#### HealthService
Simple Health Check Service
replies with OK

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CheckHealth | [CheckHealthRequest](#minder-v1-CheckHealthRequest) | [CheckHealthResponse](#minder-v1-CheckHealthResponse) |  |


<a name="minder-v1-OAuthService"></a>

#### OAuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetAuthorizationURL | [GetAuthorizationURLRequest](#minder-v1-GetAuthorizationURLRequest) | [GetAuthorizationURLResponse](#minder-v1-GetAuthorizationURLResponse) |  |
| StoreProviderToken | [StoreProviderTokenRequest](#minder-v1-StoreProviderTokenRequest) | [StoreProviderTokenResponse](#minder-v1-StoreProviderTokenResponse) |  |
| VerifyProviderTokenFrom | [VerifyProviderTokenFromRequest](#minder-v1-VerifyProviderTokenFromRequest) | [VerifyProviderTokenFromResponse](#minder-v1-VerifyProviderTokenFromResponse) | VerifyProviderTokenFrom verifies that a token has been created for a provider since given timestamp |


<a name="minder-v1-PermissionsService"></a>

#### PermissionsService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#minder-v1-ListRolesRequest) | [ListRolesResponse](#minder-v1-ListRolesResponse) |  |
| ListRoleAssignments | [ListRoleAssignmentsRequest](#minder-v1-ListRoleAssignmentsRequest) | [ListRoleAssignmentsResponse](#minder-v1-ListRoleAssignmentsResponse) |  |
| AssignRole | [AssignRoleRequest](#minder-v1-AssignRoleRequest) | [AssignRoleResponse](#minder-v1-AssignRoleResponse) |  |
| RemoveRole | [RemoveRoleRequest](#minder-v1-RemoveRoleRequest) | [RemoveRoleResponse](#minder-v1-RemoveRoleResponse) |  |


<a name="minder-v1-ProfileService"></a>

#### ProfileService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateProfile | [CreateProfileRequest](#minder-v1-CreateProfileRequest) | [CreateProfileResponse](#minder-v1-CreateProfileResponse) |  |
| UpdateProfile | [UpdateProfileRequest](#minder-v1-UpdateProfileRequest) | [UpdateProfileResponse](#minder-v1-UpdateProfileResponse) |  |
| PatchProfile | [PatchProfileRequest](#minder-v1-PatchProfileRequest) | [PatchProfileResponse](#minder-v1-PatchProfileResponse) |  |
| DeleteProfile | [DeleteProfileRequest](#minder-v1-DeleteProfileRequest) | [DeleteProfileResponse](#minder-v1-DeleteProfileResponse) |  |
| ListProfiles | [ListProfilesRequest](#minder-v1-ListProfilesRequest) | [ListProfilesResponse](#minder-v1-ListProfilesResponse) |  |
| GetProfileById | [GetProfileByIdRequest](#minder-v1-GetProfileByIdRequest) | [GetProfileByIdResponse](#minder-v1-GetProfileByIdResponse) |  |
| GetProfileStatusByName | [GetProfileStatusByNameRequest](#minder-v1-GetProfileStatusByNameRequest) | [GetProfileStatusByNameResponse](#minder-v1-GetProfileStatusByNameResponse) |  |
| GetProfileStatusByProject | [GetProfileStatusByProjectRequest](#minder-v1-GetProfileStatusByProjectRequest) | [GetProfileStatusByProjectResponse](#minder-v1-GetProfileStatusByProjectResponse) |  |
| ListRuleTypes | [ListRuleTypesRequest](#minder-v1-ListRuleTypesRequest) | [ListRuleTypesResponse](#minder-v1-ListRuleTypesResponse) |  |
| GetRuleTypeByName | [GetRuleTypeByNameRequest](#minder-v1-GetRuleTypeByNameRequest) | [GetRuleTypeByNameResponse](#minder-v1-GetRuleTypeByNameResponse) |  |
| GetRuleTypeById | [GetRuleTypeByIdRequest](#minder-v1-GetRuleTypeByIdRequest) | [GetRuleTypeByIdResponse](#minder-v1-GetRuleTypeByIdResponse) |  |
| CreateRuleType | [CreateRuleTypeRequest](#minder-v1-CreateRuleTypeRequest) | [CreateRuleTypeResponse](#minder-v1-CreateRuleTypeResponse) |  |
| UpdateRuleType | [UpdateRuleTypeRequest](#minder-v1-UpdateRuleTypeRequest) | [UpdateRuleTypeResponse](#minder-v1-UpdateRuleTypeResponse) |  |
| DeleteRuleType | [DeleteRuleTypeRequest](#minder-v1-DeleteRuleTypeRequest) | [DeleteRuleTypeResponse](#minder-v1-DeleteRuleTypeResponse) |  |


<a name="minder-v1-ProjectsService"></a>

#### ProjectsService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListProjects | [ListProjectsRequest](#minder-v1-ListProjectsRequest) | [ListProjectsResponse](#minder-v1-ListProjectsResponse) |  |
| CreateProject | [CreateProjectRequest](#minder-v1-CreateProjectRequest) | [CreateProjectResponse](#minder-v1-CreateProjectResponse) |  |
| DeleteProject | [DeleteProjectRequest](#minder-v1-DeleteProjectRequest) | [DeleteProjectResponse](#minder-v1-DeleteProjectResponse) |  |
| UpdateProject | [UpdateProjectRequest](#minder-v1-UpdateProjectRequest) | [UpdateProjectResponse](#minder-v1-UpdateProjectResponse) |  |
| PatchProject | [PatchProjectRequest](#minder-v1-PatchProjectRequest) | [PatchProjectResponse](#minder-v1-PatchProjectResponse) |  |
| CreateEntityReconciliationTask | [CreateEntityReconciliationTaskRequest](#minder-v1-CreateEntityReconciliationTaskRequest) | [CreateEntityReconciliationTaskResponse](#minder-v1-CreateEntityReconciliationTaskResponse) |  |


<a name="minder-v1-ProvidersService"></a>

#### ProvidersService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProvider | [GetProviderRequest](#minder-v1-GetProviderRequest) | [GetProviderResponse](#minder-v1-GetProviderResponse) |  |
| ListProviders | [ListProvidersRequest](#minder-v1-ListProvidersRequest) | [ListProvidersResponse](#minder-v1-ListProvidersResponse) |  |
| CreateProvider | [CreateProviderRequest](#minder-v1-CreateProviderRequest) | [CreateProviderResponse](#minder-v1-CreateProviderResponse) |  |
| DeleteProvider | [DeleteProviderRequest](#minder-v1-DeleteProviderRequest) | [DeleteProviderResponse](#minder-v1-DeleteProviderResponse) |  |
| GetUnclaimedProviders | [GetUnclaimedProvidersRequest](#minder-v1-GetUnclaimedProvidersRequest) | [GetUnclaimedProvidersResponse](#minder-v1-GetUnclaimedProvidersResponse) | GetUnclaimedProviders returns a list of known provider configurations that this user could claim based on their identity.  This is a read-only operation for use by clients which wish to present a menu of options. |
| ListProviderClasses | [ListProviderClassesRequest](#minder-v1-ListProviderClassesRequest) | [ListProviderClassesResponse](#minder-v1-ListProviderClassesResponse) |  |


<a name="minder-v1-RepositoryService"></a>

#### RepositoryService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| RegisterRepository | [RegisterRepositoryRequest](#minder-v1-RegisterRepositoryRequest) | [RegisterRepositoryResponse](#minder-v1-RegisterRepositoryResponse) |  |
| ListRemoteRepositoriesFromProvider | [ListRemoteRepositoriesFromProviderRequest](#minder-v1-ListRemoteRepositoriesFromProviderRequest) | [ListRemoteRepositoriesFromProviderResponse](#minder-v1-ListRemoteRepositoriesFromProviderResponse) |  |
| ListRepositories | [ListRepositoriesRequest](#minder-v1-ListRepositoriesRequest) | [ListRepositoriesResponse](#minder-v1-ListRepositoriesResponse) |  |
| GetRepositoryById | [GetRepositoryByIdRequest](#minder-v1-GetRepositoryByIdRequest) | [GetRepositoryByIdResponse](#minder-v1-GetRepositoryByIdResponse) |  |
| GetRepositoryByName | [GetRepositoryByNameRequest](#minder-v1-GetRepositoryByNameRequest) | [GetRepositoryByNameResponse](#minder-v1-GetRepositoryByNameResponse) |  |
| DeleteRepositoryById | [DeleteRepositoryByIdRequest](#minder-v1-DeleteRepositoryByIdRequest) | [DeleteRepositoryByIdResponse](#minder-v1-DeleteRepositoryByIdResponse) |  |
| DeleteRepositoryByName | [DeleteRepositoryByNameRequest](#minder-v1-DeleteRepositoryByNameRequest) | [DeleteRepositoryByNameResponse](#minder-v1-DeleteRepositoryByNameResponse) |  |


<a name="minder-v1-UserService"></a>

#### UserService
manage Users CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#minder-v1-CreateUserRequest) | [CreateUserResponse](#minder-v1-CreateUserResponse) |  |
| DeleteUser | [DeleteUserRequest](#minder-v1-DeleteUserRequest) | [DeleteUserResponse](#minder-v1-DeleteUserResponse) |  |
| GetUser | [GetUserRequest](#minder-v1-GetUserRequest) | [GetUserResponse](#minder-v1-GetUserResponse) |  |


### Messages

<a name="minder-v1-Artifact"></a>

#### Artifact



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact_pk | [string](#string) |  |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| type | [string](#string) |  |  |
| visibility | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| versions | [ArtifactVersion](#minder-v1-ArtifactVersion) | repeated |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="minder-v1-ArtifactType"></a>

#### ArtifactType
ArtifactType defines the artifact data evaluation.


<a name="minder-v1-ArtifactVersion"></a>

#### ArtifactVersion



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version_id | [int64](#int64) |  |  |
| tags | [string](#string) | repeated |  |
| sha | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="minder-v1-AssignRoleRequest"></a>

#### AssignRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the role assignment is evaluated. |
| role_assignment | [RoleAssignment](#minder-v1-RoleAssignment) |  | role_assignment is the role assignment to be created. |


<a name="minder-v1-AssignRoleResponse"></a>

#### AssignRoleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignment | [RoleAssignment](#minder-v1-RoleAssignment) |  | role_assignment is the role assignment that was created. |


<a name="minder-v1-AuthorizationParams"></a>

#### AuthorizationParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authorization_url | [string](#string) |  | authorization_url is an external URL to use to authorize the provider. |


<a name="minder-v1-BranchProtection"></a>

#### BranchProtection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [string](#string) |  |  |
| is_protected | [bool](#bool) |  | Add other relevant fields |


<a name="minder-v1-BuiltinType"></a>

#### BuiltinType
BuiltinType defines the builtin data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  |  |


<a name="minder-v1-CheckHealthRequest"></a>

#### CheckHealthRequest



<a name="minder-v1-CheckHealthResponse"></a>

#### CheckHealthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |


<a name="minder-v1-Context"></a>

#### Context
Context defines the context in which a rule is evaluated.
this normally refers to a combination of the provider, organization and project.

Removing the 'optional' keyword from the following two fields below will break
buf compatibility checks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) | optional | name of the provider |
| project | [string](#string) | optional | ID of the project |
| retired_organization | [string](#string) | optional |  |


<a name="minder-v1-CreateEntityReconciliationTaskRequest"></a>

#### CreateEntityReconciliationTaskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | [EntityTypedId](#minder-v1-EntityTypedId) |  | entity is the entity to be reconciled. |
| context | [Context](#minder-v1-Context) |  | context is the context in which the entity reconciliation task is created. |


<a name="minder-v1-CreateEntityReconciliationTaskResponse"></a>

#### CreateEntityReconciliationTaskResponse



<a name="minder-v1-CreateProfileRequest"></a>

#### CreateProfileRequest
Profile service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-CreateProfileResponse"></a>

#### CreateProfileResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-CreateProjectRequest"></a>

#### CreateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the project is created. |
| name | [string](#string) |  | name is the name of the project to create. |


<a name="minder-v1-CreateProjectResponse"></a>

#### CreateProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#minder-v1-Project) |  | project is the project that was created. |


<a name="minder-v1-CreateProviderRequest"></a>

#### CreateProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the provider is created. |
| provider | [Provider](#minder-v1-Provider) |  | provider is the provider to be created. |


<a name="minder-v1-CreateProviderResponse"></a>

#### CreateProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [Provider](#minder-v1-Provider) |  | provider is the provider that was created. |
| authorization | [AuthorizationParams](#minder-v1-AuthorizationParams) |  | authorization provides additional authorization information needed to complete the initialization of the provider. |


<a name="minder-v1-CreateRuleTypeRequest"></a>

#### CreateRuleTypeRequest
CreateRuleTypeRequest is the request to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type to be created. |


<a name="minder-v1-CreateRuleTypeResponse"></a>

#### CreateRuleTypeResponse
CreateRuleTypeResponse is the response to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type that was created. |


<a name="minder-v1-CreateUserRequest"></a>

#### CreateUserRequest
User service


<a name="minder-v1-CreateUserResponse"></a>

#### CreateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [string](#string) |  | **Deprecated.**  |
| organizatio_name | [string](#string) |  | **Deprecated.**  |
| project_id | [string](#string) |  |  |
| project_name | [string](#string) |  |  |
| identity_subject | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-DeleteProfileRequest"></a>

#### DeleteProfileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the profile to delete |


<a name="minder-v1-DeleteProfileResponse"></a>

#### DeleteProfileResponse



<a name="minder-v1-DeleteProjectRequest"></a>

#### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the project is deleted. |


<a name="minder-v1-DeleteProjectResponse"></a>

#### DeleteProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  | project_id is the id of the project that was deleted. |


<a name="minder-v1-DeleteProviderRequest"></a>

#### DeleteProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the provider is deleted. Both project and provider are required in this context. |


<a name="minder-v1-DeleteProviderResponse"></a>

#### DeleteProviderResponse



<a name="minder-v1-DeleteRepositoryByIdRequest"></a>

#### DeleteRepositoryByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-DeleteRepositoryByIdResponse"></a>

#### DeleteRepositoryByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [string](#string) |  |  |


<a name="minder-v1-DeleteRepositoryByNameRequest"></a>

#### DeleteRepositoryByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| name | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-DeleteRepositoryByNameResponse"></a>

#### DeleteRepositoryByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |


<a name="minder-v1-DeleteRuleTypeRequest"></a>

#### DeleteRuleTypeRequest
DeleteRuleTypeRequest is the request to delete a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the rule type to be deleted. |


<a name="minder-v1-DeleteRuleTypeResponse"></a>

#### DeleteRuleTypeResponse
DeleteRuleTypeResponse is the response to delete a rule type.


<a name="minder-v1-DeleteUserRequest"></a>

#### DeleteUserRequest



<a name="minder-v1-DeleteUserResponse"></a>

#### DeleteUserResponse



<a name="minder-v1-Dependency"></a>

#### Dependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystem | [DepEcosystem](#minder-v1-DepEcosystem) |  |  |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |


<a name="minder-v1-DiffType"></a>

#### DiffType
DiffType defines the diff data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystems | [DiffType.Ecosystem](#minder-v1-DiffType-Ecosystem) | repeated | ecosystems is the list of ecosystems to be used for the "dep" diff type. |
| type | [string](#string) |  | type is the type of diff ingestor to use. The default is "dep" which will leverage the ecosystems array. |


<a name="minder-v1-DiffType-Ecosystem"></a>

#### DiffType.Ecosystem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the ecosystem. |
| depfile | [string](#string) |  | depfile is the file that contains the dependencies for this ecosystem |


<a name="minder-v1-EntityTypedId"></a>

#### EntityTypedId
EntiryTypeId is a message that carries an ID together with a type to uniquely identify an entity
such as (repo, 1), (artifact, 2), ...


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Entity](#minder-v1-Entity) |  | entity is the entity to get status for. Incompatible with `all` |
| id | [string](#string) |  | id is the ID of the entity to get status for. Incompatible with `all` |


<a name="minder-v1-EvalResultAlert"></a>

#### EvalResultAlert
EvalResultAlert holds the alert details for a given rule evaluation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  | status is the status of the alert |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the alert was performed or attempted |
| details | [string](#string) |  | details is the description of the alert attempt if any |
| url | [string](#string) |  | url is the URL to the alert |


<a name="minder-v1-GetArtifactByIdRequest"></a>

#### GetArtifactByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-GetArtifactByIdResponse"></a>

#### GetArtifactByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [Artifact](#minder-v1-Artifact) |  |  |
| versions | [ArtifactVersion](#minder-v1-ArtifactVersion) | repeated |  |


<a name="minder-v1-GetArtifactByNameRequest"></a>

#### GetArtifactByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-GetArtifactByNameResponse"></a>

#### GetArtifactByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [Artifact](#minder-v1-Artifact) |  |  |
| versions | [ArtifactVersion](#minder-v1-ArtifactVersion) | repeated |  |


<a name="minder-v1-GetAuthorizationURLRequest"></a>

#### GetAuthorizationURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cli | [bool](#bool) |  |  |
| port | [int32](#int32) |  |  |
| owner | [string](#string) | optional |  |
| context | [Context](#minder-v1-Context) |  |  |
| redirect_url | [string](#string) | optional |  |


<a name="minder-v1-GetAuthorizationURLResponse"></a>

#### GetAuthorizationURLResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  |  |


<a name="minder-v1-GetProfileByIdRequest"></a>

#### GetProfileByIdRequest
get profile by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context which contains the profiles |
| id | [string](#string) |  | id is the id of the profile to get |


<a name="minder-v1-GetProfileByIdResponse"></a>

#### GetProfileByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-GetProfileStatusByNameRequest"></a>

#### GetProfileStatusByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |
| name | [string](#string) |  | name is the name of the profile to get |
| entity | [EntityTypedId](#minder-v1-EntityTypedId) |  |  |
| all | [bool](#bool) |  |  |
| rule | [string](#string) |  | **Deprecated.** rule is the type of the rule. Deprecated in favor of rule_type |
| rule_type | [string](#string) |  |  |
| rule_name | [string](#string) |  |  |


<a name="minder-v1-GetProfileStatusByNameResponse"></a>

#### GetProfileStatusByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | [ProfileStatus](#minder-v1-ProfileStatus) |  | profile_status is the status of the profile |
| rule_evaluation_status | [RuleEvaluationStatus](#minder-v1-RuleEvaluationStatus) | repeated | rule_evaluation_status is the status of the rules |


<a name="minder-v1-GetProfileStatusByProjectRequest"></a>

#### GetProfileStatusByProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |


<a name="minder-v1-GetProfileStatusByProjectResponse"></a>

#### GetProfileStatusByProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | [ProfileStatus](#minder-v1-ProfileStatus) | repeated | profile_status is the status of the profile |


<a name="minder-v1-GetProviderRequest"></a>

#### GetProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the provider is evaluated. |
| name | [string](#string) |  | name is the name of the provider to get. |


<a name="minder-v1-GetProviderResponse"></a>

#### GetProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [Provider](#minder-v1-Provider) |  | provider is the provider that was retrieved. |


<a name="minder-v1-GetRepositoryByIdRequest"></a>

#### GetRepositoryByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-GetRepositoryByIdResponse"></a>

#### GetRepositoryByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#minder-v1-Repository) |  |  |


<a name="minder-v1-GetRepositoryByNameRequest"></a>

#### GetRepositoryByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| name | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-GetRepositoryByNameResponse"></a>

#### GetRepositoryByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#minder-v1-Repository) |  |  |


<a name="minder-v1-GetRuleTypeByIdRequest"></a>

#### GetRuleTypeByIdRequest
GetRuleTypeByIdRequest is the request to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the rule type. |


<a name="minder-v1-GetRuleTypeByIdResponse"></a>

#### GetRuleTypeByIdResponse
GetRuleTypeByIdResponse is the response to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type. |


<a name="minder-v1-GetRuleTypeByNameRequest"></a>

#### GetRuleTypeByNameRequest
GetRuleTypeByNameRequest is the request to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule type is evaluated. |
| name | [string](#string) |  | name is the name of the rule type. |


<a name="minder-v1-GetRuleTypeByNameResponse"></a>

#### GetRuleTypeByNameResponse
GetRuleTypeByNameResponse is the response to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type. |


<a name="minder-v1-GetUnclaimedProvidersRequest"></a>

#### GetUnclaimedProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the set of providers are evaluated. |


<a name="minder-v1-GetUnclaimedProvidersResponse"></a>

#### GetUnclaimedProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| providers | [ProviderParameter](#minder-v1-ProviderParameter) | repeated | providers is a set of parameters which can be supplied to allow the user to assign existing unclaimed credentials to a new provider in the project via CreateProvider(). |


<a name="minder-v1-GetUserRequest"></a>

#### GetUserRequest
list users
get user


<a name="minder-v1-GetUserResponse"></a>

#### GetUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#minder-v1-UserRecord) | optional |  |
| projects | [Project](#minder-v1-Project) | repeated |  |


<a name="minder-v1-GitHubAppParams"></a>

#### GitHubAppParams
GitHubAppParams is the parameters for a GitHub App provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| installation_id | [int64](#int64) |  | The GitHub installation ID for the app. On create, this is the only parameter used; the organization parameters are ignored. |
| organization | [string](#string) |  | The GitHub organization slug where the app is installed. This is an output-only parameter, and is validated on input if set (i.e. the value must be either empty or match the org of the installation_id). |
| organization_id | [int64](#int64) |  | The GitHub organization ID where the app is installed. This is an output-only parameter, and is validated on input if set (i.e. the value must be either empty or match the org of the installation_id). |


<a name="minder-v1-GitHubAppProviderConfig"></a>

#### GitHubAppProviderConfig
GitHubAppProviderConfig contains the configuration for the GitHub App provider


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | Endpoint is the GitHub API endpoint. If using the public GitHub API, Endpoint can be left blank. |


<a name="minder-v1-GitHubProviderConfig"></a>

#### GitHubProviderConfig
GitHubProviderConfig contains the configuration for the GitHub client

Endpoint: is the GitHub API endpoint

If using the public GitHub API, Endpoint can be left blank
disable revive linting for this struct as there is nothing wrong with the
naming convention


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | Endpoint is the GitHub API endpoint. If using the public GitHub API, Endpoint can be left blank. |


<a name="minder-v1-GitType"></a>

#### GitType
GitType defines the git data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clone_url | [string](#string) |  | clone_url is the url of the git repository. |
| branch | [string](#string) |  | branch is the branch of the git repository. |


<a name="minder-v1-ListArtifactsRequest"></a>

#### ListArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| context | [Context](#minder-v1-Context) |  |  |
| from | [string](#string) |  |  |


<a name="minder-v1-ListArtifactsResponse"></a>

#### ListArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [Artifact](#minder-v1-Artifact) | repeated |  |


<a name="minder-v1-ListEvaluationResultsRequest"></a>

#### ListEvaluationResultsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the evaluation results are evaluated. |
| profile | [string](#string) |  | ID can contain either a profile name or an ID |
| label_filter | [string](#string) |  | Filter profiles to only those matching the specified labels.

The default is to return all user-created profiles; the string "*" can be used to select all profiles, including system profiles. This syntax may be expanded in the future. |
| entity | [EntityTypedId](#minder-v1-EntityTypedId) | repeated | If set, only return evaluation results for the named entities. If empty, return evaluation results for all entities |
| rule_name | [string](#string) | repeated | If set, only return evaluation results for the named rules. If empty, return evaluation results for all rules |


<a name="minder-v1-ListEvaluationResultsResponse"></a>

#### ListEvaluationResultsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | [ListEvaluationResultsResponse.EntityEvaluationResults](#minder-v1-ListEvaluationResultsResponse-EntityEvaluationResults) | repeated | Each entity selected by the list request will have _single_ entry in entities which contains results of all evaluations for each profile. |


<a name="minder-v1-ListEvaluationResultsResponse-EntityEvaluationResults"></a>

#### ListEvaluationResultsResponse.EntityEvaluationResults



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | [EntityTypedId](#minder-v1-EntityTypedId) |  |  |
| profiles | [ListEvaluationResultsResponse.EntityProfileEvaluationResults](#minder-v1-ListEvaluationResultsResponse-EntityProfileEvaluationResults) | repeated |  |


<a name="minder-v1-ListEvaluationResultsResponse-EntityProfileEvaluationResults"></a>

#### ListEvaluationResultsResponse.EntityProfileEvaluationResults



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | [ProfileStatus](#minder-v1-ProfileStatus) |  | profile_status is the status of the profile - id, name, status, last_updated |
| results | [RuleEvaluationStatus](#minder-v1-RuleEvaluationStatus) | repeated | Note that some fields like profile_id and entity might be empty Eventually we might replace this type with another one that fits the API better |


<a name="minder-v1-ListProfilesRequest"></a>

#### ListProfilesRequest
list profiles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context which contains the profiles |
| label_filter | [string](#string) |  | Filter profiles to only those matching the specified labels.

The default is to return all user-created profiles; the string "*" can be used to select all profiles, including system profiles. This syntax may be expanded in the future. |


<a name="minder-v1-ListProfilesResponse"></a>

#### ListProfilesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profiles | [Profile](#minder-v1-Profile) | repeated |  |


<a name="minder-v1-ListProjectsRequest"></a>

#### ListProjectsRequest



<a name="minder-v1-ListProjectsResponse"></a>

#### ListProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#minder-v1-Project) | repeated |  |


<a name="minder-v1-ListProviderClassesRequest"></a>

#### ListProviderClassesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the provider classes are evaluated. |


<a name="minder-v1-ListProviderClassesResponse"></a>

#### ListProviderClassesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider_classes | [string](#string) | repeated | provider_classes is the list of provider classes. |


<a name="minder-v1-ListProvidersRequest"></a>

#### ListProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the providers are evaluated. |
| limit | [int32](#int32) |  | limit is the maximum number of providers to return. |
| cursor | [string](#string) |  | cursor is the cursor to use for the page of results, empty if at the beginning |


<a name="minder-v1-ListProvidersResponse"></a>

#### ListProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| providers | [Provider](#minder-v1-Provider) | repeated |  |
| cursor | [string](#string) |  | cursor is the cursor to use for the next page of results, empty if at the end |


<a name="minder-v1-ListRemoteRepositoriesFromProviderRequest"></a>

#### ListRemoteRepositoriesFromProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-ListRemoteRepositoriesFromProviderResponse"></a>

#### ListRemoteRepositoriesFromProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [UpstreamRepositoryRef](#minder-v1-UpstreamRepositoryRef) | repeated |  |


<a name="minder-v1-ListRepositoriesRequest"></a>

#### ListRepositoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| limit | [int64](#int64) |  |  |
| context | [Context](#minder-v1-Context) |  |  |
| cursor | [string](#string) |  |  |


<a name="minder-v1-ListRepositoriesResponse"></a>

#### ListRepositoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [Repository](#minder-v1-Repository) | repeated |  |
| cursor | [string](#string) |  | cursor is the cursor to use for the next page of results, empty if at the end |


<a name="minder-v1-ListRoleAssignmentsRequest"></a>

#### ListRoleAssignmentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the role assignments are evaluated. |


<a name="minder-v1-ListRoleAssignmentsResponse"></a>

#### ListRoleAssignmentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignments | [RoleAssignment](#minder-v1-RoleAssignment) | repeated |  |


<a name="minder-v1-ListRolesRequest"></a>

#### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the roles are evaluated. |


<a name="minder-v1-ListRolesResponse"></a>

#### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [Role](#minder-v1-Role) | repeated |  |


<a name="minder-v1-ListRuleTypesRequest"></a>

#### ListRuleTypesRequest
ListRuleTypesRequest is the request to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule types are evaluated. |


<a name="minder-v1-ListRuleTypesResponse"></a>

#### ListRuleTypesResponse
ListRuleTypesResponse is the response to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_types | [RuleType](#minder-v1-RuleType) | repeated | rule_types is the list of rule types. |


<a name="minder-v1-PatchProfileRequest"></a>

#### PatchProfileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | The context in which the patch is applied. Provided explicitly so that the patch itself can be minimal and contain only the attribute to set, e.g. remediate=true |
| id | [string](#string) |  | The id of the profile to patch. Same explanation about explicitness as for the context |
| patch | [Profile](#minder-v1-Profile) |  | The patch to apply to the profile |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | needed to enable PATCH, see https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/patch_feature/ is not exposed to the API user |


<a name="minder-v1-PatchProfileResponse"></a>

#### PatchProfileResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-PatchProjectRequest"></a>

#### PatchProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the project is updated. |
| patch | [ProjectPatch](#minder-v1-ProjectPatch) |  | patch is the patch to apply to the project |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | needed to enable PATCH, see https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/patch_feature/ is not exposed to the API user |


<a name="minder-v1-PatchProjectResponse"></a>

#### PatchProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#minder-v1-Project) |  | project is the project that was updated. |


<a name="minder-v1-PrContents"></a>

#### PrContents



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pr | [PullRequest](#minder-v1-PullRequest) |  |  |
| files | [PrContents.File](#minder-v1-PrContents-File) | repeated |  |


<a name="minder-v1-PrContents-File"></a>

#### PrContents.File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| file_patch_url | [string](#string) |  |  |
| patch_lines | [PrContents.File.Line](#minder-v1-PrContents-File-Line) | repeated |  |


<a name="minder-v1-PrContents-File-Line"></a>

#### PrContents.File.Line



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line_number | [int32](#int32) |  | Deliberately left as an int32: a diff with more than 2^31 lines could lead to various problems while processing. |
| content | [string](#string) |  |  |


<a name="minder-v1-PrDependencies"></a>

#### PrDependencies



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pr | [PullRequest](#minder-v1-PullRequest) |  |  |
| deps | [PrDependencies.ContextualDependency](#minder-v1-PrDependencies-ContextualDependency) | repeated |  |


<a name="minder-v1-PrDependencies-ContextualDependency"></a>

#### PrDependencies.ContextualDependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dep | [Dependency](#minder-v1-Dependency) |  |  |
| file | [PrDependencies.ContextualDependency.FilePatch](#minder-v1-PrDependencies-ContextualDependency-FilePatch) |  |  |


<a name="minder-v1-PrDependencies-ContextualDependency-FilePatch"></a>

#### PrDependencies.ContextualDependency.FilePatch



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | file changed, e.g. package-lock.json |
| patch_url | [string](#string) |  | points to the the raw patchfile |


<a name="minder-v1-Profile"></a>

#### Profile
Profile defines a profile that is user defined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the profile is evaluated. |
| id | [string](#string) | optional | id is the id of the profile. This is optional and is set by the system. |
| name | [string](#string) |  | name is the name of the profile instance. |
| labels | [string](#string) | repeated | labels are a set of system-provided attributes which can be used to filter profiles and status results. Labels cannot be set by the user, but are returned in ListProfiles.

Labels use DNS label constraints, with a possible namespace prefix separated by a colon (:). They are intended to allow filtering, but not to store arbitrary metadata. DNS labels are 1-63 character alphanumeric strings with internal hyphens. An RE2-style validation regex would be:

DNS_STR = "[a-zA-Z0-9](?[-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?" ($DNS_STR:)?$DNS_STR |
| repository | [Profile.Rule](#minder-v1-Profile-Rule) | repeated | These are the entities that one could set in the profile. |
| build_environment | [Profile.Rule](#minder-v1-Profile-Rule) | repeated |  |
| artifact | [Profile.Rule](#minder-v1-Profile-Rule) | repeated |  |
| pull_request | [Profile.Rule](#minder-v1-Profile-Rule) | repeated |  |
| remediate | [string](#string) | optional | whether and how to remediate (on,off,dry_run) this is optional and defaults to "off" |
| alert | [string](#string) | optional | whether and how to alert (on,off,dry_run) this is optional and defaults to "on" |
| type | [string](#string) |  | type is a placeholder for the object type. It should always be set to "profile". |
| version | [string](#string) |  | version is the version of the profile type. In this case, it is "v1" |
| display_name | [string](#string) |  | display_name is the display name of the profile. |


<a name="minder-v1-Profile-Rule"></a>

#### Profile.Rule
Rule defines the individual call of a certain rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the rule to be instantiated. |
| params | [google.protobuf.Struct](#google-protobuf-Struct) |  | params are the parameters that are passed to the rule. This is optional and depends on the rule type. |
| def | [google.protobuf.Struct](#google-protobuf-Struct) |  | def is the definition of the rule. This depends on the rule type. |
| name | [string](#string) |  | name is the descriptive name of the rule, not to be confused with type |


<a name="minder-v1-ProfileStatus"></a>

#### ProfileStatus
get the overall profile status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | [string](#string) |  | profile_id is the id of the profile |
| profile_name | [string](#string) |  | profile_name is the name of the profile |
| profile_status | [string](#string) |  | profile_status is the status of the profile |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the profile was updated |
| profile_display_name | [string](#string) |  | profile_display_name is the display name of the profile |


<a name="minder-v1-Project"></a>

#### Project
Project API Objects


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| display_name | [string](#string) |  | display_name allows for a human-readable name to be used. display_names are short *non-unique* strings to provide a user-friendly name for presentation in lists, etc. |


<a name="minder-v1-ProjectPatch"></a>

#### ProjectPatch



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| display_name | [string](#string) | optional | display_name is the display name of the project to update. |
| description | [string](#string) | optional | description is the description of the project to update. |


<a name="minder-v1-Provider"></a>

#### Provider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the provider. |
| class | [string](#string) |  | class is the name of the provider implementation, eg. 'github' or 'gh-app'. |
| project | [string](#string) |  | project is the project where the provider is. This is ignored on input in favor of the context field in CreateProviderRequest. |
| version | [string](#string) |  | version is the version of the provider. |
| implements | [ProviderType](#minder-v1-ProviderType) | repeated | implements is the list of interfaces that the provider implements. |
| config | [google.protobuf.Struct](#google-protobuf-Struct) |  | config is the configuration of the provider. |
| auth_flows | [AuthorizationFlow](#minder-v1-AuthorizationFlow) | repeated | auth_flows is the list of authorization flows that the provider supports. |
| parameters | [ProviderParameter](#minder-v1-ProviderParameter) |  | parameters is the list of parameters that the provider requires. |
| credentials_state | [string](#string) |  | credentials_state is the state of the credentials for the provider. This is an output-only field. It may be: "set", "unset", "not_applicable". |


<a name="minder-v1-ProviderParameter"></a>

#### ProviderParameter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| github_app | [GitHubAppParams](#minder-v1-GitHubAppParams) |  |  |


<a name="minder-v1-PullRequest"></a>

#### PullRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The full URL to the PR |
| commit_sha | [string](#string) |  | Commit SHA of the PR HEAD. Will be useful to submit a review |
| number | [int64](#int64) |  | The sequential PR number (not the DB PK!) |
| repo_owner | [string](#string) |  | The owner of the repo, will be used to submit a review |
| repo_name | [string](#string) |  | The name of the repo, will be used to submit a review |
| author_id | [int64](#int64) |  | The author of the PR, will be used to check if we can request changes |
| action | [string](#string) |  | The action that triggered the webhook |


<a name="minder-v1-RESTProviderConfig"></a>

#### RESTProviderConfig
RESTProviderConfig contains the configuration for the REST provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| base_url | [string](#string) |  | base_url is the base URL for the REST provider. |


<a name="minder-v1-RegisterRepoResult"></a>

#### RegisterRepoResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#minder-v1-Repository) |  |  |
| status | [RegisterRepoResult.Status](#minder-v1-RegisterRepoResult-Status) |  |  |


<a name="minder-v1-RegisterRepoResult-Status"></a>

#### RegisterRepoResult.Status



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) | optional |  |


<a name="minder-v1-RegisterRepositoryRequest"></a>

#### RegisterRepositoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| repository | [UpstreamRepositoryRef](#minder-v1-UpstreamRepositoryRef) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-RegisterRepositoryResponse"></a>

#### RegisterRepositoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| result | [RegisterRepoResult](#minder-v1-RegisterRepoResult) |  |  |


<a name="minder-v1-RemoveRoleRequest"></a>

#### RemoveRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the role assignment is evaluated. |
| role_assignment | [RoleAssignment](#minder-v1-RoleAssignment) |  | role_assignment is the role assignment to be removed. |


<a name="minder-v1-RemoveRoleResponse"></a>

#### RemoveRoleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignment | [RoleAssignment](#minder-v1-RoleAssignment) |  | role_assignment is the role assignment that was removed. |


<a name="minder-v1-Repository"></a>

#### Repository



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | This is optional when returning remote repositories |
| context | [Context](#minder-v1-Context) | optional |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int64](#int64) |  |  |
| hook_id | [int64](#int64) |  |  |
| hook_url | [string](#string) |  |  |
| deploy_url | [string](#string) |  |  |
| clone_url | [string](#string) |  |  |
| hook_name | [string](#string) |  |  |
| hook_type | [string](#string) |  |  |
| hook_uuid | [string](#string) |  |  |
| is_private | [bool](#bool) |  |  |
| is_fork | [bool](#bool) |  |  |
| registered | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| default_branch | [string](#string) |  |  |
| license | [string](#string) |  |  |


<a name="minder-v1-RestType"></a>

#### RestType
RestType defines the rest data evaluation.
This is used to fetch data from a REST endpoint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | endpoint is the endpoint to fetch data from. This can be a URL or the path on the API.bool This is a required field and must be set. This is also evaluated via a template which allows us dynamically fill in the values. |
| method | [string](#string) |  | method is the method to use to fetch data. |
| headers | [string](#string) | repeated | headers are the headers to be sent to the endpoint. |
| body | [string](#string) | optional | body is the body to be sent to the endpoint. |
| parse | [string](#string) |  | parse is the parsing mechanism to be used to parse the data. |
| fallback | [RestType.Fallback](#minder-v1-RestType-Fallback) | repeated | fallback provides a body that the ingester would return in case the REST call returns a non-200 status code. |


<a name="minder-v1-RestType-Fallback"></a>

#### RestType.Fallback



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_code | [int32](#int32) |  |  |
| body | [string](#string) |  |  |


<a name="minder-v1-Role"></a>

#### Role



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the role. |
| description | [string](#string) |  | description is the description of the role. |


<a name="minder-v1-RoleAssignment"></a>

#### RoleAssignment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | role is the role that is assigned. |
| subject | [string](#string) |  | subject is the subject to which the role is assigned. |
| project | [string](#string) | optional | projectt is the projectt in which the role is assigned. |


<a name="minder-v1-RpcOptions"></a>

#### RpcOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| no_log | [bool](#bool) |  |  |
| target_resource | [TargetResource](#minder-v1-TargetResource) |  |  |
| relation | [Relation](#minder-v1-Relation) |  |  |


<a name="minder-v1-RuleEvaluationStatus"></a>

#### RuleEvaluationStatus
get the status of the rules for a given profile


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | [string](#string) |  | profile_id is the id of the profile |
| rule_id | [string](#string) |  | rule_id is the id of the rule |
| rule_name | [string](#string) |  | **Deprecated.** rule_name is the type of the rule. Deprecated in favor of rule_type_name |
| entity | [string](#string) |  | entity is the entity that was evaluated |
| status | [string](#string) |  | status is the status of the evaluation |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the profile was updated |
| entity_info | [RuleEvaluationStatus.EntityInfoEntry](#minder-v1-RuleEvaluationStatus-EntityInfoEntry) | repeated | entity_info is the information about the entity |
| details | [string](#string) |  | details is the description of the evaluation if any |
| guidance | [string](#string) |  | guidance is the guidance for the evaluation if any |
| remediation_status | [string](#string) |  | remediation_status is the status of the remediation |
| remediation_last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | remediation_last_updated is the last time the remediation was performed or attempted |
| remediation_details | [string](#string) |  | remediation_details is the description of the remediation attempt if any |
| rule_type_name | [string](#string) |  | rule_type_name is the name of the rule |
| rule_description_name | [string](#string) |  | rule_description_name is the name to describe the rule |
| alert | [EvalResultAlert](#minder-v1-EvalResultAlert) |  | alert holds the alert details if the rule generated an alert in an external system |
| severity | [Severity](#minder-v1-Severity) |  | severity is the severity of the rule |
| rule_evaluation_id | [string](#string) |  | rule_evaluation_id is the id of the rule evaluation |


<a name="minder-v1-RuleEvaluationStatus-EntityInfoEntry"></a>

#### RuleEvaluationStatus.EntityInfoEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |


<a name="minder-v1-RuleType"></a>

#### RuleType
RuleType defines rules that may or may not be user defined.
The version is assumed from the folder's version.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | id is the id of the rule type. This is mostly optional and is set by the server. |
| name | [string](#string) |  | name is the name of the rule type. |
| context | [Context](#minder-v1-Context) |  | context is the context in which the rule is evaluated. |
| def | [RuleType.Definition](#minder-v1-RuleType-Definition) |  | def is the definition of the rule type. |
| description | [string](#string) |  | description is the description of the rule type. |
| guidance | [string](#string) |  | guidance are instructions we give the user in case a rule fails. |
| severity | [Severity](#minder-v1-Severity) |  | severity is the severity of the rule type. |


<a name="minder-v1-RuleType-Definition"></a>

#### RuleType.Definition
Definition defines the rule type. It encompases the schema and the data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| in_entity | [string](#string) |  | in_entity is the entity in which the rule is evaluated. This can be repository, build_environment or artifact. |
| rule_schema | [google.protobuf.Struct](#google-protobuf-Struct) |  | rule_schema is the schema of the rule. This is expressed in JSON Schema. |
| param_schema | [google.protobuf.Struct](#google-protobuf-Struct) | optional | param_schema is the schema of the parameters that are passed to the rule. This is expressed in JSON Schema. |
| ingest | [RuleType.Definition.Ingest](#minder-v1-RuleType-Definition-Ingest) |  |  |
| eval | [RuleType.Definition.Eval](#minder-v1-RuleType-Definition-Eval) |  |  |
| remediate | [RuleType.Definition.Remediate](#minder-v1-RuleType-Definition-Remediate) |  |  |
| alert | [RuleType.Definition.Alert](#minder-v1-RuleType-Definition-Alert) |  |  |


<a name="minder-v1-RuleType-Definition-Alert"></a>

#### RuleType.Definition.Alert



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| security_advisory | [RuleType.Definition.Alert.AlertTypeSA](#minder-v1-RuleType-Definition-Alert-AlertTypeSA) | optional |  |


<a name="minder-v1-RuleType-Definition-Alert-AlertTypeSA"></a>

#### RuleType.Definition.Alert.AlertTypeSA



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| severity | [string](#string) |  |  |


<a name="minder-v1-RuleType-Definition-Eval"></a>

#### RuleType.Definition.Eval
Eval defines the data evaluation definition.
This pertains to the way we traverse data from the upstream
endpoint and how we compare it to the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data evaluation. Right now only `jq` is supported as a driver |
| jq | [RuleType.Definition.Eval.JQComparison](#minder-v1-RuleType-Definition-Eval-JQComparison) | repeated | jq is only used if the `jq` type is selected. It defines the comparisons that are made between the ingested data and the profile rule. |
| rego | [RuleType.Definition.Eval.Rego](#minder-v1-RuleType-Definition-Eval-Rego) | optional | rego is only used if the `rego` type is selected. |
| vulncheck | [RuleType.Definition.Eval.Vulncheck](#minder-v1-RuleType-Definition-Eval-Vulncheck) | optional | vulncheck is only used if the `vulncheck` type is selected. |
| trusty | [RuleType.Definition.Eval.Trusty](#minder-v1-RuleType-Definition-Eval-Trusty) | optional | The trusty type is no longer used, but is still here for backwards compatibility with existing stored rules |
| homoglyphs | [RuleType.Definition.Eval.Homoglyphs](#minder-v1-RuleType-Definition-Eval-Homoglyphs) | optional | homoglyphs is only used if the `homoglyphs` type is selected. |


<a name="minder-v1-RuleType-Definition-Eval-Homoglyphs"></a>

#### RuleType.Definition.Eval.Homoglyphs



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |


<a name="minder-v1-RuleType-Definition-Eval-JQComparison"></a>

#### RuleType.Definition.Eval.JQComparison



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ingested | [RuleType.Definition.Eval.JQComparison.Operator](#minder-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Ingested points to the data retrieved in the `ingest` section |
| profile | [RuleType.Definition.Eval.JQComparison.Operator](#minder-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Profile points to the profile itself. |


<a name="minder-v1-RuleType-Definition-Eval-JQComparison-Operator"></a>

#### RuleType.Definition.Eval.JQComparison.Operator



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| def | [string](#string) |  |  |


<a name="minder-v1-RuleType-Definition-Eval-Rego"></a>

#### RuleType.Definition.Eval.Rego



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of evaluation engine to use for rego. We currently have two modes of operation: - deny-by-default: this is the default mode of operation where we deny access by default and allow access only if the profile explicitly allows it. It expects the profile to set an `allow` variable to true or false. - constraints: this is the mode of operation where we allow access by default and deny access only if a violation is found. It expects the profile to set a `violations` variable with a "msg" field. |
| def | [string](#string) |  | def is the definition of the rego profile. |
| violation_format | [string](#string) | optional | how are violations reported. This is only used if the `constraints` type is selected. The default is `text` which returns human-readable text. The other option is `json` which returns a JSON array containing the violations. |


<a name="minder-v1-RuleType-Definition-Eval-Trusty"></a>

#### RuleType.Definition.Eval.Trusty



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | This is no longer used, but is still here for backwards compatibility with existing stored rules |


<a name="minder-v1-RuleType-Definition-Eval-Vulncheck"></a>

#### RuleType.Definition.Eval.Vulncheck
no configuration for now


<a name="minder-v1-RuleType-Definition-Ingest"></a>

#### RuleType.Definition.Ingest
Ingest defines how the data is ingested.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data ingestion. we currently support rest, artifact and builtin. |
| rest | [RestType](#minder-v1-RestType) | optional | rest is the rest data ingestion. this is only used if the type is rest. |
| builtin | [BuiltinType](#minder-v1-BuiltinType) | optional | builtin is the builtin data ingestion. |
| artifact | [ArtifactType](#minder-v1-ArtifactType) | optional | artifact is the artifact data ingestion. |
| git | [GitType](#minder-v1-GitType) | optional | git is the git data ingestion. |
| diff | [DiffType](#minder-v1-DiffType) | optional | diff is the diff data ingestion. |


<a name="minder-v1-RuleType-Definition-Remediate"></a>

#### RuleType.Definition.Remediate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| rest | [RestType](#minder-v1-RestType) | optional |  |
| gh_branch_protection | [RuleType.Definition.Remediate.GhBranchProtectionType](#minder-v1-RuleType-Definition-Remediate-GhBranchProtectionType) | optional |  |
| pull_request | [RuleType.Definition.Remediate.PullRequestRemediation](#minder-v1-RuleType-Definition-Remediate-PullRequestRemediation) | optional |  |


<a name="minder-v1-RuleType-Definition-Remediate-GhBranchProtectionType"></a>

#### RuleType.Definition.Remediate.GhBranchProtectionType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| patch | [string](#string) |  |  |


<a name="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation"></a>

#### RuleType.Definition.Remediate.PullRequestRemediation
the name stutters a bit but we already use a PullRequest message for handling PR entities


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | the title of the PR |
| body | [string](#string) |  | the body of the PR |
| contents | [RuleType.Definition.Remediate.PullRequestRemediation.Content](#minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content) | repeated |  |
| method | [string](#string) |  | the method to use to create the PR. For now, these are supported: -- minder.content - ensures that the content of the file is exactly as specified refer to the Content message for more details -- minder.actions.replace_tags_with_sha - finds any github actions within a workflow file and replaces the tag with the SHA |
| actions_replace_tags_with_sha | [RuleType.Definition.Remediate.PullRequestRemediation.ActionsReplaceTagsWithSha](#minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-ActionsReplaceTagsWithSha) | optional | If the method is minder.actions.replace_tags_with_sha, this is the configuration for that method |


<a name="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-ActionsReplaceTagsWithSha"></a>

#### RuleType.Definition.Remediate.PullRequestRemediation.ActionsReplaceTagsWithSha



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exclude | [string](#string) | repeated | List of actions to exclude from the replacement |


<a name="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content"></a>

#### RuleType.Definition.Remediate.PullRequestRemediation.Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | the file to patch |
| action | [string](#string) |  | how to patch the file. For now, only replace is supported |
| content | [string](#string) |  | the content of the file |
| mode | [string](#string) | optional | the GIT mode of the file. Not UNIX mode! String because the GH API also uses strings the usual modes are: 100644 for regular files, 100755 for executable files and 040000 for submodules (which we don't use but now you know the meaning of the 1 in 100644) see e.g. https://github.com/go-git/go-git/blob/32e0172851c35ae2fac495069c923330040903d2/plumbing/filemode/filemode.go#L16 |


<a name="minder-v1-Severity"></a>

#### Severity
Severity defines the severity of the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [Severity.Value](#minder-v1-Severity-Value) |  | value is the severity value. |


<a name="minder-v1-StoreProviderTokenRequest"></a>

#### StoreProviderTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| access_token | [string](#string) |  |  |
| owner | [string](#string) | optional |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-StoreProviderTokenResponse"></a>

#### StoreProviderTokenResponse



<a name="minder-v1-UpdateProfileRequest"></a>

#### UpdateProfileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-UpdateProfileResponse"></a>

#### UpdateProfileResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#minder-v1-Profile) |  |  |


<a name="minder-v1-UpdateProjectRequest"></a>

#### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#minder-v1-Context) |  | context is the context in which the project is updated. |
| display_name | [string](#string) |  | display_name is the display name of the project to update. |
| description | [string](#string) |  | description is the description of the project to update. |


<a name="minder-v1-UpdateProjectResponse"></a>

#### UpdateProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#minder-v1-Project) |  | project is the project that was updated. |


<a name="minder-v1-UpdateRuleTypeRequest"></a>

#### UpdateRuleTypeRequest
UpdateRuleTypeRequest is the request to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type to be updated. |


<a name="minder-v1-UpdateRuleTypeResponse"></a>

#### UpdateRuleTypeResponse
UpdateRuleTypeResponse is the response to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#minder-v1-RuleType) |  | rule_type is the rule type that was updated. |


<a name="minder-v1-UpstreamRepositoryRef"></a>

#### UpstreamRepositoryRef



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int64](#int64) |  | The upstream identity of the repository, as an integer. This is only set on output, and is ignored on input. |


<a name="minder-v1-UserRecord"></a>

#### UserRecord
user record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| identity_subject | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="minder-v1-VerifyProviderTokenFromRequest"></a>

#### VerifyProviderTokenFromRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | **Deprecated.**  |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| context | [Context](#minder-v1-Context) |  |  |


<a name="minder-v1-VerifyProviderTokenFromResponse"></a>

#### VerifyProviderTokenFromResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |


| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| name | string | .google.protobuf.EnumValueOptions | 42445 |  |
| rpc_options | RpcOptions | .google.protobuf.MethodOptions | 51077 |  |





<a name="minder-v1-AuthorizationFlow"></a>

### AuthorizationFlow


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHORIZATION_FLOW_UNSPECIFIED | 0 |  |
| AUTHORIZATION_FLOW_NONE | 1 |  |
| AUTHORIZATION_FLOW_USER_INPUT | 2 |  |
| AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW | 3 |  |
| AUTHORIZATION_FLOW_GITHUB_APP_FLOW | 4 |  |


<a name="minder-v1-DepEcosystem"></a>

### DepEcosystem


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEP_ECOSYSTEM_UNSPECIFIED | 0 |  |
| DEP_ECOSYSTEM_NPM | 1 |  |
| DEP_ECOSYSTEM_GO | 2 |  |
| DEP_ECOSYSTEM_PYPI | 3 |  |


<a name="minder-v1-Entity"></a>

### Entity
Entity defines the entity that is supported by the provider.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITY_UNSPECIFIED | 0 |  |
| ENTITY_REPOSITORIES | 1 |  |
| ENTITY_BUILD_ENVIRONMENTS | 2 |  |
| ENTITY_ARTIFACTS | 3 |  |
| ENTITY_PULL_REQUESTS | 4 |  |


<a name="minder-v1-ObjectOwner"></a>

### ObjectOwner


| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_OWNER_UNSPECIFIED | 0 |  |
| OBJECT_OWNER_PROJECT | 2 |  |
| OBJECT_OWNER_USER | 3 |  |


<a name="minder-v1-ProviderType"></a>

### ProviderType
ProviderTrait is the type of the provider.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| PROVIDER_TYPE_GITHUB | 1 |  |
| PROVIDER_TYPE_REST | 2 |  |
| PROVIDER_TYPE_GIT | 3 |  |
| PROVIDER_TYPE_OCI | 4 |  |
| PROVIDER_TYPE_REPO_LISTER | 5 |  |
| PROVIDER_TYPE_IMAGE_LISTER | 6 |  |
| PROVIDER_TYPE_GHCR | 7 |  |
| PROVIDER_TYPE_DOCKERHUB | 8 |  |


<a name="minder-v1-Relation"></a>

### Relation


| Name | Number | Description |
| ---- | ------ | ----------- |
| RELATION_UNSPECIFIED | 0 |  |
| RELATION_CREATE | 1 |  |
| RELATION_GET | 2 |  |
| RELATION_UPDATE | 3 |  |
| RELATION_DELETE | 4 |  |
| RELATION_ROLE_LIST | 5 |  |
| RELATION_ROLE_ASSIGNMENT_LIST | 6 |  |
| RELATION_ROLE_ASSIGNMENT_CREATE | 7 |  |
| RELATION_ROLE_ASSIGNMENT_REMOVE | 8 |  |
| RELATION_REPO_GET | 9 |  |
| RELATION_REPO_CREATE | 10 |  |
| RELATION_REPO_UPDATE | 11 |  |
| RELATION_REPO_DELETE | 12 |  |
| RELATION_ARTIFACT_GET | 13 |  |
| RELATION_ARTIFACT_CREATE | 14 |  |
| RELATION_ARTIFACT_UPDATE | 15 |  |
| RELATION_ARTIFACT_DELETE | 16 |  |
| RELATION_PR_GET | 17 |  |
| RELATION_PR_CREATE | 18 |  |
| RELATION_PR_UPDATE | 19 |  |
| RELATION_PR_DELETE | 20 |  |
| RELATION_PROVIDER_GET | 21 |  |
| RELATION_PROVIDER_CREATE | 22 |  |
| RELATION_PROVIDER_UPDATE | 23 |  |
| RELATION_PROVIDER_DELETE | 24 |  |
| RELATION_RULE_TYPE_GET | 25 |  |
| RELATION_RULE_TYPE_CREATE | 26 |  |
| RELATION_RULE_TYPE_UPDATE | 27 |  |
| RELATION_RULE_TYPE_DELETE | 28 |  |
| RELATION_PROFILE_GET | 29 |  |
| RELATION_PROFILE_CREATE | 30 |  |
| RELATION_PROFILE_UPDATE | 31 |  |
| RELATION_PROFILE_DELETE | 32 |  |
| RELATION_PROFILE_STATUS_GET | 33 |  |
| RELATION_REMOTE_REPO_GET | 34 |  |
| RELATION_ENTITY_RECONCILIATION_TASK_CREATE | 35 |  |


<a name="minder-v1-Severity-Value"></a>

### Severity.Value
Value enumerates the severity values.

| Name | Number | Description |
| ---- | ------ | ----------- |
| VALUE_UNSPECIFIED | 0 |  |
| VALUE_UNKNOWN | 1 | unknown severity means that the severity is unknown or hasn't been set. |
| VALUE_INFO | 2 | info severity means that the severity is informational and does not incur risk. |
| VALUE_LOW | 3 | low severity means that the severity is low and does not incur significant risk. |
| VALUE_MEDIUM | 4 | medium severity means that the severity is medium and may incur some risk. |
| VALUE_HIGH | 5 | high severity means that the severity is high and may incur significant risk. |
| VALUE_CRITICAL | 6 | critical severity means that the severity is critical and requires immediate attention. |


<a name="minder-v1-TargetResource"></a>

### TargetResource


| Name | Number | Description |
| ---- | ------ | ----------- |
| TARGET_RESOURCE_UNSPECIFIED | 0 |  |
| TARGET_RESOURCE_NONE | 1 |  |
| TARGET_RESOURCE_USER | 2 |  |
| TARGET_RESOURCE_PROJECT | 3 |  |




<a name="minder_v1_minder-proto-extensions"></a>

### File-level Extensions
| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| name | string | .google.protobuf.EnumValueOptions | 42445 |  |
| rpc_options | RpcOptions | .google.protobuf.MethodOptions | 51077 |  |





## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |
