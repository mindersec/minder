---
sidebar_position: 60
toc_max_heading_level: 4
---
# Protocol Documentation
<a name="top"></a>



<a name="mediator_v1_mediator-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## mediator/v1/mediator.proto

### Services

<a name="mediator-v1-ArtifactService"></a>

#### ArtifactService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListArtifacts | [ListArtifactsRequest](#mediator-v1-ListArtifactsRequest) | [ListArtifactsResponse](#mediator-v1-ListArtifactsResponse) |  |
| GetArtifactById | [GetArtifactByIdRequest](#mediator-v1-GetArtifactByIdRequest) | [GetArtifactByIdResponse](#mediator-v1-GetArtifactByIdResponse) |  |


<a name="mediator-v1-BranchProtectionService"></a>

#### BranchProtectionService
Get Branch Protection Settings

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetBranchProtection | [GetBranchProtectionRequest](#mediator-v1-GetBranchProtectionRequest) | [GetBranchProtectionResponse](#mediator-v1-GetBranchProtectionResponse) |  |


<a name="mediator-v1-HealthService"></a>

#### HealthService
Simple Health Check Service
replies with OK

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CheckHealth | [CheckHealthRequest](#mediator-v1-CheckHealthRequest) | [CheckHealthResponse](#mediator-v1-CheckHealthResponse) |  |


<a name="mediator-v1-KeyService"></a>

#### KeyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPublicKey | [GetPublicKeyRequest](#mediator-v1-GetPublicKeyRequest) | [GetPublicKeyResponse](#mediator-v1-GetPublicKeyResponse) |  |
| CreateKeyPair | [CreateKeyPairRequest](#mediator-v1-CreateKeyPairRequest) | [CreateKeyPairResponse](#mediator-v1-CreateKeyPairResponse) |  |


<a name="mediator-v1-OAuthService"></a>

#### OAuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetAuthorizationURL | [GetAuthorizationURLRequest](#mediator-v1-GetAuthorizationURLRequest) | [GetAuthorizationURLResponse](#mediator-v1-GetAuthorizationURLResponse) |  |
| ExchangeCodeForTokenCLI | [ExchangeCodeForTokenCLIRequest](#mediator-v1-ExchangeCodeForTokenCLIRequest) | [.google.api.HttpBody](#google-api-HttpBody) | buf:lint:ignore RPC_RESPONSE_STANDARD_NAME  protolint:disable:this |
| ExchangeCodeForTokenWEB | [ExchangeCodeForTokenWEBRequest](#mediator-v1-ExchangeCodeForTokenWEBRequest) | [ExchangeCodeForTokenWEBResponse](#mediator-v1-ExchangeCodeForTokenWEBResponse) |  |
| StoreProviderToken | [StoreProviderTokenRequest](#mediator-v1-StoreProviderTokenRequest) | [StoreProviderTokenResponse](#mediator-v1-StoreProviderTokenResponse) |  |
| RevokeOauthTokens | [RevokeOauthTokensRequest](#mediator-v1-RevokeOauthTokensRequest) | [RevokeOauthTokensResponse](#mediator-v1-RevokeOauthTokensResponse) | RevokeOauthTokens is used to revoke all tokens this a nuclear option and should only be used in emergencies |
| RevokeOauthProjectToken | [RevokeOauthProjectTokenRequest](#mediator-v1-RevokeOauthProjectTokenRequest) | [RevokeOauthProjectTokenResponse](#mediator-v1-RevokeOauthProjectTokenResponse) | revoke token for a project |
| VerifyProviderTokenFrom | [VerifyProviderTokenFromRequest](#mediator-v1-VerifyProviderTokenFromRequest) | [VerifyProviderTokenFromResponse](#mediator-v1-VerifyProviderTokenFromResponse) | VerifyProviderTokenFrom verifies that a token has been created for a provider since given timestamp |


<a name="mediator-v1-ProfileService"></a>

#### ProfileService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateProfile | [CreateProfileRequest](#mediator-v1-CreateProfileRequest) | [CreateProfileResponse](#mediator-v1-CreateProfileResponse) |  |
| DeleteProfile | [DeleteProfileRequest](#mediator-v1-DeleteProfileRequest) | [DeleteProfileResponse](#mediator-v1-DeleteProfileResponse) |  |
| ListProfiles | [ListProfilesRequest](#mediator-v1-ListProfilesRequest) | [ListProfilesResponse](#mediator-v1-ListProfilesResponse) |  |
| GetProfileById | [GetProfileByIdRequest](#mediator-v1-GetProfileByIdRequest) | [GetProfileByIdResponse](#mediator-v1-GetProfileByIdResponse) |  |
| GetProfileStatusByName | [GetProfileStatusByNameRequest](#mediator-v1-GetProfileStatusByNameRequest) | [GetProfileStatusByNameResponse](#mediator-v1-GetProfileStatusByNameResponse) |  |
| GetProfileStatusByProject | [GetProfileStatusByProjectRequest](#mediator-v1-GetProfileStatusByProjectRequest) | [GetProfileStatusByProjectResponse](#mediator-v1-GetProfileStatusByProjectResponse) |  |
| ListRuleTypes | [ListRuleTypesRequest](#mediator-v1-ListRuleTypesRequest) | [ListRuleTypesResponse](#mediator-v1-ListRuleTypesResponse) |  |
| GetRuleTypeByName | [GetRuleTypeByNameRequest](#mediator-v1-GetRuleTypeByNameRequest) | [GetRuleTypeByNameResponse](#mediator-v1-GetRuleTypeByNameResponse) |  |
| GetRuleTypeById | [GetRuleTypeByIdRequest](#mediator-v1-GetRuleTypeByIdRequest) | [GetRuleTypeByIdResponse](#mediator-v1-GetRuleTypeByIdResponse) |  |
| CreateRuleType | [CreateRuleTypeRequest](#mediator-v1-CreateRuleTypeRequest) | [CreateRuleTypeResponse](#mediator-v1-CreateRuleTypeResponse) |  |
| UpdateRuleType | [UpdateRuleTypeRequest](#mediator-v1-UpdateRuleTypeRequest) | [UpdateRuleTypeResponse](#mediator-v1-UpdateRuleTypeResponse) |  |
| DeleteRuleType | [DeleteRuleTypeRequest](#mediator-v1-DeleteRuleTypeRequest) | [DeleteRuleTypeResponse](#mediator-v1-DeleteRuleTypeResponse) |  |


<a name="mediator-v1-RepositoryService"></a>

#### RepositoryService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| RegisterRepository | [RegisterRepositoryRequest](#mediator-v1-RegisterRepositoryRequest) | [RegisterRepositoryResponse](#mediator-v1-RegisterRepositoryResponse) |  |
| ListRemoteRepositoriesFromProvider | [ListRemoteRepositoriesFromProviderRequest](#mediator-v1-ListRemoteRepositoriesFromProviderRequest) | [ListRemoteRepositoriesFromProviderResponse](#mediator-v1-ListRemoteRepositoriesFromProviderResponse) |  |
| ListRepositories | [ListRepositoriesRequest](#mediator-v1-ListRepositoriesRequest) | [ListRepositoriesResponse](#mediator-v1-ListRepositoriesResponse) |  |
| GetRepositoryById | [GetRepositoryByIdRequest](#mediator-v1-GetRepositoryByIdRequest) | [GetRepositoryByIdResponse](#mediator-v1-GetRepositoryByIdResponse) |  |
| GetRepositoryByName | [GetRepositoryByNameRequest](#mediator-v1-GetRepositoryByNameRequest) | [GetRepositoryByNameResponse](#mediator-v1-GetRepositoryByNameResponse) |  |


<a name="mediator-v1-UserService"></a>

#### UserService
manage Users CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#mediator-v1-CreateUserRequest) | [CreateUserResponse](#mediator-v1-CreateUserResponse) |  |
| DeleteUser | [DeleteUserRequest](#mediator-v1-DeleteUserRequest) | [DeleteUserResponse](#mediator-v1-DeleteUserResponse) |  |
| GetUser | [GetUserRequest](#mediator-v1-GetUserRequest) | [GetUserResponse](#mediator-v1-GetUserResponse) |  |


### Messages

<a name="mediator-v1-Artifact"></a>

#### Artifact



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact_pk | [string](#string) |  |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| type | [string](#string) |  |  |
| visibility | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| versions | [ArtifactVersion](#mediator-v1-ArtifactVersion) | repeated |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-ArtifactType"></a>

#### ArtifactType
ArtifactType defines the artifact data evaluation.


<a name="mediator-v1-ArtifactVersion"></a>

#### ArtifactVersion



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version_id | [int64](#int64) |  |  |
| tags | [string](#string) | repeated |  |
| sha | [string](#string) |  |  |
| signature_verification | [SignatureVerification](#mediator-v1-SignatureVerification) |  |  |
| github_workflow | [GithubWorkflow](#mediator-v1-GithubWorkflow) | optional |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-BranchProtection"></a>

#### BranchProtection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [string](#string) |  |  |
| is_protected | [bool](#bool) |  | Add other relevant fields |


<a name="mediator-v1-BuiltinType"></a>

#### BuiltinType
BuiltinType defines the builtin data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  |  |


<a name="mediator-v1-CheckHealthRequest"></a>

#### CheckHealthRequest



<a name="mediator-v1-CheckHealthResponse"></a>

#### CheckHealthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |


<a name="mediator-v1-Context"></a>

#### Context
Context defines the context in which a rule is evaluated.
this normally refers to a combination of the provider, organization and project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| organization | [string](#string) | optional |  |
| project | [string](#string) | optional |  |


<a name="mediator-v1-CreateKeyPairRequest"></a>

#### CreateKeyPairRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| passphrase | [string](#string) |  |  |
| project_id | [string](#string) |  |  |


<a name="mediator-v1-CreateKeyPairResponse"></a>

#### CreateKeyPairResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key_identifier | [string](#string) |  |  |
| public_key | [string](#string) |  |  |


<a name="mediator-v1-CreateProfileRequest"></a>

#### CreateProfileRequest
Profile service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#mediator-v1-Profile) |  |  |


<a name="mediator-v1-CreateProfileResponse"></a>

#### CreateProfileResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#mediator-v1-Profile) |  |  |


<a name="mediator-v1-CreateRuleTypeRequest"></a>

#### CreateRuleTypeRequest
CreateRuleTypeRequest is the request to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type to be created. |


<a name="mediator-v1-CreateRuleTypeResponse"></a>

#### CreateRuleTypeResponse
CreateRuleTypeResponse is the response to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type that was created. |


<a name="mediator-v1-CreateUserRequest"></a>

#### CreateUserRequest
User service


<a name="mediator-v1-CreateUserResponse"></a>

#### CreateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [string](#string) |  |  |
| organizatio_name | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| project_name | [string](#string) |  |  |
| identity_subject | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-DeleteProfileRequest"></a>

#### DeleteProfileRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the profile to delete |


<a name="mediator-v1-DeleteProfileResponse"></a>

#### DeleteProfileResponse



<a name="mediator-v1-DeleteRuleTypeRequest"></a>

#### DeleteRuleTypeRequest
DeleteRuleTypeRequest is the request to delete a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the rule type to be deleted. |


<a name="mediator-v1-DeleteRuleTypeResponse"></a>

#### DeleteRuleTypeResponse
DeleteRuleTypeResponse is the response to delete a rule type.


<a name="mediator-v1-DeleteUserRequest"></a>

#### DeleteUserRequest



<a name="mediator-v1-DeleteUserResponse"></a>

#### DeleteUserResponse



<a name="mediator-v1-Dependency"></a>

#### Dependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystem | [DepEcosystem](#mediator-v1-DepEcosystem) |  |  |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |


<a name="mediator-v1-DiffType"></a>

#### DiffType
DiffType defines the diff data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystems | [DiffType.Ecosystem](#mediator-v1-DiffType-Ecosystem) | repeated |  |


<a name="mediator-v1-DiffType-Ecosystem"></a>

#### DiffType.Ecosystem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the ecosystem. |
| depfile | [string](#string) |  | depfile is the file that contains the dependencies for this ecosystem |


<a name="mediator-v1-ExchangeCodeForTokenCLIRequest"></a>

#### ExchangeCodeForTokenCLIRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| code | [string](#string) |  |  |
| state | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |


<a name="mediator-v1-ExchangeCodeForTokenWEBRequest"></a>

#### ExchangeCodeForTokenWEBRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| code | [string](#string) |  |  |
| redirect_uri | [string](#string) |  |  |


<a name="mediator-v1-ExchangeCodeForTokenWEBResponse"></a>

#### ExchangeCodeForTokenWEBResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| token_type | [string](#string) |  |  |
| expires_in | [int64](#int64) |  |  |
| status | [string](#string) |  |  |


<a name="mediator-v1-GetArtifactByIdRequest"></a>

#### GetArtifactByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| latest_versions | [int32](#int32) |  |  |
| tag | [string](#string) |  |  |


<a name="mediator-v1-GetArtifactByIdResponse"></a>

#### GetArtifactByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | [Artifact](#mediator-v1-Artifact) |  |  |
| versions | [ArtifactVersion](#mediator-v1-ArtifactVersion) | repeated |  |


<a name="mediator-v1-GetAuthorizationURLRequest"></a>

#### GetAuthorizationURLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| cli | [bool](#bool) |  |  |
| port | [int32](#int32) |  |  |
| owner | [string](#string) | optional |  |


<a name="mediator-v1-GetAuthorizationURLResponse"></a>

#### GetAuthorizationURLResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  |  |


<a name="mediator-v1-GetBranchProtectionRequest"></a>

#### GetBranchProtectionRequest



<a name="mediator-v1-GetBranchProtectionResponse"></a>

#### GetBranchProtectionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch_protections | [BranchProtection](#mediator-v1-BranchProtection) | repeated |  |


<a name="mediator-v1-GetProfileByIdRequest"></a>

#### GetProfileByIdRequest
get profile by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context which contains the profiles |
| id | [string](#string) |  | id is the id of the profile to get |


<a name="mediator-v1-GetProfileByIdResponse"></a>

#### GetProfileByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | [Profile](#mediator-v1-Profile) |  |  |


<a name="mediator-v1-GetProfileStatusByNameRequest"></a>

#### GetProfileStatusByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| name | [string](#string) |  | name is the name of the profile to get |
| entity | [GetProfileStatusByNameRequest.EntityTypedId](#mediator-v1-GetProfileStatusByNameRequest-EntityTypedId) |  |  |
| all | [bool](#bool) |  |  |
| rule | [string](#string) |  |  |


<a name="mediator-v1-GetProfileStatusByNameRequest-EntityTypedId"></a>

#### GetProfileStatusByNameRequest.EntityTypedId
EntiryTypeId is a message that carries an ID together with a type to uniquely identify an entity
such as (repo, 1), (artifact, 2), ...
if the struct is reused in other messages, it should be moved to a top-level definition


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Entity](#mediator-v1-Entity) |  | entity is the entity to get status for. Incompatible with `all` |
| id | [string](#string) |  | id is the ID of the entity to get status for. Incompatible with `all` |


<a name="mediator-v1-GetProfileStatusByNameResponse"></a>

#### GetProfileStatusByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | [ProfileStatus](#mediator-v1-ProfileStatus) |  | profile_status is the status of the profile |
| rule_evaluation_status | [RuleEvaluationStatus](#mediator-v1-RuleEvaluationStatus) | repeated | rule_evaluation_status is the status of the rules |


<a name="mediator-v1-GetProfileStatusByProjectRequest"></a>

#### GetProfileStatusByProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |


<a name="mediator-v1-GetProfileStatusByProjectResponse"></a>

#### GetProfileStatusByProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | [ProfileStatus](#mediator-v1-ProfileStatus) | repeated | profile_status is the status of the profile |


<a name="mediator-v1-GetPublicKeyRequest"></a>

#### GetPublicKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key_identifier | [string](#string) |  |  |


<a name="mediator-v1-GetPublicKeyResponse"></a>

#### GetPublicKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| public_key | [string](#string) |  |  |


<a name="mediator-v1-GetRepositoryByIdRequest"></a>

#### GetRepositoryByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | [string](#string) |  |  |


<a name="mediator-v1-GetRepositoryByIdResponse"></a>

#### GetRepositoryByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#mediator-v1-Repository) |  |  |


<a name="mediator-v1-GetRepositoryByNameRequest"></a>

#### GetRepositoryByNameRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| name | [string](#string) |  |  |


<a name="mediator-v1-GetRepositoryByNameResponse"></a>

#### GetRepositoryByNameResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#mediator-v1-Repository) |  |  |


<a name="mediator-v1-GetRuleTypeByIdRequest"></a>

#### GetRuleTypeByIdRequest
GetRuleTypeByIdRequest is the request to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| id | [string](#string) |  | id is the id of the rule type. |


<a name="mediator-v1-GetRuleTypeByIdResponse"></a>

#### GetRuleTypeByIdResponse
GetRuleTypeByIdResponse is the response to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type. |


<a name="mediator-v1-GetRuleTypeByNameRequest"></a>

#### GetRuleTypeByNameRequest
GetRuleTypeByNameRequest is the request to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule type is evaluated. |
| name | [string](#string) |  | name is the name of the rule type. |


<a name="mediator-v1-GetRuleTypeByNameResponse"></a>

#### GetRuleTypeByNameResponse
GetRuleTypeByNameResponse is the response to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type. |


<a name="mediator-v1-GetSecretByIdRequest"></a>

#### GetSecretByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |


<a name="mediator-v1-GetSecretByIdResponse"></a>

#### GetSecretByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  | Add other relevant fields |


<a name="mediator-v1-GetSecretsRequest"></a>

#### GetSecretsRequest



<a name="mediator-v1-GetSecretsResponse"></a>

#### GetSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [GetSecretByIdResponse](#mediator-v1-GetSecretByIdResponse) | repeated |  |


<a name="mediator-v1-GetUserRequest"></a>

#### GetUserRequest
list users
get user


<a name="mediator-v1-GetUserResponse"></a>

#### GetUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [UserRecord](#mediator-v1-UserRecord) | optional |  |
| projects | [Project](#mediator-v1-Project) | repeated |  |


<a name="mediator-v1-GetVulnerabilitiesRequest"></a>

#### GetVulnerabilitiesRequest



<a name="mediator-v1-GetVulnerabilitiesResponse"></a>

#### GetVulnerabilitiesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vulns | [GetVulnerabilityByIdResponse](#mediator-v1-GetVulnerabilityByIdResponse) | repeated |  |


<a name="mediator-v1-GetVulnerabilityByIdRequest"></a>

#### GetVulnerabilityByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |


<a name="mediator-v1-GetVulnerabilityByIdResponse"></a>

#### GetVulnerabilityByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | May require adjustment, currently set up for GitHub Security Advisories only |
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


<a name="mediator-v1-GitHubProviderConfig"></a>

#### GitHubProviderConfig
GitHubProviderConfig contains the configuration for the GitHub client

Endpoint: is the GitHub API endpoint

If using the public GitHub API, Endpoint can be left blank
disable revive linting for this struct as there is nothing wrong with the
naming convention


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | Endpoint is the GitHub API endpoint. If using the public GitHub API, Endpoint can be left blank. |


<a name="mediator-v1-GitType"></a>

#### GitType
GitType defines the git data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clone_url | [string](#string) |  | clone_url is the url of the git repository. |
| branch | [string](#string) |  | branch is the branch of the git repository. |


<a name="mediator-v1-GithubWorkflow"></a>

#### GithubWorkflow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| repository | [string](#string) |  |  |
| commit_sha | [string](#string) |  |  |
| trigger | [string](#string) |  |  |


<a name="mediator-v1-ListArtifactsRequest"></a>

#### ListArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |


<a name="mediator-v1-ListArtifactsResponse"></a>

#### ListArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [Artifact](#mediator-v1-Artifact) | repeated |  |


<a name="mediator-v1-ListProfilesRequest"></a>

#### ListProfilesRequest
list profiles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context which contains the profiles |


<a name="mediator-v1-ListProfilesResponse"></a>

#### ListProfilesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profiles | [Profile](#mediator-v1-Profile) | repeated |  |


<a name="mediator-v1-ListRemoteRepositoriesFromProviderRequest"></a>

#### ListRemoteRepositoriesFromProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |


<a name="mediator-v1-ListRemoteRepositoriesFromProviderResponse"></a>

#### ListRemoteRepositoriesFromProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [UpstreamRepositoryRef](#mediator-v1-UpstreamRepositoryRef) | repeated |  |


<a name="mediator-v1-ListRepositoriesRequest"></a>

#### ListRepositoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| limit | [int32](#int32) |  |  |
| offset | [int32](#int32) |  |  |


<a name="mediator-v1-ListRepositoriesResponse"></a>

#### ListRepositoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [Repository](#mediator-v1-Repository) | repeated |  |


<a name="mediator-v1-ListRuleTypesRequest"></a>

#### ListRuleTypesRequest
ListRuleTypesRequest is the request to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule types are evaluated. |


<a name="mediator-v1-ListRuleTypesResponse"></a>

#### ListRuleTypesResponse
ListRuleTypesResponse is the response to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_types | [RuleType](#mediator-v1-RuleType) | repeated | rule_types is the list of rule types. |


<a name="mediator-v1-PrDependencies"></a>

#### PrDependencies



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pr | [PullRequest](#mediator-v1-PullRequest) |  |  |
| deps | [PrDependencies.ContextualDependency](#mediator-v1-PrDependencies-ContextualDependency) | repeated |  |


<a name="mediator-v1-PrDependencies-ContextualDependency"></a>

#### PrDependencies.ContextualDependency



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dep | [Dependency](#mediator-v1-Dependency) |  |  |
| file | [PrDependencies.ContextualDependency.FilePatch](#mediator-v1-PrDependencies-ContextualDependency-FilePatch) |  |  |


<a name="mediator-v1-PrDependencies-ContextualDependency-FilePatch"></a>

#### PrDependencies.ContextualDependency.FilePatch



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | file changed, e.g. package-lock.json |
| patch_url | [string](#string) |  | points to the the raw patchfile |


<a name="mediator-v1-Profile"></a>

#### Profile
Profile defines a profile that is user defined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the profile is evaluated. |
| id | [string](#string) | optional | id is the id of the profile. This is optional and is set by the system. |
| name | [string](#string) |  | name is the name of the profile instance. |
| repository | [Profile.Rule](#mediator-v1-Profile-Rule) | repeated | These are the entities that one could set in the profile. |
| build_environment | [Profile.Rule](#mediator-v1-Profile-Rule) | repeated |  |
| artifact | [Profile.Rule](#mediator-v1-Profile-Rule) | repeated |  |
| pull_request | [Profile.Rule](#mediator-v1-Profile-Rule) | repeated |  |
| remediate | [string](#string) | optional | whether and how to remediate (on,off,dry_run) this is optional as the default is set by the system |
| alert | [string](#string) | optional | whether and how to alert (on,off,dry_run) this is optional as the default is set by the system |


<a name="mediator-v1-Profile-Rule"></a>

#### Profile.Rule
Rule defines the individual call of a certain rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the rule to be instantiated. |
| params | [google.protobuf.Struct](#google-protobuf-Struct) |  | params are the parameters that are passed to the rule. This is optional and depends on the rule type. |
| def | [google.protobuf.Struct](#google-protobuf-Struct) |  | def is the definition of the rule. This depends on the rule type. |


<a name="mediator-v1-ProfileStatus"></a>

#### ProfileStatus
get the overall profile status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | [string](#string) |  | profile_id is the id of the profile |
| profile_name | [string](#string) |  | profile_name is the name of the profile |
| profile_status | [string](#string) |  | profile_status is the status of the profile |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the profile was updated |


<a name="mediator-v1-Project"></a>

#### Project
Project API Objects


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| is_protected | [bool](#bool) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-Provider"></a>

#### Provider
Provider defines a provider that is used to connect to a certain service.
This is used to define the context in which a rule is evaluated and serves
as a data ingestion point. They are top level entities and are scoped to
an organization.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| context | [Provider.Context](#mediator-v1-Provider-Context) |  |  |
| version | [string](#string) |  | Version defines the version of the provider. Currently only v1 is supported. |
| implements | [string](#string) | repeated | Implements defines the provider types that this provider implements. This is used to determine the interface to use to interact with the provider. This is a required field and must be set. currently, the following interfaces are supported: - rest - github - git |
| def | [Provider.Definition](#mediator-v1-Provider-Definition) |  |  |


<a name="mediator-v1-Provider-Context"></a>

#### Provider.Context
Context defines the context in which a provider is evaluated.
Given thta a provider is a top level entity, it may only be scoped to
an organization.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organization | [string](#string) |  |  |
| project | [string](#string) |  |  |


<a name="mediator-v1-Provider-Definition"></a>

#### Provider.Definition
Definition defines the definition of the provider.
This is used to define the connection to the provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rest | [RESTProviderConfig](#mediator-v1-RESTProviderConfig) | optional | rest is the REST provider configuration. |
| github | [GitHubProviderConfig](#mediator-v1-GitHubProviderConfig) | optional | github is the GitHub provider configuration. |


<a name="mediator-v1-PullRequest"></a>

#### PullRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The full URL to the PR |
| commit_sha | [string](#string) |  | Commit SHA of the PR HEAD. Will be useful to submit a review |
| number | [int32](#int32) |  | The sequential PR number (not the DB PK!) |
| repo_owner | [string](#string) |  | The owner of the repo, will be used to submit a review |
| repo_name | [string](#string) |  | The name of the repo, will be used to submit a review |
| author_id | [int64](#int64) |  | The author of the PR, will be used to check if we can request changes |
| action | [string](#string) |  | The action that triggered the webhook |


<a name="mediator-v1-RESTProviderConfig"></a>

#### RESTProviderConfig
RESTProviderConfig contains the configuration for the REST provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| base_url | [string](#string) |  | base_url is the base URL for the REST provider. |


<a name="mediator-v1-RefreshTokenRequest"></a>

#### RefreshTokenRequest



<a name="mediator-v1-RefreshTokenResponse"></a>

#### RefreshTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| access_token_expires_in | [int64](#int64) |  |  |


<a name="mediator-v1-RegisterRepoResult"></a>

#### RegisterRepoResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | [Repository](#mediator-v1-Repository) |  |  |
| status | [RegisterRepoResult.Status](#mediator-v1-RegisterRepoResult-Status) |  |  |


<a name="mediator-v1-RegisterRepoResult-Status"></a>

#### RegisterRepoResult.Status



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) | optional |  |


<a name="mediator-v1-RegisterRepositoryRequest"></a>

#### RegisterRepositoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| repositories | [UpstreamRepositoryRef](#mediator-v1-UpstreamRepositoryRef) | repeated |  |


<a name="mediator-v1-RegisterRepositoryResponse"></a>

#### RegisterRepositoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [RegisterRepoResult](#mediator-v1-RegisterRepoResult) | repeated |  |


<a name="mediator-v1-Repository"></a>

#### Repository



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | This is optional when returning remote repositories |
| context | [Context](#mediator-v1-Context) | optional |  |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |
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


<a name="mediator-v1-RestType"></a>

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
| fallback | [RestType.Fallback](#mediator-v1-RestType-Fallback) | repeated | fallback provides a body that the ingester would return in case the REST call returns a non-200 status code. |


<a name="mediator-v1-RestType-Fallback"></a>

#### RestType.Fallback



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_code | [int32](#int32) |  |  |
| body | [string](#string) |  |  |


<a name="mediator-v1-RevokeOauthProjectTokenRequest"></a>

#### RevokeOauthProjectTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |


<a name="mediator-v1-RevokeOauthProjectTokenResponse"></a>

#### RevokeOauthProjectTokenResponse



<a name="mediator-v1-RevokeOauthTokensRequest"></a>

#### RevokeOauthTokensRequest



<a name="mediator-v1-RevokeOauthTokensResponse"></a>

#### RevokeOauthTokensResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revoked_tokens | [int32](#int32) |  |  |


<a name="mediator-v1-RpcOptions"></a>

#### RpcOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| anonymous | [bool](#bool) |  |  |
| no_log | [bool](#bool) |  |  |
| owner_only | [bool](#bool) |  |  |
| root_admin_only | [bool](#bool) |  |  |
| auth_scope | [ObjectOwner](#mediator-v1-ObjectOwner) |  |  |


<a name="mediator-v1-RuleEvaluationStatus"></a>

#### RuleEvaluationStatus
get the status of the rules for a given profile


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | [string](#string) |  | profile_id is the id of the profile |
| rule_id | [string](#string) |  | rule_id is the id of the rule |
| rule_name | [string](#string) |  | rule_name is the name of the rule |
| entity | [string](#string) |  | entity is the entity that was evaluated |
| status | [string](#string) |  | status is the status of the evaluation |
| last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_updated is the last time the profile was updated |
| entity_info | [RuleEvaluationStatus.EntityInfoEntry](#mediator-v1-RuleEvaluationStatus-EntityInfoEntry) | repeated | entity_info is the information about the entity |
| details | [string](#string) |  | details is the description of the evaluation if any |
| guidance | [string](#string) |  | guidance is the guidance for the evaluation if any |
| remediation_status | [string](#string) |  | remediation_status is the status of the remediation |
| remediation_last_updated | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | remediation_last_updated is the last time the remediation was performed or attempted |
| remediation_details | [string](#string) |  | remediation_details is the description of the remediation attempt if any |


<a name="mediator-v1-RuleEvaluationStatus-EntityInfoEntry"></a>

#### RuleEvaluationStatus.EntityInfoEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |


<a name="mediator-v1-RuleType"></a>

#### RuleType
RuleType defines rules that may or may not be user defined.
The version is assumed from the folder's version.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) | optional | id is the id of the rule type. This is mostly optional and is set by the server. |
| name | [string](#string) |  | name is the name of the rule type. |
| context | [Context](#mediator-v1-Context) |  | context is the context in which the rule is evaluated. |
| def | [RuleType.Definition](#mediator-v1-RuleType-Definition) |  | def is the definition of the rule type. |
| description | [string](#string) |  | description is the description of the rule type. |
| guidance | [string](#string) |  | guidance are instructions we give the user in case a rule fails. |


<a name="mediator-v1-RuleType-Definition"></a>

#### RuleType.Definition
Definition defines the rule type. It encompases the schema and the data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| in_entity | [string](#string) |  | in_entity is the entity in which the rule is evaluated. This can be repository, build_environment or artifact. |
| rule_schema | [google.protobuf.Struct](#google-protobuf-Struct) |  | rule_schema is the schema of the rule. This is expressed in JSON Schema. |
| param_schema | [google.protobuf.Struct](#google-protobuf-Struct) | optional | param_schema is the schema of the parameters that are passed to the rule. This is expressed in JSON Schema. |
| ingest | [RuleType.Definition.Ingest](#mediator-v1-RuleType-Definition-Ingest) |  |  |
| eval | [RuleType.Definition.Eval](#mediator-v1-RuleType-Definition-Eval) |  |  |
| remediate | [RuleType.Definition.Remediate](#mediator-v1-RuleType-Definition-Remediate) |  |  |
| alert | [RuleType.Definition.Alert](#mediator-v1-RuleType-Definition-Alert) |  |  |


<a name="mediator-v1-RuleType-Definition-Alert"></a>

#### RuleType.Definition.Alert



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| security_advisory | [RuleType.Definition.Alert.AlertTypeSA](#mediator-v1-RuleType-Definition-Alert-AlertTypeSA) | optional |  |


<a name="mediator-v1-RuleType-Definition-Alert-AlertTypeSA"></a>

#### RuleType.Definition.Alert.AlertTypeSA



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| severity | [string](#string) |  |  |


<a name="mediator-v1-RuleType-Definition-Eval"></a>

#### RuleType.Definition.Eval
Eval defines the data evaluation definition.
This pertains to the way we traverse data from the upstream
endpoint and how we compare it to the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data evaluation. Right now only `jq` is supported as a driver |
| jq | [RuleType.Definition.Eval.JQComparison](#mediator-v1-RuleType-Definition-Eval-JQComparison) | repeated | jq is only used if the `jq` type is selected. It defines the comparisons that are made between the ingested data and the profile rule. |
| rego | [RuleType.Definition.Eval.Rego](#mediator-v1-RuleType-Definition-Eval-Rego) | optional | rego is only used if the `rego` type is selected. |
| vulncheck | [RuleType.Definition.Eval.Vulncheck](#mediator-v1-RuleType-Definition-Eval-Vulncheck) | optional | vulncheck is only used if the `vulncheck` type is selected. |
| package_intelligence | [RuleType.Definition.Eval.PackageIntelligence](#mediator-v1-RuleType-Definition-Eval-PackageIntelligence) | optional | package_intelligence is only used if the `package_intelligence` type is selected. |


<a name="mediator-v1-RuleType-Definition-Eval-JQComparison"></a>

#### RuleType.Definition.Eval.JQComparison



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ingested | [RuleType.Definition.Eval.JQComparison.Operator](#mediator-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Ingested points to the data retrieved in the `ingest` section |
| profile | [RuleType.Definition.Eval.JQComparison.Operator](#mediator-v1-RuleType-Definition-Eval-JQComparison-Operator) |  | Profile points to the profile itself. |


<a name="mediator-v1-RuleType-Definition-Eval-JQComparison-Operator"></a>

#### RuleType.Definition.Eval.JQComparison.Operator



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| def | [string](#string) |  |  |


<a name="mediator-v1-RuleType-Definition-Eval-PackageIntelligence"></a>

#### RuleType.Definition.Eval.PackageIntelligence



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | [string](#string) |  | e.g. https://staging.stacklok.dev/ |


<a name="mediator-v1-RuleType-Definition-Eval-Rego"></a>

#### RuleType.Definition.Eval.Rego



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of evaluation engine to use for rego. We currently have two modes of operation: - deny-by-default: this is the default mode of operation where we deny access by default and allow access only if the profile explicitly allows it. It expects the profile to set an `allow` variable to true or false. - constraints: this is the mode of operation where we allow access by default and deny access only if a violation is found. It expects the profile to set a `violations` variable with a "msg" field. |
| def | [string](#string) |  | def is the definition of the rego profile. |


<a name="mediator-v1-RuleType-Definition-Eval-Vulncheck"></a>

#### RuleType.Definition.Eval.Vulncheck
no configuration for now


<a name="mediator-v1-RuleType-Definition-Ingest"></a>

#### RuleType.Definition.Ingest
Ingest defines how the data is ingested.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  | type is the type of the data ingestion. we currently support rest, artifact and builtin. |
| rest | [RestType](#mediator-v1-RestType) | optional | rest is the rest data ingestion. this is only used if the type is rest. |
| builtin | [BuiltinType](#mediator-v1-BuiltinType) | optional | builtin is the builtin data ingestion. |
| artifact | [ArtifactType](#mediator-v1-ArtifactType) | optional | artifact is the artifact data ingestion. |
| git | [GitType](#mediator-v1-GitType) | optional | git is the git data ingestion. |
| diff | [DiffType](#mediator-v1-DiffType) | optional | diff is the diff data ingestion. |


<a name="mediator-v1-RuleType-Definition-Remediate"></a>

#### RuleType.Definition.Remediate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| rest | [RestType](#mediator-v1-RestType) | optional |  |
| gh_branch_protection | [RuleType.Definition.Remediate.GhBranchProtectionType](#mediator-v1-RuleType-Definition-Remediate-GhBranchProtectionType) | optional |  |
| pull_request | [RuleType.Definition.Remediate.PullRequestRemediation](#mediator-v1-RuleType-Definition-Remediate-PullRequestRemediation) | optional |  |


<a name="mediator-v1-RuleType-Definition-Remediate-GhBranchProtectionType"></a>

#### RuleType.Definition.Remediate.GhBranchProtectionType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| patch | [string](#string) |  |  |


<a name="mediator-v1-RuleType-Definition-Remediate-PullRequestRemediation"></a>

#### RuleType.Definition.Remediate.PullRequestRemediation
the name stutters a bit but we already use a PullRequest message for handling PR entities


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | the title of the PR |
| body | [string](#string) |  | the body of the PR |
| contents | [RuleType.Definition.Remediate.PullRequestRemediation.Content](#mediator-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content) | repeated |  |


<a name="mediator-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content"></a>

#### RuleType.Definition.Remediate.PullRequestRemediation.Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | the file to patch |
| action | [string](#string) |  | how to patch the file. For now, only replace is supported |
| content | [string](#string) |  | the content of the file |
| mode | [string](#string) | optional | the GIT mode of the file. Not UNIX mode! String because the GH API also uses strings the usual modes are: 100644 for regular files, 100755 for executable files and 040000 for submodules (which we don't use but now you know the meaning of the 1 in 100644) |


<a name="mediator-v1-SignatureVerification"></a>

#### SignatureVerification



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

#### StoreProviderTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| access_token | [string](#string) |  |  |
| owner | [string](#string) | optional |  |


<a name="mediator-v1-StoreProviderTokenResponse"></a>

#### StoreProviderTokenResponse



<a name="mediator-v1-UpdateRuleTypeRequest"></a>

#### UpdateRuleTypeRequest
UpdateRuleTypeRequest is the request to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type to be updated. |


<a name="mediator-v1-UpdateRuleTypeResponse"></a>

#### UpdateRuleTypeResponse
UpdateRuleTypeResponse is the response to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | [RuleType](#mediator-v1-RuleType) |  | rule_type is the rule type that was updated. |


<a name="mediator-v1-UpstreamRepositoryRef"></a>

#### UpstreamRepositoryRef



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | [string](#string) |  |  |
| name | [string](#string) |  |  |
| repo_id | [int32](#int32) |  |  |


<a name="mediator-v1-UserRecord"></a>

#### UserRecord
user record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [int32](#int32) |  |  |
| organization_id | [string](#string) |  |  |
| identity_subject | [string](#string) |  |  |
| created_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| updated_at | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-VerifyProviderTokenFromRequest"></a>

#### VerifyProviderTokenFromRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  |  |
| project_id | [string](#string) |  |  |
| timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |


<a name="mediator-v1-VerifyProviderTokenFromResponse"></a>

#### VerifyProviderTokenFromResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [string](#string) |  |  |


| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| rpc_options | RpcOptions | .google.protobuf.MethodOptions | 51077 |  |





<a name="mediator-v1-DepEcosystem"></a>

### DepEcosystem


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEP_ECOSYSTEM_UNSPECIFIED | 0 |  |
| DEP_ECOSYSTEM_NPM | 1 |  |
| DEP_ECOSYSTEM_GO | 2 |  |
| DEP_ECOSYSTEM_PYPI | 3 |  |


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


<a name="mediator-v1-ObjectOwner"></a>

### ObjectOwner


| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_OWNER_UNSPECIFIED | 0 |  |
| OBJECT_OWNER_ORGANIZATION | 1 |  |
| OBJECT_OWNER_PROJECT | 2 |  |
| OBJECT_OWNER_USER | 3 |  |




<a name="mediator_v1_mediator-proto-extensions"></a>

### File-level Extensions
| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
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
