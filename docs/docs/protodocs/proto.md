# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [mediator/v1/mediator.proto](#mediator_v1_mediator-proto)
    - [BranchProtection](#mediator-v1-BranchProtection)
    - [CheckHealthRequest](#mediator-v1-CheckHealthRequest)
    - [CheckHealthResponse](#mediator-v1-CheckHealthResponse)
    - [CreateGroupRequest](#mediator-v1-CreateGroupRequest)
    - [CreateGroupResponse](#mediator-v1-CreateGroupResponse)
    - [CreateOrganizationRequest](#mediator-v1-CreateOrganizationRequest)
    - [CreateOrganizationResponse](#mediator-v1-CreateOrganizationResponse)
    - [CreatePolicyRequest](#mediator-v1-CreatePolicyRequest)
    - [CreatePolicyResponse](#mediator-v1-CreatePolicyResponse)
    - [CreateRoleByGroupRequest](#mediator-v1-CreateRoleByGroupRequest)
    - [CreateRoleByGroupResponse](#mediator-v1-CreateRoleByGroupResponse)
    - [CreateRoleByOrganizationRequest](#mediator-v1-CreateRoleByOrganizationRequest)
    - [CreateRoleByOrganizationResponse](#mediator-v1-CreateRoleByOrganizationResponse)
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
    - [DeleteUserRequest](#mediator-v1-DeleteUserRequest)
    - [DeleteUserResponse](#mediator-v1-DeleteUserResponse)
    - [ExchangeCodeForTokenCLIRequest](#mediator-v1-ExchangeCodeForTokenCLIRequest)
    - [ExchangeCodeForTokenCLIResponse](#mediator-v1-ExchangeCodeForTokenCLIResponse)
    - [ExchangeCodeForTokenWEBRequest](#mediator-v1-ExchangeCodeForTokenWEBRequest)
    - [ExchangeCodeForTokenWEBResponse](#mediator-v1-ExchangeCodeForTokenWEBResponse)
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
    - [GetPoliciesRequest](#mediator-v1-GetPoliciesRequest)
    - [GetPoliciesResponse](#mediator-v1-GetPoliciesResponse)
    - [GetPolicyByIdRequest](#mediator-v1-GetPolicyByIdRequest)
    - [GetPolicyByIdResponse](#mediator-v1-GetPolicyByIdResponse)
    - [GetPolicyTypeByIdRequest](#mediator-v1-GetPolicyTypeByIdRequest)
    - [GetPolicyTypeByIdResponse](#mediator-v1-GetPolicyTypeByIdResponse)
    - [GetPolicyTypeRequest](#mediator-v1-GetPolicyTypeRequest)
    - [GetPolicyTypeResponse](#mediator-v1-GetPolicyTypeResponse)
    - [GetPolicyTypesRequest](#mediator-v1-GetPolicyTypesRequest)
    - [GetPolicyTypesResponse](#mediator-v1-GetPolicyTypesResponse)
    - [GetRepositoryRequest](#mediator-v1-GetRepositoryRequest)
    - [GetRepositoryResponse](#mediator-v1-GetRepositoryResponse)
    - [GetRoleByIdRequest](#mediator-v1-GetRoleByIdRequest)
    - [GetRoleByIdResponse](#mediator-v1-GetRoleByIdResponse)
    - [GetRoleByNameRequest](#mediator-v1-GetRoleByNameRequest)
    - [GetRoleByNameResponse](#mediator-v1-GetRoleByNameResponse)
    - [GetRolesByGroupRequest](#mediator-v1-GetRolesByGroupRequest)
    - [GetRolesByGroupResponse](#mediator-v1-GetRolesByGroupResponse)
    - [GetRolesRequest](#mediator-v1-GetRolesRequest)
    - [GetRolesResponse](#mediator-v1-GetRolesResponse)
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
    - [GroupRecord](#mediator-v1-GroupRecord)
    - [ListRepositories](#mediator-v1-ListRepositories)
    - [ListRepositoriesRequest](#mediator-v1-ListRepositoriesRequest)
    - [ListRepositoriesResponse](#mediator-v1-ListRepositoriesResponse)
    - [LogInRequest](#mediator-v1-LogInRequest)
    - [LogInResponse](#mediator-v1-LogInResponse)
    - [LogOutRequest](#mediator-v1-LogOutRequest)
    - [LogOutResponse](#mediator-v1-LogOutResponse)
    - [OrganizationRecord](#mediator-v1-OrganizationRecord)
    - [PolicyRecord](#mediator-v1-PolicyRecord)
    - [PolicyTypeRecord](#mediator-v1-PolicyTypeRecord)
    - [RefreshTokenRequest](#mediator-v1-RefreshTokenRequest)
    - [RefreshTokenResponse](#mediator-v1-RefreshTokenResponse)
    - [RegisterRepositoryRequest](#mediator-v1-RegisterRepositoryRequest)
    - [RegisterRepositoryResponse](#mediator-v1-RegisterRepositoryResponse)
    - [Repositories](#mediator-v1-Repositories)
    - [RepositoryResult](#mediator-v1-RepositoryResult)
    - [RevokeOauthGroupTokenRequest](#mediator-v1-RevokeOauthGroupTokenRequest)
    - [RevokeOauthGroupTokenResponse](#mediator-v1-RevokeOauthGroupTokenResponse)
    - [RevokeOauthTokensRequest](#mediator-v1-RevokeOauthTokensRequest)
    - [RevokeOauthTokensResponse](#mediator-v1-RevokeOauthTokensResponse)
    - [RevokeTokensRequest](#mediator-v1-RevokeTokensRequest)
    - [RevokeTokensResponse](#mediator-v1-RevokeTokensResponse)
    - [RevokeUserTokenRequest](#mediator-v1-RevokeUserTokenRequest)
    - [RevokeUserTokenResponse](#mediator-v1-RevokeUserTokenResponse)
    - [RoleRecord](#mediator-v1-RoleRecord)
    - [StoreProviderTokenRequest](#mediator-v1-StoreProviderTokenRequest)
    - [StoreProviderTokenResponse](#mediator-v1-StoreProviderTokenResponse)
    - [UpdatePasswordRequest](#mediator-v1-UpdatePasswordRequest)
    - [UpdatePasswordResponse](#mediator-v1-UpdatePasswordResponse)
    - [UpdateProfileRequest](#mediator-v1-UpdateProfileRequest)
    - [UpdateProfileResponse](#mediator-v1-UpdateProfileResponse)
    - [UserRecord](#mediator-v1-UserRecord)
    - [VerifyProviderTokenFromRequest](#mediator-v1-VerifyProviderTokenFromRequest)
    - [VerifyProviderTokenFromResponse](#mediator-v1-VerifyProviderTokenFromResponse)
    - [VerifyRequest](#mediator-v1-VerifyRequest)
    - [VerifyResponse](#mediator-v1-VerifyResponse)
  
    - [RepoFilter](#mediator-v1-RepoFilter)
  
    - [AuthService](#mediator-v1-AuthService)
    - [BranchProtectionService](#mediator-v1-BranchProtectionService)
    - [GroupService](#mediator-v1-GroupService)
    - [HealthService](#mediator-v1-HealthService)
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



<a name="mediator-v1-BranchProtection"></a>

### BranchProtection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [string](#string) |  |  |
| is_protected | [bool](#bool) |  | Add other relevant fields |






<a name="mediator-v1-CheckHealthRequest"></a>

### CheckHealthRequest







<a name="mediator-v1-CheckHealthResponse"></a>

### CheckHealthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |






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
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| type | [string](#string) |  |  |
| policy_definition | [string](#string) |  |  |






<a name="mediator-v1-CreatePolicyResponse"></a>

### CreatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [PolicyRecord](#mediator-v1-PolicyRecord) |  |  |






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
| id | [int32](#int32) |  |  |






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







<a name="mediator-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| force | [bool](#bool) | optional |  |






<a name="mediator-v1-DeleteUserResponse"></a>

### DeleteUserResponse







<a name="mediator-v1-ExchangeCodeForTokenCLIRequest"></a>

### ExchangeCodeForTokenCLIRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| code | [string](#string) |  |  |
| state | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |






<a name="mediator-v1-ExchangeCodeForTokenCLIResponse"></a>

### ExchangeCodeForTokenCLIResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| html | [string](#string) |  |  |






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






<a name="mediator-v1-GetAuthorizationURLRequest"></a>

### GetAuthorizationURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| cli | [bool](#bool) |  |  |
| port | [int32](#int32) |  |  |






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






<a name="mediator-v1-GetPoliciesRequest"></a>

### GetPoliciesRequest
list policies


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| limit | [int32](#int32) | optional |  |
| offset | [int32](#int32) | optional |  |






<a name="mediator-v1-GetPoliciesResponse"></a>

### GetPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [PolicyRecord](#mediator-v1-PolicyRecord) | repeated |  |






<a name="mediator-v1-GetPolicyByIdRequest"></a>

### GetPolicyByIdRequest
get policy by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |






<a name="mediator-v1-GetPolicyByIdResponse"></a>

### GetPolicyByIdResponse
in the future it can include status and violation details for the policy


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [PolicyRecord](#mediator-v1-PolicyRecord) | optional |  |






<a name="mediator-v1-GetPolicyTypeByIdRequest"></a>

### GetPolicyTypeByIdRequest
get policy type by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="mediator-v1-GetPolicyTypeByIdResponse"></a>

### GetPolicyTypeByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_type | [PolicyTypeRecord](#mediator-v1-PolicyTypeRecord) | optional |  |






<a name="mediator-v1-GetPolicyTypeRequest"></a>

### GetPolicyTypeRequest
get policy type


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| type | [string](#string) |  |  |






<a name="mediator-v1-GetPolicyTypeResponse"></a>

### GetPolicyTypeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_type | [PolicyTypeRecord](#mediator-v1-PolicyTypeRecord) | optional |  |






<a name="mediator-v1-GetPolicyTypesRequest"></a>

### GetPolicyTypesRequest
get policy types


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |






<a name="mediator-v1-GetPolicyTypesResponse"></a>

### GetPolicyTypesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_types | [PolicyTypeRecord](#mediator-v1-PolicyTypeRecord) | repeated |  |






<a name="mediator-v1-GetRepositoryRequest"></a>

### GetRepositoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [int32](#int32) |  |  |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |






<a name="mediator-v1-GetRepositoryResponse"></a>

### GetRepositoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |
| hook_url | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| registered | [bool](#bool) |  |  |






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






<a name="mediator-v1-ListRepositories"></a>

### ListRepositories
ListRepositories is used for displaying repository list data that
is relevant to users. It is not used for registering repositories.
Due to protobuf limitations, we cannot use the same Repositories for
listing repositories and registering repositories.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |
| registered | [bool](#bool) |  |  |






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
| results | [ListRepositories](#mediator-v1-ListRepositories) | repeated |  |






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






<a name="mediator-v1-PolicyRecord"></a>

### PolicyRecord
policy record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| type | [string](#string) |  |  |
| policy_definition | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="mediator-v1-PolicyTypeRecord"></a>

### PolicyTypeRecord
policy type record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| provider | [string](#string) |  |  |
| policy_type | [string](#string) |  |  |
| description | [string](#string) | optional |  |
| json_schema | [string](#string) |  |  |
| version | [string](#string) |  |  |
| default_schema | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






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
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| hook_name | [string](#string) |  |  |
| hook_type | [string](#string) |  |  |
| success | [bool](#bool) |  |  |
| uuid | [string](#string) |  |  |
| error | [google.protobuf.StringValue](#google-protobuf-StringValue) |  |  |
| registered | [bool](#bool) |  |  |






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






<a name="mediator-v1-StoreProviderTokenRequest"></a>

### StoreProviderTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| group_id | [int32](#int32) |  |  |
| access_token | [string](#string) |  |  |






<a name="mediator-v1-StoreProviderTokenResponse"></a>

### StoreProviderTokenResponse







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





 


<a name="mediator-v1-RepoFilter"></a>

### RepoFilter
Repo filter enum

| Name | Number | Description |
| ---- | ------ | ----------- |
| REPO_FILTER_SHOW_UNSPECIFIED | 0 |  |
| REPO_FILTER_SHOW_ALL | 1 |  |
| REPO_FILTER_SHOW_NOT_REGISTERED_ONLY | 2 |  |
| REPO_FILTER_SHOW_REGISTERED_ONLY | 3 |  |


 

 


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


<a name="mediator-v1-OAuthService"></a>

### OAuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetAuthorizationURL | [GetAuthorizationURLRequest](#mediator-v1-GetAuthorizationURLRequest) | [GetAuthorizationURLResponse](#mediator-v1-GetAuthorizationURLResponse) |  |
| ExchangeCodeForTokenCLI | [ExchangeCodeForTokenCLIRequest](#mediator-v1-ExchangeCodeForTokenCLIRequest) | [ExchangeCodeForTokenCLIResponse](#mediator-v1-ExchangeCodeForTokenCLIResponse) |  |
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
| GetPolicyType | [GetPolicyTypeRequest](#mediator-v1-GetPolicyTypeRequest) | [GetPolicyTypeResponse](#mediator-v1-GetPolicyTypeResponse) |  |
| GetPolicyTypes | [GetPolicyTypesRequest](#mediator-v1-GetPolicyTypesRequest) | [GetPolicyTypesResponse](#mediator-v1-GetPolicyTypesResponse) |  |
| CreatePolicy | [CreatePolicyRequest](#mediator-v1-CreatePolicyRequest) | [CreatePolicyResponse](#mediator-v1-CreatePolicyResponse) |  |
| DeletePolicy | [DeletePolicyRequest](#mediator-v1-DeletePolicyRequest) | [DeletePolicyResponse](#mediator-v1-DeletePolicyResponse) |  |
| GetPolicies | [GetPoliciesRequest](#mediator-v1-GetPoliciesRequest) | [GetPoliciesResponse](#mediator-v1-GetPoliciesResponse) |  |
| GetPolicyById | [GetPolicyByIdRequest](#mediator-v1-GetPolicyByIdRequest) | [GetPolicyByIdResponse](#mediator-v1-GetPolicyByIdResponse) |  |


<a name="mediator-v1-RepositoryService"></a>

### RepositoryService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| RegisterRepository | [RegisterRepositoryRequest](#mediator-v1-RegisterRepositoryRequest) | [RegisterRepositoryResponse](#mediator-v1-RegisterRepositoryResponse) |  |
| ListRepositories | [ListRepositoriesRequest](#mediator-v1-ListRepositoriesRequest) | [ListRepositoriesResponse](#mediator-v1-ListRepositoriesResponse) |  |
| GetRepository | [GetRepositoryRequest](#mediator-v1-GetRepositoryRequest) | [GetRepositoryResponse](#mediator-v1-GetRepositoryResponse) |  |


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

