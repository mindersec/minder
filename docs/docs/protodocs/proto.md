# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [mediator/v1/mediator.proto](#mediator_v1_mediator-proto)
    - [Artifact](#mediator-v1-Artifact)
    - [ArtifactType](#mediator-v1-ArtifactType)
    - [ArtifactVersion](#mediator-v1-ArtifactVersion)
    - [BranchProtection](#mediator-v1-BranchProtection)
    - [BuiltinType](#mediator-v1-BuiltinType)
    - [CheckHealthRequest](#mediator-v1-CheckHealthRequest)
    - [CheckHealthResponse](#mediator-v1-CheckHealthResponse)
    - [Context](#mediator-v1-Context)
    - [CreateGroupRequest](#mediator-v1-CreateGroupRequest)
    - [CreateGroupResponse](#mediator-v1-CreateGroupResponse)
    - [CreateKeyPairRequest](#mediator-v1-CreateKeyPairRequest)
    - [CreateKeyPairResponse](#mediator-v1-CreateKeyPairResponse)
    - [CreateOrganizationRequest](#mediator-v1-CreateOrganizationRequest)
    - [CreateOrganizationResponse](#mediator-v1-CreateOrganizationResponse)
    - [CreatePolicyRequest](#mediator-v1-CreatePolicyRequest)
    - [CreatePolicyResponse](#mediator-v1-CreatePolicyResponse)
    - [CreateRoleByGroupRequest](#mediator-v1-CreateRoleByGroupRequest)
    - [CreateRoleByGroupResponse](#mediator-v1-CreateRoleByGroupResponse)
    - [CreateRoleByOrganizationRequest](#mediator-v1-CreateRoleByOrganizationRequest)
    - [CreateRoleByOrganizationResponse](#mediator-v1-CreateRoleByOrganizationResponse)
    - [CreateRuleTypeRequest](#mediator-v1-CreateRuleTypeRequest)
    - [CreateRuleTypeResponse](#mediator-v1-CreateRuleTypeResponse)
    - [CreateUserRequest](#mediator-v1-CreateUserRequest)
    - [CreateUserResponse](#mediator-v1-CreateUserResponse)
    - [DeleteGroupRequest](#mediator-v1-DeleteGroupRequest)
    - [DeleteGroupResponse](#mediator-v1-DeleteGroupResponse)
    - [DeleteOrganizationRequest](#mediator-v1-DeleteOrganizationRequest)
    - [DeleteOrganizationResponse](#mediator-v1-DeleteOrganizationResponse)
    - [DeletePolicyRequest](#mediator-v1-DeletePolicyRequest)
    - [DeletePolicyResponse](#mediator-v1-DeletePolicyResponse)
    - [DeleteRoleRequest](#mediator-v1-DeleteRoleRequest)
    - [DeleteRoleResponse](#mediator-v1-DeleteRoleResponse)
    - [DeleteRuleTypeRequest](#mediator-v1-DeleteRuleTypeRequest)
    - [DeleteRuleTypeResponse](#mediator-v1-DeleteRuleTypeResponse)
    - [DeleteUserRequest](#mediator-v1-DeleteUserRequest)
    - [DeleteUserResponse](#mediator-v1-DeleteUserResponse)
    - [Dependency](#mediator-v1-Dependency)
    - [DiffType](#mediator-v1-DiffType)
    - [DiffType.Ecosystem](#mediator-v1-DiffType-Ecosystem)
    - [ExchangeCodeForTokenCLIRequest](#mediator-v1-ExchangeCodeForTokenCLIRequest)
    - [ExchangeCodeForTokenWEBRequest](#mediator-v1-ExchangeCodeForTokenWEBRequest)
    - [ExchangeCodeForTokenWEBResponse](#mediator-v1-ExchangeCodeForTokenWEBResponse)
    - [FilePatch](#mediator-v1-FilePatch)
    - [GetArtifactByIdRequest](#mediator-v1-GetArtifactByIdRequest)
    - [GetArtifactByIdResponse](#mediator-v1-GetArtifactByIdResponse)
    - [GetAuthorizationURLRequest](#mediator-v1-GetAuthorizationURLRequest)
    - [GetAuthorizationURLResponse](#mediator-v1-GetAuthorizationURLResponse)
    - [GetBranchProtectionRequest](#mediator-v1-GetBranchProtectionRequest)
    - [GetBranchProtectionResponse](#mediator-v1-GetBranchProtectionResponse)
    - [GetGroupByIdRequest](#mediator-v1-GetGroupByIdRequest)
    - [GetGroupByIdResponse](#mediator-v1-GetGroupByIdResponse)
    - [GetGroupByNameRequest](#mediator-v1-GetGroupByNameRequest)
    - [GetGroupByNameResponse](#mediator-v1-GetGroupByNameResponse)
    - [GetGroupsRequest](#mediator-v1-GetGroupsRequest)
    - [GetGroupsResponse](#mediator-v1-GetGroupsResponse)
    - [GetOrganizationByNameRequest](#mediator-v1-GetOrganizationByNameRequest)
    - [GetOrganizationByNameResponse](#mediator-v1-GetOrganizationByNameResponse)
    - [GetOrganizationRequest](#mediator-v1-GetOrganizationRequest)
    - [GetOrganizationResponse](#mediator-v1-GetOrganizationResponse)
    - [GetOrganizationsRequest](#mediator-v1-GetOrganizationsRequest)
    - [GetOrganizationsResponse](#mediator-v1-GetOrganizationsResponse)
    - [GetPolicyByIdRequest](#mediator-v1-GetPolicyByIdRequest)
    - [GetPolicyByIdResponse](#mediator-v1-GetPolicyByIdResponse)
    - [GetPolicyStatusByGroupRequest](#mediator-v1-GetPolicyStatusByGroupRequest)
    - [GetPolicyStatusByGroupResponse](#mediator-v1-GetPolicyStatusByGroupResponse)
    - [GetPolicyStatusByIdRequest](#mediator-v1-GetPolicyStatusByIdRequest)
    - [GetPolicyStatusByIdRequest.EntityTypedId](#mediator-v1-GetPolicyStatusByIdRequest-EntityTypedId)
    - [GetPolicyStatusByIdResponse](#mediator-v1-GetPolicyStatusByIdResponse)
    - [GetPublicKeyRequest](#mediator-v1-GetPublicKeyRequest)
    - [GetPublicKeyResponse](#mediator-v1-GetPublicKeyResponse)
    - [GetRepositoryByIdRequest](#mediator-v1-GetRepositoryByIdRequest)
    - [GetRepositoryByIdResponse](#mediator-v1-GetRepositoryByIdResponse)
    - [GetRepositoryByNameRequest](#mediator-v1-GetRepositoryByNameRequest)
    - [GetRepositoryByNameResponse](#mediator-v1-GetRepositoryByNameResponse)
    - [GetRoleByIdRequest](#mediator-v1-GetRoleByIdRequest)
    - [GetRoleByIdResponse](#mediator-v1-GetRoleByIdResponse)
    - [GetRoleByNameRequest](#mediator-v1-GetRoleByNameRequest)
    - [GetRoleByNameResponse](#mediator-v1-GetRoleByNameResponse)
    - [GetRolesByGroupRequest](#mediator-v1-GetRolesByGroupRequest)
    - [GetRolesByGroupResponse](#mediator-v1-GetRolesByGroupResponse)
    - [GetRolesRequest](#mediator-v1-GetRolesRequest)
    - [GetRolesResponse](#mediator-v1-GetRolesResponse)
    - [GetRuleTypeByIdRequest](#mediator-v1-GetRuleTypeByIdRequest)
    - [GetRuleTypeByIdResponse](#mediator-v1-GetRuleTypeByIdResponse)
    - [GetRuleTypeByNameRequest](#mediator-v1-GetRuleTypeByNameRequest)
    - [GetRuleTypeByNameResponse](#mediator-v1-GetRuleTypeByNameResponse)
    - [GetSecretByIdRequest](#mediator-v1-GetSecretByIdRequest)
    - [GetSecretByIdResponse](#mediator-v1-GetSecretByIdResponse)
    - [GetSecretsRequest](#mediator-v1-GetSecretsRequest)
    - [GetSecretsResponse](#mediator-v1-GetSecretsResponse)
    - [GetUserByEmailRequest](#mediator-v1-GetUserByEmailRequest)
    - [GetUserByEmailResponse](#mediator-v1-GetUserByEmailResponse)
    - [GetUserByIdRequest](#mediator-v1-GetUserByIdRequest)
    - [GetUserByIdResponse](#mediator-v1-GetUserByIdResponse)
    - [GetUserByUserNameRequest](#mediator-v1-GetUserByUserNameRequest)
    - [GetUserByUserNameResponse](#mediator-v1-GetUserByUserNameResponse)
    - [GetUserRequest](#mediator-v1-GetUserRequest)
    - [GetUserResponse](#mediator-v1-GetUserResponse)
    - [GetUsersByGroupRequest](#mediator-v1-GetUsersByGroupRequest)
    - [GetUsersByGroupResponse](#mediator-v1-GetUsersByGroupResponse)
    - [GetUsersByOrganizationRequest](#mediator-v1-GetUsersByOrganizationRequest)
    - [GetUsersByOrganizationResponse](#mediator-v1-GetUsersByOrganizationResponse)
    - [GetUsersRequest](#mediator-v1-GetUsersRequest)
    - [GetUsersResponse](#mediator-v1-GetUsersResponse)
    - [GetVulnerabilitiesRequest](#mediator-v1-GetVulnerabilitiesRequest)
    - [GetVulnerabilitiesResponse](#mediator-v1-GetVulnerabilitiesResponse)
    - [GetVulnerabilityByIdRequest](#mediator-v1-GetVulnerabilityByIdRequest)
    - [GetVulnerabilityByIdResponse](#mediator-v1-GetVulnerabilityByIdResponse)
    - [GitType](#mediator-v1-GitType)
    - [GithubWorkflow](#mediator-v1-GithubWorkflow)
    - [GroupRecord](#mediator-v1-GroupRecord)
    - [ListArtifactsRequest](#mediator-v1-ListArtifactsRequest)
    - [ListArtifactsResponse](#mediator-v1-ListArtifactsResponse)
    - [ListPoliciesRequest](#mediator-v1-ListPoliciesRequest)
    - [ListPoliciesResponse](#mediator-v1-ListPoliciesResponse)
    - [ListRepositoriesRequest](#mediator-v1-ListRepositoriesRequest)
    - [ListRepositoriesResponse](#mediator-v1-ListRepositoriesResponse)
    - [ListRuleTypesRequest](#mediator-v1-ListRuleTypesRequest)
    - [ListRuleTypesResponse](#mediator-v1-ListRuleTypesResponse)
    - [LogInRequest](#mediator-v1-LogInRequest)
    - [LogInResponse](#mediator-v1-LogInResponse)
    - [LogOutRequest](#mediator-v1-LogOutRequest)
    - [LogOutResponse](#mediator-v1-LogOutResponse)
    - [OrganizationRecord](#mediator-v1-OrganizationRecord)
    - [PipelinePolicy](#mediator-v1-PipelinePolicy)
    - [PipelinePolicy.ContextualRuleSet](#mediator-v1-PipelinePolicy-ContextualRuleSet)
    - [PipelinePolicy.Rule](#mediator-v1-PipelinePolicy-Rule)
    - [PolicyStatus](#mediator-v1-PolicyStatus)
    - [PrDependencies](#mediator-v1-PrDependencies)
    - [PrDependencies.ContextualDependency](#mediator-v1-PrDependencies-ContextualDependency)
    - [Provider](#mediator-v1-Provider)
    - [Provider.Context](#mediator-v1-Provider-Context)
    - [Provider.Definition](#mediator-v1-Provider-Definition)
    - [Provider.Definition.Auth](#mediator-v1-Provider-Definition-Auth)
    - [Provider.Definition.Auth.OAuth2](#mediator-v1-Provider-Definition-Auth-OAuth2)
    - [Provider.Definition.ClientTypes](#mediator-v1-Provider-Definition-ClientTypes)
    - [PullRequest](#mediator-v1-PullRequest)
    - [RefreshTokenRequest](#mediator-v1-RefreshTokenRequest)
    - [RefreshTokenResponse](#mediator-v1-RefreshTokenResponse)
    - [RegisterRepositoryRequest](#mediator-v1-RegisterRepositoryRequest)
    - [RegisterRepositoryResponse](#mediator-v1-RegisterRepositoryResponse)
    - [Repositories](#mediator-v1-Repositories)
    - [RepositoryRecord](#mediator-v1-RepositoryRecord)
    - [RepositoryResult](#mediator-v1-RepositoryResult)
    - [RestType](#mediator-v1-RestType)
    - [RevokeOauthGroupTokenRequest](#mediator-v1-RevokeOauthGroupTokenRequest)
    - [RevokeOauthGroupTokenResponse](#mediator-v1-RevokeOauthGroupTokenResponse)
    - [RevokeOauthTokensRequest](#mediator-v1-RevokeOauthTokensRequest)
    - [RevokeOauthTokensResponse](#mediator-v1-RevokeOauthTokensResponse)
    - [RevokeTokensRequest](#mediator-v1-RevokeTokensRequest)
    - [RevokeTokensResponse](#mediator-v1-RevokeTokensResponse)
    - [RevokeUserTokenRequest](#mediator-v1-RevokeUserTokenRequest)
    - [RevokeUserTokenResponse](#mediator-v1-RevokeUserTokenResponse)
    - [RoleRecord](#mediator-v1-RoleRecord)
    - [RuleEvaluationStatus](#mediator-v1-RuleEvaluationStatus)
    - [RuleEvaluationStatus.EntityInfoEntry](#mediator-v1-RuleEvaluationStatus-EntityInfoEntry)
    - [RuleType](#mediator-v1-RuleType)
    - [RuleType.Definition](#mediator-v1-RuleType-Definition)
    - [RuleType.Definition.Eval](#mediator-v1-RuleType-Definition-Eval)
    - [RuleType.Definition.Eval.JQComparison](#mediator-v1-RuleType-Definition-Eval-JQComparison)
    - [RuleType.Definition.Eval.JQComparison.Operator](#mediator-v1-RuleType-Definition-Eval-JQComparison-Operator)
    - [RuleType.Definition.Eval.Rego](#mediator-v1-RuleType-Definition-Eval-Rego)
    - [RuleType.Definition.Ingest](#mediator-v1-RuleType-Definition-Ingest)
    - [SignatureVerification](#mediator-v1-SignatureVerification)
    - [StoreProviderTokenRequest](#mediator-v1-StoreProviderTokenRequest)
    - [StoreProviderTokenResponse](#mediator-v1-StoreProviderTokenResponse)
    - [SyncRepositoriesRequest](#mediator-v1-SyncRepositoriesRequest)
    - [SyncRepositoriesResponse](#mediator-v1-SyncRepositoriesResponse)
    - [UpdatePasswordRequest](#mediator-v1-UpdatePasswordRequest)
    - [UpdatePasswordResponse](#mediator-v1-UpdatePasswordResponse)
    - [UpdateProfileRequest](#mediator-v1-UpdateProfileRequest)
    - [UpdateProfileResponse](#mediator-v1-UpdateProfileResponse)
    - [UpdateRuleTypeRequest](#mediator-v1-UpdateRuleTypeRequest)
    - [UpdateRuleTypeResponse](#mediator-v1-UpdateRuleTypeResponse)
    - [UserRecord](#mediator-v1-UserRecord)
    - [VerifyProviderTokenFromRequest](#mediator-v1-VerifyProviderTokenFromRequest)
    - [VerifyProviderTokenFromResponse](#mediator-v1-VerifyProviderTokenFromResponse)
    - [VerifyRequest](#mediator-v1-VerifyRequest)
    - [VerifyResponse](#mediator-v1-VerifyResponse)
    - [VersionedArtifact](#mediator-v1-VersionedArtifact)
  
    - [DepEcosystem](#mediator-v1-DepEcosystem)
    - [Entity](#mediator-v1-Entity)
    - [Provider.Definition.Auth.Type](#mediator-v1-Provider-Definition-Auth-Type)
    - [Provider.Definition.ClientTypes.Type](#mediator-v1-Provider-Definition-ClientTypes-Type)
    - [RepoFilter](#mediator-v1-RepoFilter)
  
    - [ArtifactService](#mediator-v1-ArtifactService)
    - [AuthService](#mediator-v1-AuthService)
    - [BranchProtectionService](#mediator-v1-BranchProtectionService)
    - [GroupService](#mediator-v1-GroupService)
    - [HealthService](#mediator-v1-HealthService)
    - [KeyService](#mediator-v1-KeyService)
    - [OAuthService](#mediator-v1-OAuthService)
    - [OrganizationService](#mediator-v1-OrganizationService)
    - [PolicyService](#mediator-v1-PolicyService)
    - [RepositoryService](#mediator-v1-RepositoryService)
    - [RoleService](#mediator-v1-RoleService)
    - [SecretsService](#mediator-v1-SecretsService)
    - [UserService](#mediator-v1-UserService)
    - [VulnerabilitiesService](#mediator-v1-VulnerabilitiesService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="mediator_v1_mediator-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mediator/v1/mediator.proto



<a name="mediator-v1-Artifact"></a>

### Artifact



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact_pk | [int64](#int64) |  |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| type | [string](#string) |  |  |
| visibility | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-ArtifactType"></a>

### ArtifactType
ArtifactType defines the artifact data evaluation.






<a name="mediator-v1-ArtifactVersion"></a>

### ArtifactVersion



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version_id | [int64](#int64) |  |  |
| tags | [string](#string) | repeated |  |
| sha | [string](#string) |  |  |
| signature_verification | [SignatureVerification](#mediator-v1-SignatureVerification) |  |  |
| github_workflow | [GithubWorkflow](#mediator-v1-GithubWorkflow) | optional |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-BranchProtection"></a>

### BranchProtection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [string](#string) |  |  |
| is_protected | [bool](#bool) |  | Add other relevant fields |






<a name="mediator-v1-BuiltinType"></a>

### BuiltinType
BuiltinType defines the builtin data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  |  |






<a name="mediator-v1-CheckHealthRequest"></a>

### CheckHealthRequest







<a name="mediator-v1-CheckHealthResponse"></a>

### CheckHealthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="mediator-v1-Context"></a>

### Context
Context defines the context in which a rule is evaluated.
this normally refers to a combination of the provider, organization and group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| organization | [string](#string) | optional |  |
| group | [string](#string) | optional |  |






<a name="mediator-v1-CreateGroupRequest"></a>

### CreateGroupRequest
The CreateGroupRequest message represents a request to create a group


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| is_protected | [bool](#bool) | optional |  |






<a name="mediator-v1-CreateGroupResponse"></a>

### CreateGroupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-CreateKeyPairRequest"></a>

### CreateKeyPairRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| passphrase | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-CreateKeyPairResponse"></a>

### CreateKeyPairResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key_identifier | [string](#string) |  |  |
| public_key | [string](#string) |  |  |






<a name="mediator-v1-CreateOrganizationRequest"></a>

### CreateOrganizationRequest
Organization service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| company | [string](#string) |  |  |
| create_default_records | [bool](#bool) |  |  |






<a name="mediator-v1-CreateOrganizationResponse"></a>

### CreateOrganizationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| company | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| default_group | [GroupRecord](#mediator-v1-GroupRecord) | optional |  |
| default_roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |
| default_user | [UserRecord](#mediator-v1-UserRecord) | optional |  |






<a name="mediator-v1-CreatePolicyRequest"></a>

### CreatePolicyRequest
Policy service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [PipelinePolicy](#mediator-v1-PipelinePolicy) |  |  |






<a name="mediator-v1-CreatePolicyResponse"></a>

### CreatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [PipelinePolicy](#mediator-v1-PipelinePolicy) |  |  |






<a name="mediator-v1-CreateRoleByGroupRequest"></a>

### CreateRoleByGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| group_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| is_admin | [bool](#bool) | optional |  |
| is_protected | [bool](#bool) | optional |  |






<a name="mediator-v1-CreateRoleByGroupResponse"></a>

### CreateRoleByGroupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| group_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| is_admin | [bool](#bool) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-CreateRoleByOrganizationRequest"></a>

### CreateRoleByOrganizationRequest
Role service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| is_admin | [bool](#bool) | optional |  |
| is_protected | [bool](#bool) | optional |  |






<a name="mediator-v1-CreateRoleByOrganizationResponse"></a>

### CreateRoleByOrganizationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| is_admin | [bool](#bool) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-CreateRuleTypeRequest"></a>

### CreateRuleTypeRequest
CreateRuleTypeRequest is the request to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type to be created. |






<a name="mediator-v1-CreateRuleTypeResponse"></a>

### CreateRuleTypeResponse
CreateRuleTypeResponse is the response to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type that was created. |






<a name="mediator-v1-CreateUserRequest"></a>

### CreateUserRequest
User service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| email | [string](#string) | optional |  |
| username | [string](#string) |  |  |
| password | [string](#string) | optional |  |
| first_name | [string](#string) | optional |  |
| last_name | [string](#string) | optional |  |
| is_protected | [bool](#bool) | optional |  |
| needs_password_change | [bool](#bool) | optional |  |
| group_ids | [int32](#int32) | repeated |  |
| role_ids | [int32](#int32) | repeated |  |






<a name="mediator-v1-CreateUserResponse"></a>

### CreateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| email | [string](#string) | optional |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| first_name | [string](#string) | optional |  |
| last_name | [string](#string) | optional |  |
| is_protected | [bool](#bool) | optional |  |
| needs_password_change | [bool](#bool) | optional |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-DeleteGroupRequest"></a>

### DeleteGroupRequest
DeleteGroupRequest represents a request to delete a group


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| force | [bool](#bool) | optional |  |






<a name="mediator-v1-DeleteGroupResponse"></a>

### DeleteGroupResponse
DeleteGroupResponse represents a response to a delete group request






<a name="mediator-v1-DeleteOrganizationRequest"></a>

### DeleteOrganizationRequest
DeleteOrganizationRequest represents a request to delete a organization


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| force | [bool](#bool) | optional |  |






<a name="mediator-v1-DeleteOrganizationResponse"></a>

### DeleteOrganizationResponse
DeleteOrganizationResponse represents a response to a delete organization request






<a name="mediator-v1-DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [int32](#int32) |  | id is the id of the policy to delete |






<a name="mediator-v1-DeletePolicyResponse"></a>

### DeletePolicyResponse







<a name="mediator-v1-DeleteRoleRequest"></a>

### DeleteRoleRequest
delete role


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| force | [bool](#bool) | optional |  |






<a name="mediator-v1-DeleteRoleResponse"></a>

### DeleteRoleResponse







<a name="mediator-v1-DeleteRuleTypeRequest"></a>

### DeleteRuleTypeRequest
DeleteRuleTypeRequest is the request to delete a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [int32](#int32) |  | id is the id of the rule type to be deleted. |






<a name="mediator-v1-DeleteRuleTypeResponse"></a>

### DeleteRuleTypeResponse
DeleteRuleTypeResponse is the response to delete a rule type.






<a name="mediator-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| force | [bool](#bool) | optional |  |






<a name="mediator-v1-DeleteUserResponse"></a>

### DeleteUserResponse







<a name="mediator-v1-Dependency"></a>

### Dependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystem | [DepEcosystem](#mediator-v1-DepEcosystem) |  |  |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="mediator-v1-DiffType"></a>

### DiffType
DiffType defines the diff data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystems | [DiffType.Ecosystem](#mediator-v1-DiffType-Ecosystem) | repeated |  |






<a name="mediator-v1-DiffType-Ecosystem"></a>

### DiffType.Ecosystem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the ecosystem. |
| depfile | [string](#string) |  | depfile is the file that contains the dependencies for this ecosystem |






<a name="mediator-v1-ExchangeCodeForTokenCLIRequest"></a>

### ExchangeCodeForTokenCLIRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| code | [string](#string) |  |  |
| state | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |






<a name="mediator-v1-ExchangeCodeForTokenWEBRequest"></a>

### ExchangeCodeForTokenWEBRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| code | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |






<a name="mediator-v1-ExchangeCodeForTokenWEBResponse"></a>

### ExchangeCodeForTokenWEBResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| token_type | [string](#string) |  |  |
| expires_in | [int64](#int64) |  |  |
| status | [string](#string) |  |  |






<a name="mediator-v1-FilePatch"></a>

### FilePatch



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | file changed, e.g. package-lock.json |
| patchUrl | [string](#string) |  | points to the the raw patchfile |






<a name="mediator-v1-GetArtifactByIdRequest"></a>

### GetArtifactByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| latest_versions | [int32](#int32) |  |  |
| tag | [string](#string) |  |  |






<a name="mediator-v1-GetArtifactByIdResponse"></a>

### GetArtifactByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [Artifact](#mediator-v1-Artifact) |  |  |
| versions | [ArtifactVersion](#mediator-v1-ArtifactVersion) | repeated |  |






<a name="mediator-v1-GetAuthorizationURLRequest"></a>

### GetAuthorizationURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| cli | [bool](#bool) |  |  |
| port | [int32](#int32) |  |  |
| owner | [string](#string) | optional |  |






<a name="mediator-v1-GetAuthorizationURLResponse"></a>

### GetAuthorizationURLResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  |  |






<a name="mediator-v1-GetBranchProtectionRequest"></a>

### GetBranchProtectionRequest







<a name="mediator-v1-GetBranchProtectionResponse"></a>

### GetBranchProtectionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch_protections | [BranchProtection](#mediator-v1-BranchProtection) | repeated |  |






<a name="mediator-v1-GetGroupByIdRequest"></a>

### GetGroupByIdRequest
The GetGroupByIdRequest message represents a request to get a group by ID


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-GetGroupByIdResponse"></a>

### GetGroupByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [GroupRecord](#mediator-v1-GroupRecord) | optional |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetGroupByNameRequest"></a>

### GetGroupByNameRequest
The GetGroupByNameRequest message represents a request to get a group by name


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="mediator-v1-GetGroupByNameResponse"></a>

### GetGroupByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [GroupRecord](#mediator-v1-GroupRecord) | optional |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetGroupsRequest"></a>

### GetGroupsRequest
The GetGroupsRequest message represents a request to get an array of groups


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| limit | [int32](#int32) |  |  |
| offset | [int32](#int32) |  |  |






<a name="mediator-v1-GetGroupsResponse"></a>

### GetGroupsResponse
The GetGroupsResponse message represents a response with an array of groups


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |






<a name="mediator-v1-GetOrganizationByNameRequest"></a>

### GetOrganizationByNameRequest
get organization by name


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="mediator-v1-GetOrganizationByNameResponse"></a>

### GetOrganizationByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization | [OrganizationRecord](#mediator-v1-OrganizationRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetOrganizationRequest"></a>

### GetOrganizationRequest
get organization by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |






<a name="mediator-v1-GetOrganizationResponse"></a>

### GetOrganizationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization | [OrganizationRecord](#mediator-v1-OrganizationRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetOrganizationsRequest"></a>

### GetOrganizationsRequest
list organizations


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetOrganizationsResponse"></a>

### GetOrganizationsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organizations | [OrganizationRecord](#mediator-v1-OrganizationRecord) | repeated |  |






<a name="mediator-v1-GetPolicyByIdRequest"></a>

### GetPolicyByIdRequest
get policy by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context which contains the policies |
| id | [int32](#int32) |  | id is the id of the policy to get |






<a name="mediator-v1-GetPolicyByIdResponse"></a>

### GetPolicyByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [PipelinePolicy](#mediator-v1-PipelinePolicy) |  |  |






<a name="mediator-v1-GetPolicyStatusByGroupRequest"></a>

### GetPolicyStatusByGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |






<a name="mediator-v1-GetPolicyStatusByGroupResponse"></a>

### GetPolicyStatusByGroupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_status | [PolicyStatus](#mediator-v1-PolicyStatus) | repeated | policy_status is the status of the policy |






<a name="mediator-v1-GetPolicyStatusByIdRequest"></a>

### GetPolicyStatusByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| policy_id | [int32](#int32) |  | policy_id is the id of the policy to get |
| entity | [GetPolicyStatusByIdRequest.EntityTypedId](#mediator-v1-GetPolicyStatusByIdRequest-EntityTypedId) |  |  |
| all | [bool](#bool) |  |  |






<a name="mediator-v1-GetPolicyStatusByIdRequest-EntityTypedId"></a>

### GetPolicyStatusByIdRequest.EntityTypedId
EntiryTypeId is a message that carries an ID together with a type to uniquely identify an entity
such as (repo, 1), (artifact, 2), ...
if the struct is reused in other messages, it should be moved to a top-level definition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Entity](#mediator-v1-Entity) |  | entity is the entity to get status for. Incompatible with `all` |
| id | [int32](#int32) |  | repo_id is the id of the repo to get status for. Incompatible with `all` |






<a name="mediator-v1-GetPolicyStatusByIdResponse"></a>

### GetPolicyStatusByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_status | [PolicyStatus](#mediator-v1-PolicyStatus) |  | policy_status is the status of the policy |
| rule_evaluation_status | [RuleEvaluationStatus](#mediator-v1-RuleEvaluationStatus) | repeated | rule_evaluation_status is the status of the rules |






<a name="mediator-v1-GetPublicKeyRequest"></a>

### GetPublicKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key_identifier | [string](#string) |  |  |






<a name="mediator-v1-GetPublicKeyResponse"></a>

### GetPublicKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| public_key | [string](#string) |  |  |






<a name="mediator-v1-GetRepositoryByIdRequest"></a>

### GetRepositoryByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [int32](#int32) |  |  |






<a name="mediator-v1-GetRepositoryByIdResponse"></a>

### GetRepositoryByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [RepositoryRecord](#mediator-v1-RepositoryRecord) |  |  |






<a name="mediator-v1-GetRepositoryByNameRequest"></a>

### GetRepositoryByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| name | [string](#string) |  |  |






<a name="mediator-v1-GetRepositoryByNameResponse"></a>

### GetRepositoryByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [RepositoryRecord](#mediator-v1-RepositoryRecord) |  |  |






<a name="mediator-v1-GetRoleByIdRequest"></a>

### GetRoleByIdRequest
get role by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |






<a name="mediator-v1-GetRoleByIdResponse"></a>

### GetRoleByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [RoleRecord](#mediator-v1-RoleRecord) | optional |  |






<a name="mediator-v1-GetRoleByNameRequest"></a>

### GetRoleByNameRequest
get role by group and name


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |






<a name="mediator-v1-GetRoleByNameResponse"></a>

### GetRoleByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [RoleRecord](#mediator-v1-RoleRecord) | optional |  |






<a name="mediator-v1-GetRolesByGroupRequest"></a>

### GetRolesByGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_id | [int32](#int32) |  |  |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetRolesByGroupResponse"></a>

### GetRolesByGroupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetRolesRequest"></a>

### GetRolesRequest
list roles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetRolesResponse"></a>

### GetRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetRuleTypeByIdRequest"></a>

### GetRuleTypeByIdRequest
GetRuleTypeByIdRequest is the request to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [int32](#int32) |  | id is the id of the rule type. |






<a name="mediator-v1-GetRuleTypeByIdResponse"></a>

### GetRuleTypeByIdResponse
GetRuleTypeByIdResponse is the response to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type. |






<a name="mediator-v1-GetRuleTypeByNameRequest"></a>

### GetRuleTypeByNameRequest
GetRuleTypeByNameRequest is the request to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| name | [string](#string) |  | name is the name of the rule type. |






<a name="mediator-v1-GetRuleTypeByNameResponse"></a>

### GetRuleTypeByNameResponse
GetRuleTypeByNameResponse is the response to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type. |






<a name="mediator-v1-GetSecretByIdRequest"></a>

### GetSecretByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="mediator-v1-GetSecretByIdResponse"></a>

### GetSecretByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  | Add other relevant fields |






<a name="mediator-v1-GetSecretsRequest"></a>

### GetSecretsRequest







<a name="mediator-v1-GetSecretsResponse"></a>

### GetSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [GetSecretByIdResponse](#mediator-v1-GetSecretByIdResponse) | repeated |  |






<a name="mediator-v1-GetUserByEmailRequest"></a>

### GetUserByEmailRequest
get user by email


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="mediator-v1-GetUserByEmailResponse"></a>

### GetUserByEmailResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#mediator-v1-UserRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetUserByIdRequest"></a>

### GetUserByIdRequest
get user by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |






<a name="mediator-v1-GetUserByIdResponse"></a>

### GetUserByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#mediator-v1-UserRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetUserByUserNameRequest"></a>

### GetUserByUserNameRequest
get user by username


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| username | [string](#string) |  |  |






<a name="mediator-v1-GetUserByUserNameResponse"></a>

### GetUserByUserNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#mediator-v1-UserRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetUserRequest"></a>

### GetUserRequest
get user






<a name="mediator-v1-GetUserResponse"></a>

### GetUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#mediator-v1-UserRecord) | optional |  |
| groups | [GroupRecord](#mediator-v1-GroupRecord) | repeated |  |
| roles | [RoleRecord](#mediator-v1-RoleRecord) | repeated |  |






<a name="mediator-v1-GetUsersByGroupRequest"></a>

### GetUsersByGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_id | [int32](#int32) |  |  |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetUsersByGroupResponse"></a>

### GetUsersByGroupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetUsersByOrganizationRequest"></a>

### GetUsersByOrganizationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization_id | [int32](#int32) |  |  |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetUsersByOrganizationResponse"></a>

### GetUsersByOrganizationResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetUsersRequest"></a>

### GetUsersRequest
list users


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetUsersResponse"></a>

### GetUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [UserRecord](#mediator-v1-UserRecord) | repeated |  |






<a name="mediator-v1-GetVulnerabilitiesRequest"></a>

### GetVulnerabilitiesRequest







<a name="mediator-v1-GetVulnerabilitiesResponse"></a>

### GetVulnerabilitiesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vulns | [GetVulnerabilityByIdResponse](#mediator-v1-GetVulnerabilityByIdResponse) | repeated |  |






<a name="mediator-v1-GetVulnerabilityByIdRequest"></a>

### GetVulnerabilityByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="mediator-v1-GetVulnerabilityByIdResponse"></a>

### GetVulnerabilityByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [uint64](#uint64) |  | May require adjustment, currently set up for GitHub Security Advisories only |
| github_id | [int64](#int64) |  |  |
| repo_id | [int64](#int64) |  |  |
| repo_name | [string](#string) |  |  |
| package_name | [string](#string) |  |  |
| severity | [string](#string) |  |  |
| version_affected | [string](#string) |  |  |
| upgrade_version | [string](#string) |  |  |
| ghsaid | [string](#string) |  |  |
| advisroy_url | [string](#string) |  |  |
| scanned_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-GitType"></a>

### GitType
GitType defines the git data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clone_url | [string](#string) |  | clone_url is the url of the git repository. |
| branch | [string](#string) |  | branch is the branch of the git repository. |






<a name="mediator-v1-GithubWorkflow"></a>

### GithubWorkflow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| commit_sha | [string](#string) |  |  |
| trigger | [string](#string) |  |  |






<a name="mediator-v1-GroupRecord"></a>

### GroupRecord
BUF does not allow grouping (which is a shame)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-ListArtifactsRequest"></a>

### ListArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-ListArtifactsResponse"></a>

### ListArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [Artifact](#mediator-v1-Artifact) | repeated |  |






<a name="mediator-v1-ListPoliciesRequest"></a>

### ListPoliciesRequest
list policies


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context which contains the policies |






<a name="mediator-v1-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [PipelinePolicy](#mediator-v1-PipelinePolicy) | repeated |  |






<a name="mediator-v1-ListRepositoriesRequest"></a>

### ListRepositoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| limit | [int32](#int32) |  |  |
| offset | [int32](#int32) |  |  |
| filter | [RepoFilter](#mediator-v1-RepoFilter) |  |  |






<a name="mediator-v1-ListRepositoriesResponse"></a>

### ListRepositoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [RepositoryRecord](#mediator-v1-RepositoryRecord) | repeated |  |






<a name="mediator-v1-ListRuleTypesRequest"></a>

### ListRuleTypesRequest
ListRuleTypesRequest is the request to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule types are evaluated. |






<a name="mediator-v1-ListRuleTypesResponse"></a>

### ListRuleTypesResponse
ListRuleTypesResponse is the response to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_types | [RuleType](#mediator-v1-RuleType) | repeated | rule_types is the list of rule types. |






<a name="mediator-v1-LogInRequest"></a>

### LogInRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="mediator-v1-LogInResponse"></a>

### LogInResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| refresh_token | [string](#string) |  |  |
| access_token | [string](#string) |  |  |
| refresh_token_expires_in | [int64](#int64) |  |  |
| access_token_expires_in | [int64](#int64) |  |  |






<a name="mediator-v1-LogOutRequest"></a>

### LogOutRequest







<a name="mediator-v1-LogOutResponse"></a>

### LogOutResponse







<a name="mediator-v1-OrganizationRecord"></a>

### OrganizationRecord



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| name | [string](#string) |  |  |
| company | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-PipelinePolicy"></a>

### PipelinePolicy
PipelinePolicy defines a policy that is user defined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the policy is evaluated. |
| id | [int32](#int32) | optional | id is the id of the policy. This is optional and is set by the system. |
| name | [string](#string) |  | name is the name of the policy instance. |
| repository | [PipelinePolicy.ContextualRuleSet](#mediator-v1-PipelinePolicy-ContextualRuleSet) | repeated | These are the entities that one could set in the pipeline policy. |
| build_environment | [PipelinePolicy.ContextualRuleSet](#mediator-v1-PipelinePolicy-ContextualRuleSet) | repeated |  |
| artifact | [PipelinePolicy.ContextualRuleSet](#mediator-v1-PipelinePolicy-ContextualRuleSet) | repeated |  |
| pull_request | [PipelinePolicy.ContextualRuleSet](#mediator-v1-PipelinePolicy-ContextualRuleSet) | repeated |  |






<a name="mediator-v1-PipelinePolicy-ContextualRuleSet"></a>

### PipelinePolicy.ContextualRuleSet
ContextualRuleSet defines a set of rules that are evaluated in a certain context.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [string](#string) | optional | context is the context in which the rules are evaluated. This refers to the provider. |
| rules | [PipelinePolicy.Rule](#mediator-v1-PipelinePolicy-Rule) | repeated | rule is the set of rules that are evaluated. |






<a name="mediator-v1-PipelinePolicy-Rule"></a>

### PipelinePolicy.Rule
Rule defines the individual call of a certain rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the rule to be instantiated. |
| params | [google.protobuf.Struct](#google-protobuf-Struct) |  | params are the parameters that are passed to the rule. This is optional and depends on the rule type. |
| def | [google.protobuf.Struct](#google-protobuf-Struct) |  | def is the definition of the rule. This depends on the rule type. |






<a name="mediator-v1-PolicyStatus"></a>

### PolicyStatus
get the overall policy status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_id | [int32](#int32) |  | policy_id is the id of the policy |
| policy_name | [string](#string) |  | policy_name is the name of the policy |
| policy_status | [string](#string) |  | policy_status is the status of the policy |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the policy was updated |






<a name="mediator-v1-PrDependencies"></a>

### PrDependencies



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pr | [PullRequest](#mediator-v1-PullRequest) |  |  |
| deps | [PrDependencies.ContextualDependency](#mediator-v1-PrDependencies-ContextualDependency) | repeated |  |






<a name="mediator-v1-PrDependencies-ContextualDependency"></a>

### PrDependencies.ContextualDependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dep | [Dependency](#mediator-v1-Dependency) |  |  |
| file | [FilePatch](#mediator-v1-FilePatch) |  |  |






<a name="mediator-v1-Provider"></a>

### Provider
Provider defines a provider that is used to connect to a certain service.
This is used to define the context in which a rule is evaluated and serves
as a data ingestion point. They are top level entities and are scoped to
an organization.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| context | [Provider.Context](#mediator-v1-Provider-Context) |  |  |
| def | [Provider.Definition](#mediator-v1-Provider-Definition) |  |  |






<a name="mediator-v1-Provider-Context"></a>

### Provider.Context
Context defines the context in which a provider is evaluated.
Given thta a provider is a top level entity, it may only be scoped to
an organization.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization | [string](#string) |  |  |
| group | [string](#string) |  |  |






<a name="mediator-v1-Provider-Definition"></a>

### Provider.Definition
Definition defines the definition of the provider.
This is used to define the connection to the provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  |  |
| auth | [Provider.Definition.Auth](#mediator-v1-Provider-Definition-Auth) |  |  |
| client_types | [Provider.Definition.ClientTypes](#mediator-v1-Provider-Definition-ClientTypes) |  | ClientTypes defines the client types that are supported by the provider. |
| entities | [Entity](#mediator-v1-Entity) | repeated | entities defines the entities that are supported by the provider. |






<a name="mediator-v1-Provider-Definition-Auth"></a>

### Provider.Definition.Auth
Auth defines the authentication mechanism to be used.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Provider.Definition.Auth.Type](#mediator-v1-Provider-Definition-Auth-Type) |  |  |
| oauth2 | [Provider.Definition.Auth.OAuth2](#mediator-v1-Provider-Definition-Auth-OAuth2) | optional |  |






<a name="mediator-v1-Provider-Definition-Auth-OAuth2"></a>

### Provider.Definition.Auth.OAuth2
OAuth2 defines the OAuth2 authentication mechanism.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |
| auth_url | [string](#string) |  |  |
| token_url | [string](#string) |  |  |
| scope | [string](#string) |  |  |






<a name="mediator-v1-Provider-Definition-ClientTypes"></a>

### Provider.Definition.ClientTypes
ClientTypes defines the client types that are supported by the provider.
This is used to define the types of clients that are supported by the provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [Provider.Definition.ClientTypes.Type](#mediator-v1-Provider-Definition-ClientTypes-Type) | repeated |  |






<a name="mediator-v1-PullRequest"></a>

### PullRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The full URL to the PR |
| commit_sha | [string](#string) |  | Commit SHA of the PR HEAD. Will be useful to submit a review |
| number | [int32](#int32) |  | The sequential PR number (not the DB PK!) |
| repo_owner | [string](#string) |  | The owner of the repo, will be used to submit a review |
| repo_name | [string](#string) |  | The name of the repo, will be used to submit a review |
| patches | [FilePatch](#mediator-v1-FilePatch) | repeated | The list of files changed in the PR. Does not include file contents |






<a name="mediator-v1-RefreshTokenRequest"></a>

### RefreshTokenRequest







<a name="mediator-v1-RefreshTokenResponse"></a>

### RefreshTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| access_token_expires_in | [int64](#int64) |  |  |






<a name="mediator-v1-RegisterRepositoryRequest"></a>

### RegisterRepositoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| repositories | [Repositories](#mediator-v1-Repositories) | repeated |  |
| events | [string](#string) | repeated |  |






<a name="mediator-v1-RegisterRepositoryResponse"></a>

### RegisterRepositoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [RepositoryResult](#mediator-v1-RepositoryResult) | repeated |  |






<a name="mediator-v1-Repositories"></a>

### Repositories



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |






<a name="mediator-v1-RepositoryRecord"></a>

### RepositoryRecord
RepositoryRecord is used for registering repositories.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |
| is_private | [bool](#bool) |  |  |
| is_fork | [bool](#bool) |  |  |
| hook_url | [string](#string) |  |  |
| deploy_url | [string](#string) |  |  |
| clone_url | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-RepositoryResult"></a>

### RepositoryResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |
| hook_id | [int64](#int64) |  |  |
| hook_url | [string](#string) |  |  |
| deploy_url | [string](#string) |  |  |
| clone_url | [string](#string) |  |  |
| hook_name | [string](#string) |  |  |
| hook_type | [string](#string) |  |  |
| success | [bool](#bool) |  |  |
| uuid | [string](#string) |  |  |
| registered | [bool](#bool) |  |  |
| error | [google.protobuf.StringValue](#google-protobuf-StringValue) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-RestType"></a>

### RestType
RestType defines the rest data evaluation.
This is used to fetch data from a REST endpoint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | endpoint is the endpoint to fetch data from. This can be a URL or the path on the API.bool This is a required field and must be set. This is also evaluated via a template which allows us dynamically fill in the values. |
| method | [string](#string) |  | method is the method to use to fetch data. |
| headers | [string](#string) | repeated | headers are the headers to be sent to the endpoint. |
| body | [string](#string) | optional | body is the body to be sent to the endpoint. |
| parse | [string](#string) |  | parse is the parsing mechanism to be used to parse the data. |






<a name="mediator-v1-RevokeOauthGroupTokenRequest"></a>

### RevokeOauthGroupTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-RevokeOauthGroupTokenResponse"></a>

### RevokeOauthGroupTokenResponse







<a name="mediator-v1-RevokeOauthTokensRequest"></a>

### RevokeOauthTokensRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |






<a name="mediator-v1-RevokeOauthTokensResponse"></a>

### RevokeOauthTokensResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revoked_tokens | [int32](#int32) |  |  |






<a name="mediator-v1-RevokeTokensRequest"></a>

### RevokeTokensRequest







<a name="mediator-v1-RevokeTokensResponse"></a>

### RevokeTokensResponse







<a name="mediator-v1-RevokeUserTokenRequest"></a>

### RevokeUserTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [int32](#int32) |  |  |






<a name="mediator-v1-RevokeUserTokenResponse"></a>

### RevokeUserTokenResponse







<a name="mediator-v1-RoleRecord"></a>

### RoleRecord



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| group_id | [int32](#int32) | optional |  |
| name | [string](#string) |  |  |
| is_admin | [bool](#bool) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-RuleEvaluationStatus"></a>

### RuleEvaluationStatus
get the status of the rules for a given policy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_id | [int32](#int32) |  | policy_id is the id of the policy |
| rule_id | [int32](#int32) |  | rule_id is the id of the rule |
| rule_name | [string](#string) |  | rule_name is the name of the rule |
| entity | [string](#string) |  | entity is the entity that was evaluated |
| status | [string](#string) |  | status is the status of the evaluation |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the policy was updated |
| entity_info | [RuleEvaluationStatus.EntityInfoEntry](#mediator-v1-RuleEvaluationStatus-EntityInfoEntry) | repeated | entity_info is the information about the entity |
| details | [string](#string) |  | details is the description of the evaluation if any |
| guidance | [string](#string) |  | guidance is the guidance for the evaluation if any |






<a name="mediator-v1-RuleEvaluationStatus-EntityInfoEntry"></a>

### RuleEvaluationStatus.EntityInfoEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="mediator-v1-RuleType"></a>

### RuleType
RuleType defines rules that may or may not be user defined.
The version is assumed from the folder&#39;s version.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) | optional | id is the id of the rule type. This is mostly optional and is set by the server. |
| name | [string](#string) |  | name is the name of the rule type. |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule is evaluated. |
| def | [RuleType.Definition](#mediator-v1-RuleType-Definition) |  | def is the definition of the rule type. |
| description | [string](#string) |  | description is the description of the rule type. |
| guidance | [string](#string) |  | guidance are instructions we give the user in case a rule fails. |






<a name="mediator-v1-RuleType-Definition"></a>

### RuleType.Definition
Definition defines the rule type. It encompases the schema and the data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| in_entity | [string](#string) |  | in_entity is the entity in which the rule is evaluated. This can be repository, build_environment or artifact. |
| rule_schema | [google.protobuf.Struct](#google-protobuf-Struct) |  | rule_schema is the schema of the rule. This is expressed in JSON Schema. |
| param_schema | [google.protobuf.Struct](#google-protobuf-Struct) | optional | param_schema is the schema of the parameters that are passed to the rule. This is expressed in JSON Schema. |
| ingest | [RuleType.Definition.Ingest](#mediator-v1-RuleType-Definition-Ingest) |  |  |
| eval | [RuleType.Definition.Eval](#mediator-v1-RuleType-Definition-Eval) |  |  |






<a name="mediator-v1-RuleType-Definition-Eval"></a>

### RuleType.Definition.Eval
Eval defines the data evaluation definition.
This pertains to the way we traverse data from the upstream
endpoint and how we compare it to the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data evaluation. Right now only `jq` is supported as a driver |
| jq | [RuleType.Definition.Eval.JQComparison](#mediator-v1-RuleType-Definition-Eval-JQComparison) | repeated | jq is only used if the `jq` type is selected. It defines the comparisons that are made between the ingested data and the policy rule. |
| rego | [RuleType.Definition.Eval.Rego](#mediator-v1-RuleType-Definition-Eval-Rego) | optional | rego is only used if the `rego` type is selected. |






<a name="mediator-v1-RuleType-Definition-Eval-JQComparison"></a>

### RuleType.Definition.Eval.JQComparison



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ingested | [RuleType.Definition.Eval.JQComparison.Operator](#mediator-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Ingested points to the data retrieved in the `ingest` section |
| policy | [RuleType.Definition.Eval.JQComparison.Operator](#mediator-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Policy points to the policy itself. |






<a name="mediator-v1-RuleType-Definition-Eval-JQComparison-Operator"></a>

### RuleType.Definition.Eval.JQComparison.Operator



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| def | [string](#string) |  |  |






<a name="mediator-v1-RuleType-Definition-Eval-Rego"></a>

### RuleType.Definition.Eval.Rego



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of evaluation engine to use for rego. We currently have two modes of operation: - deny-by-default: this is the default mode of operation where we deny access by default and allow access only if the policy explicitly allows it. It expects the policy to set an `allow` variable to true or false. - constraints: this is the mode of operation where we allow access by default and deny access only if a violation is found. It expects the policy to set a `violations` variable with a &#34;msg&#34; field. |
| def | [string](#string) |  | def is the definition of the rego policy. |






<a name="mediator-v1-RuleType-Definition-Ingest"></a>

### RuleType.Definition.Ingest
Ingest defines how the data is ingested.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data ingestion. we currently support rest, artifact and builtin. |
| rest | [RestType](#mediator-v1-RestType) | optional | rest is the rest data ingestion. this is only used if the type is rest. |
| builtin | [BuiltinType](#mediator-v1-BuiltinType) | optional | builtin is the builtin data ingestion. |
| artifact | [ArtifactType](#mediator-v1-ArtifactType) | optional | artifact is the artifact data ingestion. |
| git | [GitType](#mediator-v1-GitType) | optional | git is the git data ingestion. |
| diff | [DiffType](#mediator-v1-DiffType) | optional | diff is the diff data ingestion. |






<a name="mediator-v1-SignatureVerification"></a>

### SignatureVerification



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| is_signed | [bool](#bool) |  |  |
| is_verified | [bool](#bool) |  |  |
| is_bundle_verified | [bool](#bool) |  |  |
| cert_identity | [string](#string) | optional |  |
| cert_issuer | [string](#string) | optional |  |
| rekor_log_id | [string](#string) | optional |  |
| rekor_log_index | [int32](#int32) | optional |  |
| signature_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |






<a name="mediator-v1-StoreProviderTokenRequest"></a>

### StoreProviderTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| access_token | [string](#string) |  |  |
| owner | [string](#string) | optional |  |






<a name="mediator-v1-StoreProviderTokenResponse"></a>

### StoreProviderTokenResponse







<a name="mediator-v1-SyncRepositoriesRequest"></a>

### SyncRepositoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-SyncRepositoriesResponse"></a>

### SyncRepositoriesResponse







<a name="mediator-v1-UpdatePasswordRequest"></a>

### UpdatePasswordRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password | [string](#string) |  |  |
| password_confirmation | [string](#string) |  |  |






<a name="mediator-v1-UpdatePasswordResponse"></a>

### UpdatePasswordResponse







<a name="mediator-v1-UpdateProfileRequest"></a>

### UpdateProfileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) | optional |  |
| first_name | [string](#string) | optional |  |
| last_name | [string](#string) | optional |  |






<a name="mediator-v1-UpdateProfileResponse"></a>

### UpdateProfileResponse







<a name="mediator-v1-UpdateRuleTypeRequest"></a>

### UpdateRuleTypeRequest
UpdateRuleTypeRequest is the request to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type to be updated. |






<a name="mediator-v1-UpdateRuleTypeResponse"></a>

### UpdateRuleTypeResponse
UpdateRuleTypeResponse is the response to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type that was updated. |






<a name="mediator-v1-UserRecord"></a>

### UserRecord
user record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [int32](#int32) |  |  |
| email | [string](#string) | optional |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| first_name | [string](#string) | optional |  |
| last_name | [string](#string) | optional |  |
| is_protected | [bool](#bool) | optional |  |
| needs_password_change | [bool](#bool) | optional |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-VerifyProviderTokenFromRequest"></a>

### VerifyProviderTokenFromRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-VerifyProviderTokenFromResponse"></a>

### VerifyProviderTokenFromResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="mediator-v1-VerifyRequest"></a>

### VerifyRequest







<a name="mediator-v1-VerifyResponse"></a>

### VerifyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






<a name="mediator-v1-VersionedArtifact"></a>

### VersionedArtifact



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [Artifact](#mediator-v1-Artifact) |  |  |
| version | [ArtifactVersion](#mediator-v1-ArtifactVersion) |  |  |





 


<a name="mediator-v1-DepEcosystem"></a>

### DepEcosystem


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEP_ECOSYSTEM_UNKNOWN | 0 |  |
| DEP_ECOSYSTEM_NPM | 1 |  |



<a name="mediator-v1-Entity"></a>

### Entity
Entity defines the entity that is supported by the provider.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITY_UNSPECIFIED | 0 |  |
| ENTITY_REPOSITORIES | 1 |  |
| ENTITY_BUILD_ENVIRONMENTS | 2 |  |
| ENTITY_ARTIFACTS | 3 |  |
| ENTITY_PULL_REQUESTS | 4 |  |



<a name="mediator-v1-Provider-Definition-Auth-Type"></a>

### Provider.Definition.Auth.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_OAUTH2 | 1 | TYPE_OAUTH2 defines the OAuth2 authentication mechanism.

These are left as references of what we could implement BASIC = 2; TOKEN = 3; |



<a name="mediator-v1-Provider-Definition-ClientTypes-Type"></a>

### Provider.Definition.ClientTypes.Type
Type defines the type of client that is supported by the provider.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_REST | 1 |  |
| TYPE_GRAPHQL | 2 |  |



<a name="mediator-v1-RepoFilter"></a>

### RepoFilter
Repo filter enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| REPO_FILTER_SHOW_UNSPECIFIED | 0 |  |
| REPO_FILTER_SHOW_ALL | 1 |  |
| REPO_FILTER_SHOW_NOT_REGISTERED_ONLY | 2 |  |
| REPO_FILTER_SHOW_REGISTERED_ONLY | 3 |  |


 

 


<a name="mediator-v1-ArtifactService"></a>

### ArtifactService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListArtifacts | [ListArtifactsRequest](#mediator-v1-ListArtifactsRequest) | [ListArtifactsResponse](#mediator-v1-ListArtifactsResponse) |  |
| GetArtifactById | [GetArtifactByIdRequest](#mediator-v1-GetArtifactByIdRequest) | [GetArtifactByIdResponse](#mediator-v1-GetArtifactByIdResponse) |  |


<a name="mediator-v1-AuthService"></a>

### AuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| LogIn | [LogInRequest](#mediator-v1-LogInRequest) | [LogInResponse](#mediator-v1-LogInResponse) | LogIn to Mediator |
| LogOut | [LogOutRequest](#mediator-v1-LogOutRequest) | [LogOutResponse](#mediator-v1-LogOutResponse) | Logout of Mediator |
| RevokeTokens | [RevokeTokensRequest](#mediator-v1-RevokeTokensRequest) | [RevokeTokensResponse](#mediator-v1-RevokeTokensResponse) | revoke all tokens for all users |
| RevokeUserToken | [RevokeUserTokenRequest](#mediator-v1-RevokeUserTokenRequest) | [RevokeUserTokenResponse](#mediator-v1-RevokeUserTokenResponse) | revoke token for an user |
| RefreshToken | [RefreshTokenRequest](#mediator-v1-RefreshTokenRequest) | [RefreshTokenResponse](#mediator-v1-RefreshTokenResponse) | refresh a token |
| Verify | [VerifyRequest](#mediator-v1-VerifyRequest) | [VerifyResponse](#mediator-v1-VerifyResponse) | Verify user has active session to Mediator |


<a name="mediator-v1-BranchProtectionService"></a>

### BranchProtectionService
Get Branch Protection Settings

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetBranchProtection | [GetBranchProtectionRequest](#mediator-v1-GetBranchProtectionRequest) | [GetBranchProtectionResponse](#mediator-v1-GetBranchProtectionResponse) |  |


<a name="mediator-v1-GroupService"></a>

### GroupService
manage Groups CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateGroup | [CreateGroupRequest](#mediator-v1-CreateGroupRequest) | [CreateGroupResponse](#mediator-v1-CreateGroupResponse) |  |
| GetGroups | [GetGroupsRequest](#mediator-v1-GetGroupsRequest) | [GetGroupsResponse](#mediator-v1-GetGroupsResponse) |  |
| GetGroupByName | [GetGroupByNameRequest](#mediator-v1-GetGroupByNameRequest) | [GetGroupByNameResponse](#mediator-v1-GetGroupByNameResponse) |  |
| GetGroupById | [GetGroupByIdRequest](#mediator-v1-GetGroupByIdRequest) | [GetGroupByIdResponse](#mediator-v1-GetGroupByIdResponse) |  |
| DeleteGroup | [DeleteGroupRequest](#mediator-v1-DeleteGroupRequest) | [DeleteGroupResponse](#mediator-v1-DeleteGroupResponse) |  |


<a name="mediator-v1-HealthService"></a>

### HealthService
Simple Health Check Service
replies with OK

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CheckHealth | [CheckHealthRequest](#mediator-v1-CheckHealthRequest) | [CheckHealthResponse](#mediator-v1-CheckHealthResponse) |  |


<a name="mediator-v1-KeyService"></a>

### KeyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPublicKey | [GetPublicKeyRequest](#mediator-v1-GetPublicKeyRequest) | [GetPublicKeyResponse](#mediator-v1-GetPublicKeyResponse) |  |
| CreateKeyPair | [CreateKeyPairRequest](#mediator-v1-CreateKeyPairRequest) | [CreateKeyPairResponse](#mediator-v1-CreateKeyPairResponse) |  |


<a name="mediator-v1-OAuthService"></a>

### OAuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetAuthorizationURL | [GetAuthorizationURLRequest](#mediator-v1-GetAuthorizationURLRequest) | [GetAuthorizationURLResponse](#mediator-v1-GetAuthorizationURLResponse) |  |
| ExchangeCodeForTokenCLI | [ExchangeCodeForTokenCLIRequest](#mediator-v1-ExchangeCodeForTokenCLIRequest) | [.google.api.HttpBody](#google-api-HttpBody) | buf:lint:ignore RPC_RESPONSE_STANDARD_NAME

protolint:disable:this |
| ExchangeCodeForTokenWEB | [ExchangeCodeForTokenWEBRequest](#mediator-v1-ExchangeCodeForTokenWEBRequest) | [ExchangeCodeForTokenWEBResponse](#mediator-v1-ExchangeCodeForTokenWEBResponse) |  |
| StoreProviderToken | [StoreProviderTokenRequest](#mediator-v1-StoreProviderTokenRequest) | [StoreProviderTokenResponse](#mediator-v1-StoreProviderTokenResponse) |  |
| RevokeOauthTokens | [RevokeOauthTokensRequest](#mediator-v1-RevokeOauthTokensRequest) | [RevokeOauthTokensResponse](#mediator-v1-RevokeOauthTokensResponse) | revoke all tokens for all users |
| RevokeOauthGroupToken | [RevokeOauthGroupTokenRequest](#mediator-v1-RevokeOauthGroupTokenRequest) | [RevokeOauthGroupTokenResponse](#mediator-v1-RevokeOauthGroupTokenResponse) | revoke token for a group |
| VerifyProviderTokenFrom | [VerifyProviderTokenFromRequest](#mediator-v1-VerifyProviderTokenFromRequest) | [VerifyProviderTokenFromResponse](#mediator-v1-VerifyProviderTokenFromResponse) | VerifyProviderTokenFrom verifies that a token has been created for a provider since given timestamp |


<a name="mediator-v1-OrganizationService"></a>

### OrganizationService
manage Organizations CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateOrganization | [CreateOrganizationRequest](#mediator-v1-CreateOrganizationRequest) | [CreateOrganizationResponse](#mediator-v1-CreateOrganizationResponse) |  |
| GetOrganizations | [GetOrganizationsRequest](#mediator-v1-GetOrganizationsRequest) | [GetOrganizationsResponse](#mediator-v1-GetOrganizationsResponse) |  |
| GetOrganization | [GetOrganizationRequest](#mediator-v1-GetOrganizationRequest) | [GetOrganizationResponse](#mediator-v1-GetOrganizationResponse) |  |
| GetOrganizationByName | [GetOrganizationByNameRequest](#mediator-v1-GetOrganizationByNameRequest) | [GetOrganizationByNameResponse](#mediator-v1-GetOrganizationByNameResponse) |  |
| DeleteOrganization | [DeleteOrganizationRequest](#mediator-v1-DeleteOrganizationRequest) | [DeleteOrganizationResponse](#mediator-v1-DeleteOrganizationResponse) |  |


<a name="mediator-v1-PolicyService"></a>

### PolicyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreatePolicy | [CreatePolicyRequest](#mediator-v1-CreatePolicyRequest) | [CreatePolicyResponse](#mediator-v1-CreatePolicyResponse) |  |
| DeletePolicy | [DeletePolicyRequest](#mediator-v1-DeletePolicyRequest) | [DeletePolicyResponse](#mediator-v1-DeletePolicyResponse) |  |
| ListPolicies | [ListPoliciesRequest](#mediator-v1-ListPoliciesRequest) | [ListPoliciesResponse](#mediator-v1-ListPoliciesResponse) |  |
| GetPolicyById | [GetPolicyByIdRequest](#mediator-v1-GetPolicyByIdRequest) | [GetPolicyByIdResponse](#mediator-v1-GetPolicyByIdResponse) |  |
| GetPolicyStatusById | [GetPolicyStatusByIdRequest](#mediator-v1-GetPolicyStatusByIdRequest) | [GetPolicyStatusByIdResponse](#mediator-v1-GetPolicyStatusByIdResponse) |  |
| GetPolicyStatusByGroup | [GetPolicyStatusByGroupRequest](#mediator-v1-GetPolicyStatusByGroupRequest) | [GetPolicyStatusByGroupResponse](#mediator-v1-GetPolicyStatusByGroupResponse) |  |
| ListRuleTypes | [ListRuleTypesRequest](#mediator-v1-ListRuleTypesRequest) | [ListRuleTypesResponse](#mediator-v1-ListRuleTypesResponse) |  |
| GetRuleTypeByName | [GetRuleTypeByNameRequest](#mediator-v1-GetRuleTypeByNameRequest) | [GetRuleTypeByNameResponse](#mediator-v1-GetRuleTypeByNameResponse) |  |
| GetRuleTypeById | [GetRuleTypeByIdRequest](#mediator-v1-GetRuleTypeByIdRequest) | [GetRuleTypeByIdResponse](#mediator-v1-GetRuleTypeByIdResponse) |  |
| CreateRuleType | [CreateRuleTypeRequest](#mediator-v1-CreateRuleTypeRequest) | [CreateRuleTypeResponse](#mediator-v1-CreateRuleTypeResponse) |  |
| UpdateRuleType | [UpdateRuleTypeRequest](#mediator-v1-UpdateRuleTypeRequest) | [UpdateRuleTypeResponse](#mediator-v1-UpdateRuleTypeResponse) |  |
| DeleteRuleType | [DeleteRuleTypeRequest](#mediator-v1-DeleteRuleTypeRequest) | [DeleteRuleTypeResponse](#mediator-v1-DeleteRuleTypeResponse) |  |


<a name="mediator-v1-RepositoryService"></a>

### RepositoryService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SyncRepositories | [SyncRepositoriesRequest](#mediator-v1-SyncRepositoriesRequest) | [SyncRepositoriesResponse](#mediator-v1-SyncRepositoriesResponse) |  |
| RegisterRepository | [RegisterRepositoryRequest](#mediator-v1-RegisterRepositoryRequest) | [RegisterRepositoryResponse](#mediator-v1-RegisterRepositoryResponse) |  |
| ListRepositories | [ListRepositoriesRequest](#mediator-v1-ListRepositoriesRequest) | [ListRepositoriesResponse](#mediator-v1-ListRepositoriesResponse) |  |
| GetRepositoryById | [GetRepositoryByIdRequest](#mediator-v1-GetRepositoryByIdRequest) | [GetRepositoryByIdResponse](#mediator-v1-GetRepositoryByIdResponse) |  |
| GetRepositoryByName | [GetRepositoryByNameRequest](#mediator-v1-GetRepositoryByNameRequest) | [GetRepositoryByNameResponse](#mediator-v1-GetRepositoryByNameResponse) |  |


<a name="mediator-v1-RoleService"></a>

### RoleService
manage Roles CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateRoleByOrganization | [CreateRoleByOrganizationRequest](#mediator-v1-CreateRoleByOrganizationRequest) | [CreateRoleByOrganizationResponse](#mediator-v1-CreateRoleByOrganizationResponse) |  |
| CreateRoleByGroup | [CreateRoleByGroupRequest](#mediator-v1-CreateRoleByGroupRequest) | [CreateRoleByGroupResponse](#mediator-v1-CreateRoleByGroupResponse) |  |
| DeleteRole | [DeleteRoleRequest](#mediator-v1-DeleteRoleRequest) | [DeleteRoleResponse](#mediator-v1-DeleteRoleResponse) |  |
| GetRoles | [GetRolesRequest](#mediator-v1-GetRolesRequest) | [GetRolesResponse](#mediator-v1-GetRolesResponse) |  |
| GetRolesByGroup | [GetRolesByGroupRequest](#mediator-v1-GetRolesByGroupRequest) | [GetRolesByGroupResponse](#mediator-v1-GetRolesByGroupResponse) |  |
| GetRoleById | [GetRoleByIdRequest](#mediator-v1-GetRoleByIdRequest) | [GetRoleByIdResponse](#mediator-v1-GetRoleByIdResponse) |  |
| GetRoleByName | [GetRoleByNameRequest](#mediator-v1-GetRoleByNameRequest) | [GetRoleByNameResponse](#mediator-v1-GetRoleByNameResponse) |  |


<a name="mediator-v1-SecretsService"></a>

### SecretsService
Get Secrets
Note there are different APIs for enterprise or org secrets
https://docs.github.com/en/rest/secret-scanning?apiVersion=2022-11-28

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSecrets | [GetSecretsRequest](#mediator-v1-GetSecretsRequest) | [GetSecretsResponse](#mediator-v1-GetSecretsResponse) |  |
| GetSecretById | [GetSecretByIdRequest](#mediator-v1-GetSecretByIdRequest) | [GetSecretByIdResponse](#mediator-v1-GetSecretByIdResponse) |  |


<a name="mediator-v1-UserService"></a>

### UserService
manage Users CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#mediator-v1-CreateUserRequest) | [CreateUserResponse](#mediator-v1-CreateUserResponse) |  |
| DeleteUser | [DeleteUserRequest](#mediator-v1-DeleteUserRequest) | [DeleteUserResponse](#mediator-v1-DeleteUserResponse) |  |
| GetUsers | [GetUsersRequest](#mediator-v1-GetUsersRequest) | [GetUsersResponse](#mediator-v1-GetUsersResponse) |  |
| GetUsersByOrganization | [GetUsersByOrganizationRequest](#mediator-v1-GetUsersByOrganizationRequest) | [GetUsersByOrganizationResponse](#mediator-v1-GetUsersByOrganizationResponse) |  |
| GetUsersByGroup | [GetUsersByGroupRequest](#mediator-v1-GetUsersByGroupRequest) | [GetUsersByGroupResponse](#mediator-v1-GetUsersByGroupResponse) |  |
| GetUserById | [GetUserByIdRequest](#mediator-v1-GetUserByIdRequest) | [GetUserByIdResponse](#mediator-v1-GetUserByIdResponse) |  |
| GetUserByUserName | [GetUserByUserNameRequest](#mediator-v1-GetUserByUserNameRequest) | [GetUserByUserNameResponse](#mediator-v1-GetUserByUserNameResponse) |  |
| GetUser | [GetUserRequest](#mediator-v1-GetUserRequest) | [GetUserResponse](#mediator-v1-GetUserResponse) |  |
| GetUserByEmail | [GetUserByEmailRequest](#mediator-v1-GetUserByEmailRequest) | [GetUserByEmailResponse](#mediator-v1-GetUserByEmailResponse) |  |
| UpdatePassword | [UpdatePasswordRequest](#mediator-v1-UpdatePasswordRequest) | [UpdatePasswordResponse](#mediator-v1-UpdatePasswordResponse) |  |
| UpdateProfile | [UpdateProfileRequest](#mediator-v1-UpdateProfileRequest) | [UpdateProfileResponse](#mediator-v1-UpdateProfileResponse) |  |


<a name="mediator-v1-VulnerabilitiesService"></a>

### VulnerabilitiesService
Get Vulnerabilities

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetVulnerabilities | [GetVulnerabilitiesRequest](#mediator-v1-GetVulnerabilitiesRequest) | [GetVulnerabilitiesResponse](#mediator-v1-GetVulnerabilitiesResponse) |  |
| GetVulnerabilityById | [GetVulnerabilityByIdRequest](#mediator-v1-GetVulnerabilityByIdRequest) | [GetVulnerabilityByIdResponse](#mediator-v1-GetVulnerabilityByIdResponse) |  |

 



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

