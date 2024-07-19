---
sidebar_position: 60
title: Protocol documentation
toc_max_heading_level: 4
---

import useBrokenLinks from '@docusaurus/useBrokenLinks';

export const File = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <h2 id={id} name={id}>{children}</h2>;
}
export const Service = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <h4 id={id} name={id}>{children}</h4>;
}
export const Message = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <h4 id={id} name={id}>{children}</h4>;
}
export const Extension = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <h3 id={id} name={id}>{children}</h3>;
}
export const Enum = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <h3 id={id} name={id}>{children}</h3>;
}
export const ProtoType = ({ children, id }) => {
  useBrokenLinks().collectAnchor(id);
  return <a id={id} name={id}>{children}</a>;
}
export const TypeLink = ({ children, type }) => {
  let link = type.startsWith('google-protobuf-') ?
    `https://protobuf.dev/reference/protobuf/google.protobuf/#${type.replace('google-protobuf-', '')}` :
    `#${type}`;
  return <a href={link}>{children}</a>;
}


# Protocol documentation
<a id="top"></a>




<File id="minder_v1_minder-proto">minder/v1/minder.proto</File>


### Services


<Service id="minder-v1-ArtifactService">ArtifactService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListArtifacts | [ListArtifactsRequest](#minder-v1-ListArtifactsRequest) | [ListArtifactsResponse](#minder-v1-ListArtifactsResponse) |  |
| GetArtifactById | [GetArtifactByIdRequest](#minder-v1-GetArtifactByIdRequest) | [GetArtifactByIdResponse](#minder-v1-GetArtifactByIdResponse) |  |
| GetArtifactByName | [GetArtifactByNameRequest](#minder-v1-GetArtifactByNameRequest) | [GetArtifactByNameResponse](#minder-v1-GetArtifactByNameResponse) |  |



<Service id="minder-v1-EvalResultsService">EvalResultsService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListEvaluationResults | [ListEvaluationResultsRequest](#minder-v1-ListEvaluationResultsRequest) | [ListEvaluationResultsResponse](#minder-v1-ListEvaluationResultsResponse) |  |
| ListEvaluationHistory | [ListEvaluationHistoryRequest](#minder-v1-ListEvaluationHistoryRequest) | [ListEvaluationHistoryResponse](#minder-v1-ListEvaluationHistoryResponse) |  |



<Service id="minder-v1-HealthService">HealthService</Service>

Simple Health Check Service
replies with OK

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CheckHealth | [CheckHealthRequest](#minder-v1-CheckHealthRequest) | [CheckHealthResponse](#minder-v1-CheckHealthResponse) |  |



<Service id="minder-v1-InviteService">InviteService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInviteDetails | [GetInviteDetailsRequest](#minder-v1-GetInviteDetailsRequest) | [GetInviteDetailsResponse](#minder-v1-GetInviteDetailsResponse) |  |



<Service id="minder-v1-OAuthService">OAuthService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetAuthorizationURL | [GetAuthorizationURLRequest](#minder-v1-GetAuthorizationURLRequest) | [GetAuthorizationURLResponse](#minder-v1-GetAuthorizationURLResponse) |  |
| StoreProviderToken | [StoreProviderTokenRequest](#minder-v1-StoreProviderTokenRequest) | [StoreProviderTokenResponse](#minder-v1-StoreProviderTokenResponse) |  |
| VerifyProviderTokenFrom | [VerifyProviderTokenFromRequest](#minder-v1-VerifyProviderTokenFromRequest) | [VerifyProviderTokenFromResponse](#minder-v1-VerifyProviderTokenFromResponse) | VerifyProviderTokenFrom verifies that a token has been created for a provider since given timestamp |
| VerifyProviderCredential | [VerifyProviderCredentialRequest](#minder-v1-VerifyProviderCredentialRequest) | [VerifyProviderCredentialResponse](#minder-v1-VerifyProviderCredentialResponse) | VerifyProviderCredential verifies that a credential has been created matching the enrollment nonce |



<Service id="minder-v1-PermissionsService">PermissionsService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#minder-v1-ListRolesRequest) | [ListRolesResponse](#minder-v1-ListRolesResponse) |  |
| ListRoleAssignments | [ListRoleAssignmentsRequest](#minder-v1-ListRoleAssignmentsRequest) | [ListRoleAssignmentsResponse](#minder-v1-ListRoleAssignmentsResponse) |  |
| AssignRole | [AssignRoleRequest](#minder-v1-AssignRoleRequest) | [AssignRoleResponse](#minder-v1-AssignRoleResponse) |  |
| UpdateRole | [UpdateRoleRequest](#minder-v1-UpdateRoleRequest) | [UpdateRoleResponse](#minder-v1-UpdateRoleResponse) |  |
| RemoveRole | [RemoveRoleRequest](#minder-v1-RemoveRoleRequest) | [RemoveRoleResponse](#minder-v1-RemoveRoleResponse) |  |



<Service id="minder-v1-ProfileService">ProfileService</Service>



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



<Service id="minder-v1-ProjectsService">ProjectsService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListProjects | [ListProjectsRequest](#minder-v1-ListProjectsRequest) | [ListProjectsResponse](#minder-v1-ListProjectsResponse) |  |
| CreateProject | [CreateProjectRequest](#minder-v1-CreateProjectRequest) | [CreateProjectResponse](#minder-v1-CreateProjectResponse) |  |
| ListChildProjects | [ListChildProjectsRequest](#minder-v1-ListChildProjectsRequest) | [ListChildProjectsResponse](#minder-v1-ListChildProjectsResponse) |  |
| DeleteProject | [DeleteProjectRequest](#minder-v1-DeleteProjectRequest) | [DeleteProjectResponse](#minder-v1-DeleteProjectResponse) |  |
| UpdateProject | [UpdateProjectRequest](#minder-v1-UpdateProjectRequest) | [UpdateProjectResponse](#minder-v1-UpdateProjectResponse) |  |
| PatchProject | [PatchProjectRequest](#minder-v1-PatchProjectRequest) | [PatchProjectResponse](#minder-v1-PatchProjectResponse) |  |
| CreateEntityReconciliationTask | [CreateEntityReconciliationTaskRequest](#minder-v1-CreateEntityReconciliationTaskRequest) | [CreateEntityReconciliationTaskResponse](#minder-v1-CreateEntityReconciliationTaskResponse) |  |



<Service id="minder-v1-ProvidersService">ProvidersService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| PatchProvider | [PatchProviderRequest](#minder-v1-PatchProviderRequest) | [PatchProviderResponse](#minder-v1-PatchProviderResponse) |  |
| GetProvider | [GetProviderRequest](#minder-v1-GetProviderRequest) | [GetProviderResponse](#minder-v1-GetProviderResponse) |  |
| ListProviders | [ListProvidersRequest](#minder-v1-ListProvidersRequest) | [ListProvidersResponse](#minder-v1-ListProvidersResponse) |  |
| CreateProvider | [CreateProviderRequest](#minder-v1-CreateProviderRequest) | [CreateProviderResponse](#minder-v1-CreateProviderResponse) |  |
| DeleteProvider | [DeleteProviderRequest](#minder-v1-DeleteProviderRequest) | [DeleteProviderResponse](#minder-v1-DeleteProviderResponse) |  |
| DeleteProviderByID | [DeleteProviderByIDRequest](#minder-v1-DeleteProviderByIDRequest) | [DeleteProviderByIDResponse](#minder-v1-DeleteProviderByIDResponse) |  |
| GetUnclaimedProviders | [GetUnclaimedProvidersRequest](#minder-v1-GetUnclaimedProvidersRequest) | [GetUnclaimedProvidersResponse](#minder-v1-GetUnclaimedProvidersResponse) | GetUnclaimedProviders returns a list of known provider configurations that this user could claim based on their identity.  This is a read-only operation for use by clients which wish to present a menu of options. |
| ListProviderClasses | [ListProviderClassesRequest](#minder-v1-ListProviderClassesRequest) | [ListProviderClassesResponse](#minder-v1-ListProviderClassesResponse) |  |
| ReconcileEntityRegistration | [ReconcileEntityRegistrationRequest](#minder-v1-ReconcileEntityRegistrationRequest) | [ReconcileEntityRegistrationResponse](#minder-v1-ReconcileEntityRegistrationResponse) |  |



<Service id="minder-v1-RepositoryService">RepositoryService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| RegisterRepository | [RegisterRepositoryRequest](#minder-v1-RegisterRepositoryRequest) | [RegisterRepositoryResponse](#minder-v1-RegisterRepositoryResponse) |  |
| ListRemoteRepositoriesFromProvider | [ListRemoteRepositoriesFromProviderRequest](#minder-v1-ListRemoteRepositoriesFromProviderRequest) | [ListRemoteRepositoriesFromProviderResponse](#minder-v1-ListRemoteRepositoriesFromProviderResponse) |  |
| ListRepositories | [ListRepositoriesRequest](#minder-v1-ListRepositoriesRequest) | [ListRepositoriesResponse](#minder-v1-ListRepositoriesResponse) |  |
| GetRepositoryById | [GetRepositoryByIdRequest](#minder-v1-GetRepositoryByIdRequest) | [GetRepositoryByIdResponse](#minder-v1-GetRepositoryByIdResponse) |  |
| GetRepositoryByName | [GetRepositoryByNameRequest](#minder-v1-GetRepositoryByNameRequest) | [GetRepositoryByNameResponse](#minder-v1-GetRepositoryByNameResponse) |  |
| DeleteRepositoryById | [DeleteRepositoryByIdRequest](#minder-v1-DeleteRepositoryByIdRequest) | [DeleteRepositoryByIdResponse](#minder-v1-DeleteRepositoryByIdResponse) |  |
| DeleteRepositoryByName | [DeleteRepositoryByNameRequest](#minder-v1-DeleteRepositoryByNameRequest) | [DeleteRepositoryByNameResponse](#minder-v1-DeleteRepositoryByNameResponse) |  |



<Service id="minder-v1-UserService">UserService</Service>

manage Users CRUD

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateUser | [CreateUserRequest](#minder-v1-CreateUserRequest) | [CreateUserResponse](#minder-v1-CreateUserResponse) |  |
| DeleteUser | [DeleteUserRequest](#minder-v1-DeleteUserRequest) | [DeleteUserResponse](#minder-v1-DeleteUserResponse) |  |
| GetUser | [GetUserRequest](#minder-v1-GetUserRequest) | [GetUserResponse](#minder-v1-GetUserResponse) |  |
| ListInvitations | [ListInvitationsRequest](#minder-v1-ListInvitationsRequest) | [ListInvitationsResponse](#minder-v1-ListInvitationsResponse) | ListInvitations returns a list of invitations for the user based on the user's registered email address.  Note that a user who receives an invitation code may still accept the invitation even if the code was directed to a different email address.  This is because understanding the routing of email messages is beyond the scope of Minder.  This API endpoint may be called without the logged-in user previously having called `CreateUser`. |
| ResolveInvitation | [ResolveInvitationRequest](#minder-v1-ResolveInvitationRequest) | [ResolveInvitationResponse](#minder-v1-ResolveInvitationResponse) | ResolveInvitation allows a user to accept or decline an invitation to a project given the code for the invitation. A user may call ResolveInvitation to accept or decline an invitation even if they have not called CreateUser.  If a user accepts an invitation via this call before calling CreateUser, a Minder user record will be created, but no additional projects will be created (unlike CreateUser, which will also create a default project). |


### Messages


<Message id="minder-v1-Artifact">Artifact</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact_pk | <TypeLink type="string">string</TypeLink> |  |  |
| owner | <TypeLink type="string">string</TypeLink> |  |  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| type | <TypeLink type="string">string</TypeLink> |  |  |
| visibility | <TypeLink type="string">string</TypeLink> |  |  |
| repository | <TypeLink type="string">string</TypeLink> |  |  |
| versions | <TypeLink type="minder-v1-ArtifactVersion">ArtifactVersion</TypeLink> | repeated |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-ArtifactType">ArtifactType</Message>

ArtifactType defines the artifact data evaluation.



<Message id="minder-v1-ArtifactVersion">ArtifactVersion</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version_id | <TypeLink type="int64">int64</TypeLink> |  |  |
| tags | <TypeLink type="string">string</TypeLink> | repeated |  |
| sha | <TypeLink type="string">string</TypeLink> |  |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |



<Message id="minder-v1-AssignRoleRequest">AssignRoleRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the role assignment is evaluated. |
| role_assignment | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> |  | role_assignment is the role assignment to be created. |



<Message id="minder-v1-AssignRoleResponse">AssignRoleResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignment | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> |  | role_assignment is the role assignment that was created. |
| invitation | <TypeLink type="minder-v1-Invitation">Invitation</TypeLink> |  | invitation contains the details of the invitation for the assigned user to join the project if the user is not already a member. |



<Message id="minder-v1-AuthorizationParams">AuthorizationParams</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| authorization_url | <TypeLink type="string">string</TypeLink> |  | authorization_url is an external URL to use to authorize the provider. |



<Message id="minder-v1-AutoRegistration">AutoRegistration</Message>

AutoRegistration is the configuration for auto-registering entities.
When nothing is set, it means that auto-registration is disabled. There is no difference between disabled
and undefined so for the "let's not auto-register anything" case we'd just let the repeated string empty


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | <TypeLink type="minder-v1-AutoRegistration-EntitiesEntry">AutoRegistration.EntitiesEntry</TypeLink> | repeated | enabled is the list of entities that are enabled for auto-registration. |



<Message id="minder-v1-AutoRegistration-EntitiesEntry">AutoRegistration.EntitiesEntry</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | <TypeLink type="string">string</TypeLink> |  |  |
| value | <TypeLink type="minder-v1-EntityAutoRegistrationConfig">EntityAutoRegistrationConfig</TypeLink> |  |  |



<Message id="minder-v1-BranchProtection">BranchProtection</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | <TypeLink type="string">string</TypeLink> |  |  |
| is_protected | <TypeLink type="bool">bool</TypeLink> |  | Add other relevant fields |



<Message id="minder-v1-Build">Build</Message>





<Message id="minder-v1-BuiltinType">BuiltinType</Message>

BuiltinType defines the builtin data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-CheckHealthRequest">CheckHealthRequest</Message>





<Message id="minder-v1-CheckHealthResponse">CheckHealthResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-Context">Context</Message>

Context defines the context in which a rule is evaluated.
this normally refers to a combination of the provider, organization and project.

Removing the 'optional' keyword from the following two fields below will break
buf compatibility checks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> | optional | name of the provider |
| project | <TypeLink type="string">string</TypeLink> | optional | ID of the project |
| retired_organization | <TypeLink type="string">string</TypeLink> | optional |  |



<Message id="minder-v1-ContextV2">ContextV2</Message>

ContextV2 defines the context in which a rule is evaluated.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | <TypeLink type="string">string</TypeLink> |  | project is the project ID |
| provider | <TypeLink type="string">string</TypeLink> |  | name of the provider. Set to empty string when not applicable. |



<Message id="minder-v1-CreateEntityReconciliationTaskRequest">CreateEntityReconciliationTaskRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | <TypeLink type="minder-v1-EntityTypedId">EntityTypedId</TypeLink> |  | entity is the entity to be reconciled. |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the entity reconciliation task is created. |



<Message id="minder-v1-CreateEntityReconciliationTaskResponse">CreateEntityReconciliationTaskResponse</Message>





<Message id="minder-v1-CreateProfileRequest">CreateProfileRequest</Message>

Profile service


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-CreateProfileResponse">CreateProfileResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-CreateProjectRequest">CreateProjectRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the project is created. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the project to create. |



<Message id="minder-v1-CreateProjectResponse">CreateProjectResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | <TypeLink type="minder-v1-Project">Project</TypeLink> |  | project is the project that was created. |



<Message id="minder-v1-CreateProviderRequest">CreateProviderRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the provider is created. |
| provider | <TypeLink type="minder-v1-Provider">Provider</TypeLink> |  | provider is the provider to be created. |



<Message id="minder-v1-CreateProviderResponse">CreateProviderResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="minder-v1-Provider">Provider</TypeLink> |  | provider is the provider that was created. |
| authorization | <TypeLink type="minder-v1-AuthorizationParams">AuthorizationParams</TypeLink> |  | authorization provides additional authorization information needed to complete the initialization of the provider. |



<Message id="minder-v1-CreateRuleTypeRequest">CreateRuleTypeRequest</Message>

CreateRuleTypeRequest is the request to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type to be created. |



<Message id="minder-v1-CreateRuleTypeResponse">CreateRuleTypeResponse</Message>

CreateRuleTypeResponse is the response to create a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type that was created. |



<Message id="minder-v1-CreateUserRequest">CreateUserRequest</Message>

User service



<Message id="minder-v1-CreateUserResponse">CreateUserResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="int32">int32</TypeLink> |  |  |
| organization_id | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| organizatio_name | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| project_id | <TypeLink type="string">string</TypeLink> |  |  |
| project_name | <TypeLink type="string">string</TypeLink> |  |  |
| identity_subject | <TypeLink type="string">string</TypeLink> |  |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-Cursor">Cursor</Message>

Cursor message to be used in request messages. Its purpose is to
allow clients to specify the subset of records to retrieve by means
of index within a collection, along with the number of items to
retrieve.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cursor | <TypeLink type="string">string</TypeLink> |  | cursor is the index to start from within the collection being retrieved. It's an opaque payload specified and interpreted on an per-rpc basis. |
| size | <TypeLink type="uint32">uint32</TypeLink> |  | size is the number of items to retrieve from the collection. |



<Message id="minder-v1-CursorPage">CursorPage</Message>

CursorPage message used in response messages. Its purpose is to
send to clients links pointing to next and/or previous collection
subsets with respect to the one containing this struct.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| total_records | <TypeLink type="uint32">uint32</TypeLink> |  | Total number of records matching the request. This is optional. |
| next | <TypeLink type="minder-v1-Cursor">Cursor</TypeLink> |  | Cursor pointing to retrieve results logically placed after the ones shipped with the message containing this struct. |
| prev | <TypeLink type="minder-v1-Cursor">Cursor</TypeLink> |  | Cursor pointing to retrieve results logically placed before the ones shipped with the message containing this struct. |



<Message id="minder-v1-DeleteProfileRequest">DeleteProfileRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the profile to delete |



<Message id="minder-v1-DeleteProfileResponse">DeleteProfileResponse</Message>





<Message id="minder-v1-DeleteProjectRequest">DeleteProjectRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the project is deleted. |



<Message id="minder-v1-DeleteProjectResponse">DeleteProjectResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | <TypeLink type="string">string</TypeLink> |  | project_id is the id of the project that was deleted. |



<Message id="minder-v1-DeleteProviderByIDRequest">DeleteProviderByIDRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the provider is deleted. Only the project is required in this context. |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the provider to delete |



<Message id="minder-v1-DeleteProviderByIDResponse">DeleteProviderByIDResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the provider that was deleted |



<Message id="minder-v1-DeleteProviderRequest">DeleteProviderRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the provider is deleted. Both project and provider are required in this context. |



<Message id="minder-v1-DeleteProviderResponse">DeleteProviderResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the provider that was deleted |



<Message id="minder-v1-DeleteRepositoryByIdRequest">DeleteRepositoryByIdRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-DeleteRepositoryByIdResponse">DeleteRepositoryByIdResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-DeleteRepositoryByNameRequest">DeleteRepositoryByNameRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-DeleteRepositoryByNameResponse">DeleteRepositoryByNameResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-DeleteRuleTypeRequest">DeleteRuleTypeRequest</Message>

DeleteRuleTypeRequest is the request to delete a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the rule type to be deleted. |



<Message id="minder-v1-DeleteRuleTypeResponse">DeleteRuleTypeResponse</Message>

DeleteRuleTypeResponse is the response to delete a rule type.



<Message id="minder-v1-DeleteUserRequest">DeleteUserRequest</Message>





<Message id="minder-v1-DeleteUserResponse">DeleteUserResponse</Message>





<Message id="minder-v1-DiffType">DiffType</Message>

DiffType defines the diff data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ecosystems | <TypeLink type="minder-v1-DiffType-Ecosystem">DiffType.Ecosystem</TypeLink> | repeated | ecosystems is the list of ecosystems to be used for the "dep" diff type. |
| type | <TypeLink type="string">string</TypeLink> |  | type is the type of diff ingestor to use. The default is "dep" which will leverage the ecosystems array. |



<Message id="minder-v1-DiffType-Ecosystem">DiffType.Ecosystem</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the ecosystem. |
| depfile | <TypeLink type="string">string</TypeLink> |  | depfile is the file that contains the dependencies for this ecosystem |



<Message id="minder-v1-DockerHubProviderConfig">DockerHubProviderConfig</Message>

DockerHubProviderConfig contains the configuration for the DockerHub provider.

Namespace: is the namespace for the DockerHub provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | <TypeLink type="string">string</TypeLink> | optional | namespace is the namespace for the DockerHub provider. |



<Message id="minder-v1-EntityAutoRegistrationConfig">EntityAutoRegistrationConfig</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | <TypeLink type="bool">bool</TypeLink> | optional |  |



<Message id="minder-v1-EntityTypedId">EntityTypedId</Message>

EntiryTypeId is a message that carries an ID together with a type to uniquely identify an entity
such as (repo, 1), (artifact, 2), ...


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="minder-v1-Entity">Entity</TypeLink> |  | entity is the entity to get status for. Incompatible with `all` |
| id | <TypeLink type="string">string</TypeLink> |  | id is the ID of the entity to get status for. Incompatible with `all` |



<Message id="minder-v1-EvalResultAlert">EvalResultAlert</Message>

EvalResultAlert holds the alert details for a given rule evaluation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is the status of the alert |
| last_updated | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | last_updated is the last time the alert was performed or attempted |
| details | <TypeLink type="string">string</TypeLink> |  | details is the description of the alert attempt if any |
| url | <TypeLink type="string">string</TypeLink> |  | url is the URL to the alert |



<Message id="minder-v1-EvaluationHistory">EvaluationHistory</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | <TypeLink type="minder-v1-EvaluationHistoryEntity">EvaluationHistoryEntity</TypeLink> |  | entity contains details of the entity which was evaluated. |
| rule | <TypeLink type="minder-v1-EvaluationHistoryRule">EvaluationHistoryRule</TypeLink> |  | rule contains details of the rule which the entity was evaluated against. |
| status | <TypeLink type="minder-v1-EvaluationHistoryStatus">EvaluationHistoryStatus</TypeLink> |  | status contains the evaluation status. |
| alert | <TypeLink type="minder-v1-EvaluationHistoryAlert">EvaluationHistoryAlert</TypeLink> |  | alert contains details of the alerts for this evaluation. |
| remediation | <TypeLink type="minder-v1-EvaluationHistoryRemediation">EvaluationHistoryRemediation</TypeLink> |  | remediation contains details of the remediation for this evaluation. |
| evaluated_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | created_at is the timestamp of creation of this evaluation |



<Message id="minder-v1-EvaluationHistoryAlert">EvaluationHistoryAlert</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (on, off, error, skipped, not available) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the alert. the structure and contents are alert specific, and are subject to change. |



<Message id="minder-v1-EvaluationHistoryEntity">EvaluationHistoryEntity</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> |  | id is the unique identifier of the entity. |
| type | <TypeLink type="minder-v1-Entity">Entity</TypeLink> |  | type is the entity type. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the entity name. |



<Message id="minder-v1-EvaluationHistoryRemediation">EvaluationHistoryRemediation</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (success, error, failure, skipped, not available) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the remediation. the structure and contents are remediation specific, and are subject to change. |



<Message id="minder-v1-EvaluationHistoryRule">EvaluationHistoryRule</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the rule instance. |
| rule_type | <TypeLink type="string">string</TypeLink> |  | type is the name of the rule type. |
| profile | <TypeLink type="string">string</TypeLink> |  | profile is the name of the profile which contains the rule. |



<Message id="minder-v1-EvaluationHistoryStatus">EvaluationHistoryStatus</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (success, error, failure, skipped) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the evaluation. the structure and contents are rule type specific, and are subject to change. |



<Message id="minder-v1-GHCRProviderConfig">GHCRProviderConfig</Message>

GHCRProviderConfig contains the configuration for the GHCR provider.

Namespace: is the namespace for the GHCR provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| namespace | <TypeLink type="string">string</TypeLink> | optional | namespace is the namespace for the GHCR provider. |



<Message id="minder-v1-GetArtifactByIdRequest">GetArtifactByIdRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-GetArtifactByIdResponse">GetArtifactByIdResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | <TypeLink type="minder-v1-Artifact">Artifact</TypeLink> |  |  |
| versions | <TypeLink type="minder-v1-ArtifactVersion">ArtifactVersion</TypeLink> | repeated |  |



<Message id="minder-v1-GetArtifactByNameRequest">GetArtifactByNameRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-GetArtifactByNameResponse">GetArtifactByNameResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| artifact | <TypeLink type="minder-v1-Artifact">Artifact</TypeLink> |  |  |
| versions | <TypeLink type="minder-v1-ArtifactVersion">ArtifactVersion</TypeLink> | repeated |  |



<Message id="minder-v1-GetAuthorizationURLRequest">GetAuthorizationURLRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| cli | <TypeLink type="bool">bool</TypeLink> |  |  |
| port | <TypeLink type="int32">int32</TypeLink> |  |  |
| owner | <TypeLink type="string">string</TypeLink> | optional |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| redirect_url | <TypeLink type="string">string</TypeLink> | optional |  |
| config | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> |  | config is a JSON object that can be used to pass additional configuration |
| provider_class | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-GetAuthorizationURLResponse">GetAuthorizationURLResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | <TypeLink type="string">string</TypeLink> |  |  |
| state | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-GetInviteDetailsRequest">GetInviteDetailsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | <TypeLink type="string">string</TypeLink> |  | Invite nonce/code to retrieve details for |



<Message id="minder-v1-GetInviteDetailsResponse">GetInviteDetailsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_display | <TypeLink type="string">string</TypeLink> |  | Project associated with the invite |
| sponsor_display | <TypeLink type="string">string</TypeLink> |  | Sponsor of the invite |
| expires_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | expires_at is the time at which the invitation expires. |
| expired | <TypeLink type="bool">bool</TypeLink> |  | expired is true if the invitation has expired |



<Message id="minder-v1-GetProfileByIdRequest">GetProfileByIdRequest</Message>

get profile by id


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context which contains the profiles |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the profile to get |



<Message id="minder-v1-GetProfileByIdResponse">GetProfileByIdResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-GetProfileStatusByNameRequest">GetProfileStatusByNameRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the profile to get |
| entity | <TypeLink type="minder-v1-EntityTypedId">EntityTypedId</TypeLink> |  |  |
| all | <TypeLink type="bool">bool</TypeLink> |  |  |
| rule | <TypeLink type="string">string</TypeLink> |  | **Deprecated.** rule is the type of the rule. Deprecated in favor of rule_type |
| rule_type | <TypeLink type="string">string</TypeLink> |  |  |
| rule_name | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-GetProfileStatusByNameResponse">GetProfileStatusByNameResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | <TypeLink type="minder-v1-ProfileStatus">ProfileStatus</TypeLink> |  | profile_status is the status of the profile |
| rule_evaluation_status | <TypeLink type="minder-v1-RuleEvaluationStatus">RuleEvaluationStatus</TypeLink> | repeated | rule_evaluation_status is the status of the rules |



<Message id="minder-v1-GetProfileStatusByProjectRequest">GetProfileStatusByProjectRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |



<Message id="minder-v1-GetProfileStatusByProjectResponse">GetProfileStatusByProjectResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | <TypeLink type="minder-v1-ProfileStatus">ProfileStatus</TypeLink> | repeated | profile_status is the status of the profile |



<Message id="minder-v1-GetProviderRequest">GetProviderRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the provider is evaluated. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the provider to get. |



<Message id="minder-v1-GetProviderResponse">GetProviderResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="minder-v1-Provider">Provider</TypeLink> |  | provider is the provider that was retrieved. |



<Message id="minder-v1-GetRepositoryByIdRequest">GetRepositoryByIdRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository_id | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-GetRepositoryByIdResponse">GetRepositoryByIdResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | <TypeLink type="minder-v1-Repository">Repository</TypeLink> |  |  |



<Message id="minder-v1-GetRepositoryByNameRequest">GetRepositoryByNameRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-GetRepositoryByNameResponse">GetRepositoryByNameResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | <TypeLink type="minder-v1-Repository">Repository</TypeLink> |  |  |



<Message id="minder-v1-GetRuleTypeByIdRequest">GetRuleTypeByIdRequest</Message>

GetRuleTypeByIdRequest is the request to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |
| id | <TypeLink type="string">string</TypeLink> |  | id is the id of the rule type. |



<Message id="minder-v1-GetRuleTypeByIdResponse">GetRuleTypeByIdResponse</Message>

GetRuleTypeByIdResponse is the response to get a rule type by id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type. |



<Message id="minder-v1-GetRuleTypeByNameRequest">GetRuleTypeByNameRequest</Message>

GetRuleTypeByNameRequest is the request to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule type is evaluated. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the rule type. |



<Message id="minder-v1-GetRuleTypeByNameResponse">GetRuleTypeByNameResponse</Message>

GetRuleTypeByNameResponse is the response to get a rule type by name.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type. |



<Message id="minder-v1-GetUnclaimedProvidersRequest">GetUnclaimedProvidersRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the set of providers are evaluated. |



<Message id="minder-v1-GetUnclaimedProvidersResponse">GetUnclaimedProvidersResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| providers | <TypeLink type="minder-v1-ProviderParameter">ProviderParameter</TypeLink> | repeated | providers is a set of parameters which can be supplied to allow the user to assign existing unclaimed credentials to a new provider in the project via CreateProvider(). |



<Message id="minder-v1-GetUserRequest">GetUserRequest</Message>

get user



<Message id="minder-v1-GetUserResponse">GetUserResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | <TypeLink type="minder-v1-UserRecord">UserRecord</TypeLink> | optional |  |
| projects | <TypeLink type="minder-v1-Project">Project</TypeLink> | repeated | **Deprecated.** This will be deprecated in favor of the project_roles field |
| project_roles | <TypeLink type="minder-v1-ProjectRole">ProjectRole</TypeLink> | repeated |  |



<Message id="minder-v1-GitHubAppParams">GitHubAppParams</Message>

GitHubAppParams is the parameters for a GitHub App provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| installation_id | <TypeLink type="int64">int64</TypeLink> |  | The GitHub installation ID for the app. On create, this is the only parameter used; the organization parameters are ignored. |
| organization | <TypeLink type="string">string</TypeLink> |  | The GitHub organization slug where the app is installed. This is an output-only parameter, and is validated on input if set (i.e. the value must be either empty or match the org of the installation_id). |
| organization_id | <TypeLink type="int64">int64</TypeLink> |  | The GitHub organization ID where the app is installed. This is an output-only parameter, and is validated on input if set (i.e. the value must be either empty or match the org of the installation_id). |



<Message id="minder-v1-GitHubAppProviderConfig">GitHubAppProviderConfig</Message>

GitHubAppProviderConfig contains the configuration for the GitHub App provider


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | <TypeLink type="string">string</TypeLink> | optional | Endpoint is the GitHub API endpoint. If using the public GitHub API, Endpoint can be left blank. |



<Message id="minder-v1-GitHubProviderConfig">GitHubProviderConfig</Message>

GitHubProviderConfig contains the configuration for the GitHub client

Endpoint: is the GitHub API endpoint

If using the public GitHub API, Endpoint can be left blank
disable revive linting for this struct as there is nothing wrong with the
naming convention


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | <TypeLink type="string">string</TypeLink> | optional | Endpoint is the GitHub API endpoint. If using the public GitHub API, Endpoint can be left blank. |



<Message id="minder-v1-GitType">GitType</Message>

GitType defines the git data ingester.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| clone_url | <TypeLink type="string">string</TypeLink> |  | clone_url is the url of the git repository. |
| branch | <TypeLink type="string">string</TypeLink> |  | branch is the branch of the git repository. |



<Message id="minder-v1-Invitation">Invitation</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | <TypeLink type="string">string</TypeLink> |  | role is the role that would be assigned if the user accepts the invitation. |
| email | <TypeLink type="string">string</TypeLink> |  | email is the email address of the invited user. This is presented as a convenience for display purposes, and does not affect who can accept the invitation using the code. |
| project | <TypeLink type="string">string</TypeLink> |  | project is the project to which the user is invited. |
| code | <TypeLink type="string">string</TypeLink> |  | code is a unique identifier for the invitation, which can be used by the recipient to accept or reject the invitation. The code is only transmitted in response to AssignRole or ListInvitations RPCs, and not transmitted in ListRoleAssignments or other calls. |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | created_at is the time at which the invitation was created. |
| expires_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | expires_at is the time at which the invitation expires. |
| expired | <TypeLink type="bool">bool</TypeLink> |  | expired is true if the invitation has expired. |
| sponsor | <TypeLink type="string">string</TypeLink> |  | sponsor is the account (ID) of the user who created the invitation. |
| sponsor_display | <TypeLink type="string">string</TypeLink> |  | sponsor_display is the display name of the user who created the invitation. |
| project_display | <TypeLink type="string">string</TypeLink> |  | project_display is the display name of the project to which the user is invited. |
| invite_url | <TypeLink type="string">string</TypeLink> |  | inviteURL is the URL that can be used to accept the invitation. |
| email_skipped | <TypeLink type="bool">bool</TypeLink> |  | emailSkipped is true if the email was not sent to the invitee. |



<Message id="minder-v1-ListArtifactsRequest">ListArtifactsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| from | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-ListArtifactsResponse">ListArtifactsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | <TypeLink type="minder-v1-Artifact">Artifact</TypeLink> | repeated |  |



<Message id="minder-v1-ListChildProjectsRequest">ListChildProjectsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-ContextV2">ContextV2</TypeLink> |  | context is the context in which the child projects are listed. |
| recursive | <TypeLink type="bool">bool</TypeLink> |  | recursive is true if child projects should be listed recursively. |



<Message id="minder-v1-ListChildProjectsResponse">ListChildProjectsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | <TypeLink type="minder-v1-Project">Project</TypeLink> | repeated |  |



<Message id="minder-v1-ListEvaluationHistoryRequest">ListEvaluationHistoryRequest</Message>

ListEvaluationHistoryRequest represents a request message for the
ListEvaluationHistory RPC.

Most of its fields are used for filtering, except for `cursor`
which is used for pagination.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| entity_type | <TypeLink type="string">string</TypeLink> | repeated | List of entity types to retrieve. |
| entity_name | <TypeLink type="string">string</TypeLink> | repeated | List of entity names to retrieve. |
| profile_name | <TypeLink type="string">string</TypeLink> | repeated | List of profile names to retrieve. |
| status | <TypeLink type="string">string</TypeLink> | repeated | List of evaluation statuses to retrieve. |
| remediation | <TypeLink type="string">string</TypeLink> | repeated | List of remediation statuses to retrieve. |
| alert | <TypeLink type="string">string</TypeLink> | repeated | List of alert statuses to retrieve. |
| from | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | Timestamp representing the start time of the selection window. |
| to | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | Timestamp representing the end time of the selection window. |
| cursor | <TypeLink type="minder-v1-Cursor">Cursor</TypeLink> |  | Cursor object to select the "page" of data to retrieve. |



<Message id="minder-v1-ListEvaluationHistoryResponse">ListEvaluationHistoryResponse</Message>

ListEvaluationHistoryResponse represents a response message for the
ListEvaluationHistory RPC.

It ships a collection of records retrieved and pointers to get to
the next and/or previous pages of data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | <TypeLink type="minder-v1-EvaluationHistory">EvaluationHistory</TypeLink> | repeated | List of records retrieved. |
| page | <TypeLink type="minder-v1-CursorPage">CursorPage</TypeLink> |  | Metadata of the current page and pointers to next and/or previous pages. |



<Message id="minder-v1-ListEvaluationResultsRequest">ListEvaluationResultsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the evaluation results are evaluated. |
| profile | <TypeLink type="string">string</TypeLink> |  | ID can contain either a profile name or an ID |
| label_filter | <TypeLink type="string">string</TypeLink> |  | Filter profiles to only those matching the specified labels.

The default is to return all user-created profiles; the string "*" can be used to select all profiles, including system profiles. This syntax may be expanded in the future. |
| entity | <TypeLink type="minder-v1-EntityTypedId">EntityTypedId</TypeLink> | repeated | If set, only return evaluation results for the named entities. If empty, return evaluation results for all entities |
| rule_name | <TypeLink type="string">string</TypeLink> | repeated | If set, only return evaluation results for the named rules. If empty, return evaluation results for all rules |



<Message id="minder-v1-ListEvaluationResultsResponse">ListEvaluationResultsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | <TypeLink type="minder-v1-ListEvaluationResultsResponse-EntityEvaluationResults">ListEvaluationResultsResponse.EntityEvaluationResults</TypeLink> | repeated | Each entity selected by the list request will have _single_ entry in entities which contains results of all evaluations for each profile. |



<Message id="minder-v1-ListEvaluationResultsResponse-EntityEvaluationResults">ListEvaluationResultsResponse.EntityEvaluationResults</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | <TypeLink type="minder-v1-EntityTypedId">EntityTypedId</TypeLink> |  |  |
| profiles | <TypeLink type="minder-v1-ListEvaluationResultsResponse-EntityProfileEvaluationResults">ListEvaluationResultsResponse.EntityProfileEvaluationResults</TypeLink> | repeated |  |



<Message id="minder-v1-ListEvaluationResultsResponse-EntityProfileEvaluationResults">ListEvaluationResultsResponse.EntityProfileEvaluationResults</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | <TypeLink type="minder-v1-ProfileStatus">ProfileStatus</TypeLink> |  | profile_status is the status of the profile - id, name, status, last_updated |
| results | <TypeLink type="minder-v1-RuleEvaluationStatus">RuleEvaluationStatus</TypeLink> | repeated | Note that some fields like profile_id and entity might be empty Eventually we might replace this type with another one that fits the API better |



<Message id="minder-v1-ListInvitationsRequest">ListInvitationsRequest</Message>





<Message id="minder-v1-ListInvitationsResponse">ListInvitationsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| invitations | <TypeLink type="minder-v1-Invitation">Invitation</TypeLink> | repeated |  |



<Message id="minder-v1-ListProfilesRequest">ListProfilesRequest</Message>

list profiles


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context which contains the profiles |
| label_filter | <TypeLink type="string">string</TypeLink> |  | Filter profiles to only those matching the specified labels.

The default is to return all user-created profiles; the string "*" can be used to select all profiles, including system profiles. This syntax may be expanded in the future. |



<Message id="minder-v1-ListProfilesResponse">ListProfilesResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profiles | <TypeLink type="minder-v1-Profile">Profile</TypeLink> | repeated |  |



<Message id="minder-v1-ListProjectsRequest">ListProjectsRequest</Message>





<Message id="minder-v1-ListProjectsResponse">ListProjectsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | <TypeLink type="minder-v1-Project">Project</TypeLink> | repeated |  |



<Message id="minder-v1-ListProviderClassesRequest">ListProviderClassesRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the provider classes are evaluated. |



<Message id="minder-v1-ListProviderClassesResponse">ListProviderClassesResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider_classes | <TypeLink type="string">string</TypeLink> | repeated | provider_classes is the list of provider classes. |



<Message id="minder-v1-ListProvidersRequest">ListProvidersRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the providers are evaluated. |
| limit | <TypeLink type="int32">int32</TypeLink> |  | limit is the maximum number of providers to return. |
| cursor | <TypeLink type="string">string</TypeLink> |  | cursor is the cursor to use for the page of results, empty if at the beginning |



<Message id="minder-v1-ListProvidersResponse">ListProvidersResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| providers | <TypeLink type="minder-v1-Provider">Provider</TypeLink> | repeated |  |
| cursor | <TypeLink type="string">string</TypeLink> |  | cursor is the cursor to use for the next page of results, empty if at the end |



<Message id="minder-v1-ListRemoteRepositoriesFromProviderRequest">ListRemoteRepositoriesFromProviderRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-ListRemoteRepositoriesFromProviderResponse">ListRemoteRepositoriesFromProviderResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | <TypeLink type="minder-v1-UpstreamRepositoryRef">UpstreamRepositoryRef</TypeLink> | repeated |  |



<Message id="minder-v1-ListRepositoriesRequest">ListRepositoriesRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| limit | <TypeLink type="int64">int64</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| cursor | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-ListRepositoriesResponse">ListRepositoriesResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | <TypeLink type="minder-v1-Repository">Repository</TypeLink> | repeated |  |
| cursor | <TypeLink type="string">string</TypeLink> |  | cursor is the cursor to use for the next page of results, empty if at the end |



<Message id="minder-v1-ListRoleAssignmentsRequest">ListRoleAssignmentsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the role assignments are evaluated. |



<Message id="minder-v1-ListRoleAssignmentsResponse">ListRoleAssignmentsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignments | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> | repeated | role_assignments contains permission grants which have been accepted by a user. |
| invitations | <TypeLink type="minder-v1-Invitation">Invitation</TypeLink> | repeated | invitations contains outstanding role invitations which have not yet been accepted by a user. |



<Message id="minder-v1-ListRolesRequest">ListRolesRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the roles are evaluated. |



<Message id="minder-v1-ListRolesResponse">ListRolesResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | <TypeLink type="minder-v1-Role">Role</TypeLink> | repeated |  |



<Message id="minder-v1-ListRuleTypesRequest">ListRuleTypesRequest</Message>

ListRuleTypesRequest is the request to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule types are evaluated. |



<Message id="minder-v1-ListRuleTypesResponse">ListRuleTypesResponse</Message>

ListRuleTypesResponse is the response to list rule types.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_types | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> | repeated | rule_types is the list of rule types. |



<Message id="minder-v1-PatchProfileRequest">PatchProfileRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | The context in which the patch is applied. Provided explicitly so that the patch itself can be minimal and contain only the attribute to set, e.g. remediate=true |
| id | <TypeLink type="string">string</TypeLink> |  | The id of the profile to patch. Same explanation about explicitness as for the context |
| patch | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  | The patch to apply to the profile |
| update_mask | <TypeLink type="google-protobuf-FieldMask">google.protobuf.FieldMask</TypeLink> |  | needed to enable PATCH, see https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/patch_feature/ is not exposed to the API user |



<Message id="minder-v1-PatchProfileResponse">PatchProfileResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-PatchProjectRequest">PatchProjectRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the project is updated. |
| patch | <TypeLink type="minder-v1-ProjectPatch">ProjectPatch</TypeLink> |  | patch is the patch to apply to the project |
| update_mask | <TypeLink type="google-protobuf-FieldMask">google.protobuf.FieldMask</TypeLink> |  | needed to enable PATCH, see https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/patch_feature/ is not exposed to the API user |



<Message id="minder-v1-PatchProjectResponse">PatchProjectResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | <TypeLink type="minder-v1-Project">Project</TypeLink> |  | project is the project that was updated. |



<Message id="minder-v1-PatchProviderRequest">PatchProviderRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| patch | <TypeLink type="minder-v1-Provider">Provider</TypeLink> |  |  |
| update_mask | <TypeLink type="google-protobuf-FieldMask">google.protobuf.FieldMask</TypeLink> |  |  |



<Message id="minder-v1-PatchProviderResponse">PatchProviderResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="minder-v1-Provider">Provider</TypeLink> |  |  |



<Message id="minder-v1-PipelineRun">PipelineRun</Message>





<Message id="minder-v1-Profile">Profile</Message>

Profile defines a profile that is user defined.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the profile is evaluated. |
| id | <TypeLink type="string">string</TypeLink> | optional | id is the id of the profile. This is optional and is set by the system. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the profile instance. |
| labels | <TypeLink type="string">string</TypeLink> | repeated | labels are a set of system-provided attributes which can be used to filter profiles and status results. Labels cannot be set by the user, but are returned in ListProfiles.

Labels use DNS label constraints, with a possible namespace prefix separated by a colon (:). They are intended to allow filtering, but not to store arbitrary metadata. DNS labels are 1-63 character alphanumeric strings with internal hyphens. An RE2-style validation regex would be:

DNS_STR = "[a-zA-Z0-9](?[-a-zA-Z0-9]{0,61}[a-zA-Z0-9])?" ($DNS_STR:)?$DNS_STR |
| repository | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated | These are the entities that one could set in the profile. |
| build_environment | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| artifact | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| pull_request | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| release | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| pipeline_run | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| task_run | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| build | <TypeLink type="minder-v1-Profile-Rule">Profile.Rule</TypeLink> | repeated |  |
| selection | <TypeLink type="minder-v1-Profile-Selector">Profile.Selector</TypeLink> | repeated |  |
| remediate | <TypeLink type="string">string</TypeLink> | optional | whether and how to remediate (on,off,dry_run) this is optional and defaults to "off" |
| alert | <TypeLink type="string">string</TypeLink> | optional | whether and how to alert (on,off,dry_run) this is optional and defaults to "on" |
| type | <TypeLink type="string">string</TypeLink> |  | type is a placeholder for the object type. It should always be set to "profile". |
| version | <TypeLink type="string">string</TypeLink> |  | version is the version of the profile type. In this case, it is "v1" |
| display_name | <TypeLink type="string">string</TypeLink> |  | display_name is the display name of the profile. |



<Message id="minder-v1-Profile-Rule">Profile.Rule</Message>

Rule defines the individual call of a certain rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  | type is the type of the rule to be instantiated. |
| params | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> |  | params are the parameters that are passed to the rule. This is optional and depends on the rule type. |
| def | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> |  | def is the definition of the rule. This depends on the rule type. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the descriptive name of the rule, not to be confused with type |



<Message id="minder-v1-Profile-Selector">Profile.Selector</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> |  | id is optional and use for updates to match upserts as well as read operations. It is ignored for creates. |
| entity | <TypeLink type="string">string</TypeLink> |  | entity is the entity to select. |
| selector | <TypeLink type="string">string</TypeLink> |  | expr is the expression to select the entity. |
| description | <TypeLink type="string">string</TypeLink> |  | description is the human-readable description of the selector. |



<Message id="minder-v1-ProfileStatus">ProfileStatus</Message>

get the overall profile status


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | <TypeLink type="string">string</TypeLink> |  | profile_id is the id of the profile |
| profile_name | <TypeLink type="string">string</TypeLink> |  | profile_name is the name of the profile |
| profile_status | <TypeLink type="string">string</TypeLink> |  | profile_status is the status of the profile |
| last_updated | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | last_updated is the last time the profile was updated |
| profile_display_name | <TypeLink type="string">string</TypeLink> |  | profile_display_name is the display name of the profile |



<Message id="minder-v1-Project">Project</Message>

Project API Objects


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | <TypeLink type="string">string</TypeLink> |  |  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| description | <TypeLink type="string">string</TypeLink> |  |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| updated_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| display_name | <TypeLink type="string">string</TypeLink> |  | display_name allows for a human-readable name to be used. display_names are short *non-unique* strings to provide a user-friendly name for presentation in lists, etc. |



<Message id="minder-v1-ProjectPatch">ProjectPatch</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| display_name | <TypeLink type="string">string</TypeLink> | optional | display_name is the display name of the project to update. |
| description | <TypeLink type="string">string</TypeLink> | optional | description is the description of the project to update. |



<Message id="minder-v1-ProjectRole">ProjectRole</Message>

ProjectRole has the project along with the role the user has in the project


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | <TypeLink type="minder-v1-Role">Role</TypeLink> |  |  |
| project | <TypeLink type="minder-v1-Project">Project</TypeLink> |  |  |



<Message id="minder-v1-Provider">Provider</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the provider. |
| class | <TypeLink type="string">string</TypeLink> |  | class is the name of the provider implementation, eg. 'github' or 'gh-app'. |
| project | <TypeLink type="string">string</TypeLink> |  | project is the project where the provider is. This is ignored on input in favor of the context field in CreateProviderRequest. |
| version | <TypeLink type="string">string</TypeLink> |  | version is the version of the provider. |
| implements | <TypeLink type="minder-v1-ProviderType">ProviderType</TypeLink> | repeated | implements is the list of interfaces that the provider implements. |
| config | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> |  | config is the configuration of the provider. |
| auth_flows | <TypeLink type="minder-v1-AuthorizationFlow">AuthorizationFlow</TypeLink> | repeated | auth_flows is the list of authorization flows that the provider supports. |
| parameters | <TypeLink type="minder-v1-ProviderParameter">ProviderParameter</TypeLink> |  | parameters is the list of parameters that the provider requires. |
| credentials_state | <TypeLink type="string">string</TypeLink> |  | credentials_state is the state of the credentials for the provider. This is an output-only field. It may be: "set", "unset", "not_applicable". |



<Message id="minder-v1-ProviderConfig">ProviderConfig</Message>

ProviderConfig contains the generic configuration for a provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| auto_registration | <TypeLink type="minder-v1-AutoRegistration">AutoRegistration</TypeLink> | optional | auto_registration is the configuration for auto-registering entities. |



<Message id="minder-v1-ProviderParameter">ProviderParameter</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| github_app | <TypeLink type="minder-v1-GitHubAppParams">GitHubAppParams</TypeLink> |  |  |



<Message id="minder-v1-PullRequest">PullRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | <TypeLink type="string">string</TypeLink> |  | The full URL to the PR |
| commit_sha | <TypeLink type="string">string</TypeLink> |  | Commit SHA of the PR HEAD. Will be useful to submit a review |
| number | <TypeLink type="int64">int64</TypeLink> |  | The sequential PR number (not the DB PK!) |
| repo_owner | <TypeLink type="string">string</TypeLink> |  | The owner of the repo, will be used to submit a review |
| repo_name | <TypeLink type="string">string</TypeLink> |  | The name of the repo, will be used to submit a review |
| author_id | <TypeLink type="int64">int64</TypeLink> |  | The author of the PR, will be used to check if we can request changes |
| action | <TypeLink type="string">string</TypeLink> |  | The action that triggered the webhook |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-RESTProviderConfig">RESTProviderConfig</Message>

RESTProviderConfig contains the configuration for the REST provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| base_url | <TypeLink type="string">string</TypeLink> | optional | base_url is the base URL for the REST provider. |



<Message id="minder-v1-ReconcileEntityRegistrationRequest">ReconcileEntityRegistrationRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| entity | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-ReconcileEntityRegistrationResponse">ReconcileEntityRegistrationResponse</Message>





<Message id="minder-v1-RegisterRepoResult">RegisterRepoResult</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repository | <TypeLink type="minder-v1-Repository">Repository</TypeLink> |  |  |
| status | <TypeLink type="minder-v1-RegisterRepoResult-Status">RegisterRepoResult.Status</TypeLink> |  |  |



<Message id="minder-v1-RegisterRepoResult-Status">RegisterRepoResult.Status</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | <TypeLink type="bool">bool</TypeLink> |  |  |
| error | <TypeLink type="string">string</TypeLink> | optional |  |



<Message id="minder-v1-RegisterRepositoryRequest">RegisterRepositoryRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| repository | <TypeLink type="minder-v1-UpstreamRepositoryRef">UpstreamRepositoryRef</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-RegisterRepositoryResponse">RegisterRepositoryResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| result | <TypeLink type="minder-v1-RegisterRepoResult">RegisterRepoResult</TypeLink> |  |  |



<Message id="minder-v1-Release">Release</Message>

Stubs for the SDLC entities



<Message id="minder-v1-RemoveRoleRequest">RemoveRoleRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the role assignment is evaluated. |
| role_assignment | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> |  | role_assignment is the role assignment to be removed. |



<Message id="minder-v1-RemoveRoleResponse">RemoveRoleResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignment | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> |  | role_assignment is the role assignment that was removed. |
| invitation | <TypeLink type="minder-v1-Invitation">Invitation</TypeLink> |  | invitation contains the details of the invitation that was removed. |



<Message id="minder-v1-Repository">Repository</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> | optional | This is optional when returning remote repositories |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> | optional |  |
| owner | <TypeLink type="string">string</TypeLink> |  |  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| repo_id | <TypeLink type="int64">int64</TypeLink> |  |  |
| hook_id | <TypeLink type="int64">int64</TypeLink> |  |  |
| hook_url | <TypeLink type="string">string</TypeLink> |  |  |
| deploy_url | <TypeLink type="string">string</TypeLink> |  |  |
| clone_url | <TypeLink type="string">string</TypeLink> |  |  |
| hook_name | <TypeLink type="string">string</TypeLink> |  |  |
| hook_type | <TypeLink type="string">string</TypeLink> |  |  |
| hook_uuid | <TypeLink type="string">string</TypeLink> |  |  |
| is_private | <TypeLink type="bool">bool</TypeLink> |  |  |
| is_fork | <TypeLink type="bool">bool</TypeLink> |  |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| updated_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| default_branch | <TypeLink type="string">string</TypeLink> |  |  |
| license | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-ResolveInvitationRequest">ResolveInvitationRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | <TypeLink type="string">string</TypeLink> |  | code is the code of the invitation to resolve. |
| accept | <TypeLink type="bool">bool</TypeLink> |  | accept is true if the invitation is accepted, false if it is rejected. |



<Message id="minder-v1-ResolveInvitationResponse">ResolveInvitationResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | <TypeLink type="string">string</TypeLink> |  | role is the role that would be assigned if the user accepts the invitation. |
| email | <TypeLink type="string">string</TypeLink> |  | email is the email address of the invited user. |
| project | <TypeLink type="string">string</TypeLink> |  | project is the project to which the user is invited. |
| is_accepted | <TypeLink type="bool">bool</TypeLink> |  | is_accepted is the status of the invitation. |
| project_display | <TypeLink type="string">string</TypeLink> |  | project_display is the display name of the project to which the user is invited. |



<Message id="minder-v1-RestType">RestType</Message>

RestType defines the rest data evaluation.
This is used to fetch data from a REST endpoint.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | <TypeLink type="string">string</TypeLink> |  | endpoint is the endpoint to fetch data from. This can be a URL or the path on the API.bool This is a required field and must be set. This is also evaluated via a template which allows us dynamically fill in the values. |
| method | <TypeLink type="string">string</TypeLink> |  | method is the method to use to fetch data. |
| headers | <TypeLink type="string">string</TypeLink> | repeated | headers are the headers to be sent to the endpoint. |
| body | <TypeLink type="string">string</TypeLink> | optional | body is the body to be sent to the endpoint. |
| parse | <TypeLink type="string">string</TypeLink> |  | parse is the parsing mechanism to be used to parse the data. |
| fallback | <TypeLink type="minder-v1-RestType-Fallback">RestType.Fallback</TypeLink> | repeated | fallback provides a body that the ingester would return in case the REST call returns a non-200 status code. |



<Message id="minder-v1-RestType-Fallback">RestType.Fallback</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_code | <TypeLink type="int32">int32</TypeLink> |  |  |
| body | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-Role">Role</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the role. |
| display_name | <TypeLink type="string">string</TypeLink> |  | display name of the role |
| description | <TypeLink type="string">string</TypeLink> |  | description is the description of the role. |



<Message id="minder-v1-RoleAssignment">RoleAssignment</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | <TypeLink type="string">string</TypeLink> |  | role is the role that is assigned. |
| subject | <TypeLink type="string">string</TypeLink> |  | subject is the subject to which the role is assigned. |
| display_name | <TypeLink type="string">string</TypeLink> |  | display_name is the display name of the subject. |
| project | <TypeLink type="string">string</TypeLink> | optional | project is the project in which the role is assigned. |
| email | <TypeLink type="string">string</TypeLink> |  | email is the email address of the subject used for invitations. |
| first_name | <TypeLink type="string">string</TypeLink> |  | first_name is the first name of the subject. |
| last_name | <TypeLink type="string">string</TypeLink> |  | last_name is the last name of the subject. |



<Message id="minder-v1-RpcOptions">RpcOptions</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| no_log | <TypeLink type="bool">bool</TypeLink> |  |  |
| target_resource | <TypeLink type="minder-v1-TargetResource">TargetResource</TypeLink> |  |  |
| relation | <TypeLink type="minder-v1-Relation">Relation</TypeLink> |  |  |



<Message id="minder-v1-RuleEvaluationStatus">RuleEvaluationStatus</Message>

get the status of the rules for a given profile


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_id | <TypeLink type="string">string</TypeLink> |  | profile_id is the id of the profile |
| rule_id | <TypeLink type="string">string</TypeLink> |  | rule_id is the id of the rule |
| rule_name | <TypeLink type="string">string</TypeLink> |  | **Deprecated.** rule_name is the type of the rule. Deprecated in favor of rule_type_name |
| entity | <TypeLink type="string">string</TypeLink> |  | entity is the entity that was evaluated |
| status | <TypeLink type="string">string</TypeLink> |  | status is the status of the evaluation |
| last_updated | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | last_updated is the last time the profile was updated |
| entity_info | <TypeLink type="minder-v1-RuleEvaluationStatus-EntityInfoEntry">RuleEvaluationStatus.EntityInfoEntry</TypeLink> | repeated | entity_info is the information about the entity |
| details | <TypeLink type="string">string</TypeLink> |  | details is the description of the evaluation if any |
| guidance | <TypeLink type="string">string</TypeLink> |  | guidance is the guidance for the evaluation if any |
| remediation_status | <TypeLink type="string">string</TypeLink> |  | remediation_status is the status of the remediation |
| remediation_last_updated | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> | optional | remediation_last_updated is the last time the remediation was performed or attempted |
| remediation_details | <TypeLink type="string">string</TypeLink> |  | remediation_details is the description of the remediation attempt if any |
| rule_type_name | <TypeLink type="string">string</TypeLink> |  | rule_type_name is the name of the rule |
| rule_description_name | <TypeLink type="string">string</TypeLink> |  | rule_description_name is the name to describe the rule |
| alert | <TypeLink type="minder-v1-EvalResultAlert">EvalResultAlert</TypeLink> |  | alert holds the alert details if the rule generated an alert in an external system |
| severity | <TypeLink type="minder-v1-Severity">Severity</TypeLink> |  | severity is the severity of the rule |
| rule_evaluation_id | <TypeLink type="string">string</TypeLink> |  | rule_evaluation_id is the id of the rule evaluation |
| remediation_url | <TypeLink type="string">string</TypeLink> |  | remediation_url is a url to get more data about a remediation, for PRs is the link to the PR |
| rule_display_name | <TypeLink type="string">string</TypeLink> |  | rule_display_name captures the display name of the rule |



<Message id="minder-v1-RuleEvaluationStatus-EntityInfoEntry">RuleEvaluationStatus.EntityInfoEntry</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | <TypeLink type="string">string</TypeLink> |  |  |
| value | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-RuleType">RuleType</Message>

RuleType defines rules that may or may not be user defined.
The version is assumed from the folder's version.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> | optional | id is the id of the rule type. This is mostly optional and is set by the server. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the rule type. |
| display_name | <TypeLink type="string">string</TypeLink> |  | display_name is the display name of the rule type. |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the rule is evaluated. |
| def | <TypeLink type="minder-v1-RuleType-Definition">RuleType.Definition</TypeLink> |  | def is the definition of the rule type. |
| description | <TypeLink type="string">string</TypeLink> |  | description is the description of the rule type. |
| guidance | <TypeLink type="string">string</TypeLink> |  | guidance are instructions we give the user in case a rule fails. |
| severity | <TypeLink type="minder-v1-Severity">Severity</TypeLink> |  | severity is the severity of the rule type. |



<Message id="minder-v1-RuleType-Definition">RuleType.Definition</Message>

Definition defines the rule type. It encompases the schema and the data evaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| in_entity | <TypeLink type="string">string</TypeLink> |  | in_entity is the entity in which the rule is evaluated. This can be repository, build_environment or artifact. |
| rule_schema | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> |  | rule_schema is the schema of the rule. This is expressed in JSON Schema. |
| param_schema | <TypeLink type="google-protobuf-Struct">google.protobuf.Struct</TypeLink> | optional | param_schema is the schema of the parameters that are passed to the rule. This is expressed in JSON Schema. |
| ingest | <TypeLink type="minder-v1-RuleType-Definition-Ingest">RuleType.Definition.Ingest</TypeLink> |  |  |
| eval | <TypeLink type="minder-v1-RuleType-Definition-Eval">RuleType.Definition.Eval</TypeLink> |  |  |
| remediate | <TypeLink type="minder-v1-RuleType-Definition-Remediate">RuleType.Definition.Remediate</TypeLink> |  |  |
| alert | <TypeLink type="minder-v1-RuleType-Definition-Alert">RuleType.Definition.Alert</TypeLink> |  |  |



<Message id="minder-v1-RuleType-Definition-Alert">RuleType.Definition.Alert</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  |  |
| security_advisory | <TypeLink type="minder-v1-RuleType-Definition-Alert-AlertTypeSA">RuleType.Definition.Alert.AlertTypeSA</TypeLink> | optional |  |



<Message id="minder-v1-RuleType-Definition-Alert-AlertTypeSA">RuleType.Definition.Alert.AlertTypeSA</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| severity | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-RuleType-Definition-Eval">RuleType.Definition.Eval</Message>

Eval defines the data evaluation definition.
This pertains to the way we traverse data from the upstream
endpoint and how we compare it to the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  | type is the type of the data evaluation. Right now only `jq` is supported as a driver |
| jq | <TypeLink type="minder-v1-RuleType-Definition-Eval-JQComparison">RuleType.Definition.Eval.JQComparison</TypeLink> | repeated | jq is only used if the `jq` type is selected. It defines the comparisons that are made between the ingested data and the profile rule. |
| rego | <TypeLink type="minder-v1-RuleType-Definition-Eval-Rego">RuleType.Definition.Eval.Rego</TypeLink> | optional | rego is only used if the `rego` type is selected. |
| vulncheck | <TypeLink type="minder-v1-RuleType-Definition-Eval-Vulncheck">RuleType.Definition.Eval.Vulncheck</TypeLink> | optional | vulncheck is only used if the `vulncheck` type is selected. |
| trusty | <TypeLink type="minder-v1-RuleType-Definition-Eval-Trusty">RuleType.Definition.Eval.Trusty</TypeLink> | optional | The trusty type is no longer used, but is still here for backwards compatibility with existing stored rules |
| homoglyphs | <TypeLink type="minder-v1-RuleType-Definition-Eval-Homoglyphs">RuleType.Definition.Eval.Homoglyphs</TypeLink> | optional | homoglyphs is only used if the `homoglyphs` type is selected. |



<Message id="minder-v1-RuleType-Definition-Eval-Homoglyphs">RuleType.Definition.Eval.Homoglyphs</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-RuleType-Definition-Eval-JQComparison">RuleType.Definition.Eval.JQComparison</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ingested | <TypeLink type="minder-v1-RuleType-Definition-Eval-JQComparison-Operator">RuleType.Definition.Eval.JQComparison.Operator</TypeLink> |  | Ingested points to the data retrieved in the `ingest` section |
| profile | <TypeLink type="minder-v1-RuleType-Definition-Eval-JQComparison-Operator">RuleType.Definition.Eval.JQComparison.Operator</TypeLink> |  | Profile points to the profile itself. |



<Message id="minder-v1-RuleType-Definition-Eval-JQComparison-Operator">RuleType.Definition.Eval.JQComparison.Operator</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| def | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-RuleType-Definition-Eval-Rego">RuleType.Definition.Eval.Rego</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  | type is the type of evaluation engine to use for rego. We currently have two modes of operation: - deny-by-default: this is the default mode of operation where we deny access by default and allow access only if the profile explicitly allows it. It expects the profile to set an `allow` variable to true or false. - constraints: this is the mode of operation where we allow access by default and deny access only if a violation is found. It expects the profile to set a `violations` variable with a "msg" field. |
| def | <TypeLink type="string">string</TypeLink> |  | def is the definition of the rego profile. |
| violation_format | <TypeLink type="string">string</TypeLink> | optional | how are violations reported. This is only used if the `constraints` type is selected. The default is `text` which returns human-readable text. The other option is `json` which returns a JSON array containing the violations. |



<Message id="minder-v1-RuleType-Definition-Eval-Trusty">RuleType.Definition.Eval.Trusty</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| endpoint | <TypeLink type="string">string</TypeLink> |  | This is no longer used, but is still here for backwards compatibility with existing stored rules |



<Message id="minder-v1-RuleType-Definition-Eval-Vulncheck">RuleType.Definition.Eval.Vulncheck</Message>

no configuration for now



<Message id="minder-v1-RuleType-Definition-Ingest">RuleType.Definition.Ingest</Message>

Ingest defines how the data is ingested.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  | type is the type of the data ingestion. we currently support rest, artifact and builtin. |
| rest | <TypeLink type="minder-v1-RestType">RestType</TypeLink> | optional | rest is the rest data ingestion. this is only used if the type is rest. |
| builtin | <TypeLink type="minder-v1-BuiltinType">BuiltinType</TypeLink> | optional | builtin is the builtin data ingestion. |
| artifact | <TypeLink type="minder-v1-ArtifactType">ArtifactType</TypeLink> | optional | artifact is the artifact data ingestion. |
| git | <TypeLink type="minder-v1-GitType">GitType</TypeLink> | optional | git is the git data ingestion. |
| diff | <TypeLink type="minder-v1-DiffType">DiffType</TypeLink> | optional | diff is the diff data ingestion. |



<Message id="minder-v1-RuleType-Definition-Remediate">RuleType.Definition.Remediate</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | <TypeLink type="string">string</TypeLink> |  |  |
| rest | <TypeLink type="minder-v1-RestType">RestType</TypeLink> | optional |  |
| gh_branch_protection | <TypeLink type="minder-v1-RuleType-Definition-Remediate-GhBranchProtectionType">RuleType.Definition.Remediate.GhBranchProtectionType</TypeLink> | optional |  |
| pull_request | <TypeLink type="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation">RuleType.Definition.Remediate.PullRequestRemediation</TypeLink> | optional |  |



<Message id="minder-v1-RuleType-Definition-Remediate-GhBranchProtectionType">RuleType.Definition.Remediate.GhBranchProtectionType</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| patch | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation">RuleType.Definition.Remediate.PullRequestRemediation</Message>

the name stutters a bit but we already use a PullRequest message for handling PR entities


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | <TypeLink type="string">string</TypeLink> |  | the title of the PR |
| body | <TypeLink type="string">string</TypeLink> |  | the body of the PR |
| contents | <TypeLink type="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content">RuleType.Definition.Remediate.PullRequestRemediation.Content</TypeLink> | repeated |  |
| method | <TypeLink type="string">string</TypeLink> |  | the method to use to create the PR. For now, these are supported: -- minder.content - ensures that the content of the file is exactly as specified refer to the Content message for more details -- minder.actions.replace_tags_with_sha - finds any github actions within a workflow file and replaces the tag with the SHA |
| actions_replace_tags_with_sha | <TypeLink type="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-ActionsReplaceTagsWithSha">RuleType.Definition.Remediate.PullRequestRemediation.ActionsReplaceTagsWithSha</TypeLink> | optional | If the method is minder.actions.replace_tags_with_sha, this is the configuration for that method |



<Message id="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-ActionsReplaceTagsWithSha">RuleType.Definition.Remediate.PullRequestRemediation.ActionsReplaceTagsWithSha</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exclude | <TypeLink type="string">string</TypeLink> | repeated | List of actions to exclude from the replacement |



<Message id="minder-v1-RuleType-Definition-Remediate-PullRequestRemediation-Content">RuleType.Definition.Remediate.PullRequestRemediation.Content</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | <TypeLink type="string">string</TypeLink> |  | the file to patch |
| action | <TypeLink type="string">string</TypeLink> |  | how to patch the file. For now, only replace is supported |
| content | <TypeLink type="string">string</TypeLink> |  | the content of the file |
| mode | <TypeLink type="string">string</TypeLink> | optional | the GIT mode of the file. Not UNIX mode! String because the GH API also uses strings the usual modes are: 100644 for regular files, 100755 for executable files and 040000 for submodules (which we don't use but now you know the meaning of the 1 in 100644) see e.g. https://github.com/go-git/go-git/blob/32e0172851c35ae2fac495069c923330040903d2/plumbing/filemode/filemode.go#L16 |



<Message id="minder-v1-Severity">Severity</Message>

Severity defines the severity of the rule.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | <TypeLink type="minder-v1-Severity-Value">Severity.Value</TypeLink> |  | value is the severity value. |



<Message id="minder-v1-StoreProviderTokenRequest">StoreProviderTokenRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| access_token | <TypeLink type="string">string</TypeLink> |  |  |
| owner | <TypeLink type="string">string</TypeLink> | optional |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-StoreProviderTokenResponse">StoreProviderTokenResponse</Message>





<Message id="minder-v1-TaskRun">TaskRun</Message>





<Message id="minder-v1-UpdateProfileRequest">UpdateProfileRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-UpdateProfileResponse">UpdateProfileResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile | <TypeLink type="minder-v1-Profile">Profile</TypeLink> |  |  |



<Message id="minder-v1-UpdateProjectRequest">UpdateProjectRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the project is updated. |
| display_name | <TypeLink type="string">string</TypeLink> |  | display_name is the display name of the project to update. |
| description | <TypeLink type="string">string</TypeLink> |  | description is the description of the project to update. |



<Message id="minder-v1-UpdateProjectResponse">UpdateProjectResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | <TypeLink type="minder-v1-Project">Project</TypeLink> |  | project is the project that was updated. |



<Message id="minder-v1-UpdateRoleRequest">UpdateRoleRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  | context is the context in which the role assignment is evaluated. |
| subject | <TypeLink type="string">string</TypeLink> |  | subject is the account to change permissions for. The account must already have permissions on the project |
| roles | <TypeLink type="string">string</TypeLink> | repeated | All subject roles are _replaced_ with the following role assignments. Must be non-empty, use RemoveRole to remove permissions entirely from the project. |
| email | <TypeLink type="string">string</TypeLink> |  | email is the email address of the subject used for updating invitations |



<Message id="minder-v1-UpdateRoleResponse">UpdateRoleResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_assignments | <TypeLink type="minder-v1-RoleAssignment">RoleAssignment</TypeLink> | repeated | role_assignments are the role assignments that were updated. |
| invitations | <TypeLink type="minder-v1-Invitation">Invitation</TypeLink> | repeated | invitations contains the details of the invitations that were updated. |



<Message id="minder-v1-UpdateRuleTypeRequest">UpdateRuleTypeRequest</Message>

UpdateRuleTypeRequest is the request to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type to be updated. |



<Message id="minder-v1-UpdateRuleTypeResponse">UpdateRuleTypeResponse</Message>

UpdateRuleTypeResponse is the response to update a rule type.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rule_type | <TypeLink type="minder-v1-RuleType">RuleType</TypeLink> |  | rule_type is the rule type that was updated. |



<Message id="minder-v1-UpstreamRepositoryRef">UpstreamRepositoryRef</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| owner | <TypeLink type="string">string</TypeLink> |  |  |
| name | <TypeLink type="string">string</TypeLink> |  |  |
| repo_id | <TypeLink type="int64">int64</TypeLink> |  | The upstream identity of the repository, as an integer. This is only set on output, and is ignored on input. |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| registered | <TypeLink type="bool">bool</TypeLink> |  | True if the repository is already registered in Minder. This is only set on output, and is ignored on input. |



<Message id="minder-v1-UserRecord">UserRecord</Message>

user record to be returned


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="int32">int32</TypeLink> |  |  |
| identity_subject | <TypeLink type="string">string</TypeLink> |  |  |
| created_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| updated_at | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |



<Message id="minder-v1-VerifyProviderCredentialRequest">VerifyProviderCredentialRequest</Message>

VerifyProviderCredentialRequest contains the enrollment nonce (aka state) that was used when enrolling the provider


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |
| enrollment_nonce | <TypeLink type="string">string</TypeLink> |  | enrollment_nonce is the state parameter returned when enrolling the provider |



<Message id="minder-v1-VerifyProviderCredentialResponse">VerifyProviderCredentialResponse</Message>

VerifyProviderCredentialRequest responds with a boolean indicating if the provider has been created and the provider
name, if it has been created


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| created | <TypeLink type="bool">bool</TypeLink> |  |  |
| provider_name | <TypeLink type="string">string</TypeLink> |  |  |



<Message id="minder-v1-VerifyProviderTokenFromRequest">VerifyProviderTokenFromRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | <TypeLink type="string">string</TypeLink> |  | **Deprecated.**  |
| timestamp | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  |  |
| context | <TypeLink type="minder-v1-Context">Context</TypeLink> |  |  |



<Message id="minder-v1-VerifyProviderTokenFromResponse">VerifyProviderTokenFromResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  |  |


| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| name | string | .google.protobuf.EnumValueOptions | 42445 |  |
| rpc_options | RpcOptions | .google.protobuf.MethodOptions | 51077 |  |






<Enum id="minder-v1-AuthorizationFlow">AuthorizationFlow</Enum>



| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHORIZATION_FLOW_UNSPECIFIED | 0 |  |
| AUTHORIZATION_FLOW_NONE | 1 |  |
| AUTHORIZATION_FLOW_USER_INPUT | 2 |  |
| AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW | 3 |  |
| AUTHORIZATION_FLOW_GITHUB_APP_FLOW | 4 |  |



<Enum id="minder-v1-CredentialsState">CredentialsState</Enum>



| Name | Number | Description |
| ---- | ------ | ----------- |
| CREDENTIALS_STATE_UNSPECIFIED | 0 |  |
| CREDENTIALS_STATE_SET | 1 |  |
| CREDENTIALS_STATE_UNSET | 2 |  |
| CREDENTIALS_STATE_NOT_APPLICABLE | 3 |  |



<Enum id="minder-v1-Entity">Entity</Enum>

Entity defines the entity that is supported by the provider.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITY_UNSPECIFIED | 0 |  |
| ENTITY_REPOSITORIES | 1 |  |
| ENTITY_BUILD_ENVIRONMENTS | 2 |  |
| ENTITY_ARTIFACTS | 3 |  |
| ENTITY_PULL_REQUESTS | 4 |  |
| ENTITY_RELEASE | 5 |  |
| ENTITY_PIPELINE_RUN | 6 |  |
| ENTITY_TASK_RUN | 7 |  |
| ENTITY_BUILD | 8 |  |



<Enum id="minder-v1-ObjectOwner">ObjectOwner</Enum>



| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_OWNER_UNSPECIFIED | 0 |  |
| OBJECT_OWNER_PROJECT | 2 |  |
| OBJECT_OWNER_USER | 3 |  |



<Enum id="minder-v1-ProviderClass">ProviderClass</Enum>



| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_CLASS_UNSPECIFIED | 0 |  |
| PROVIDER_CLASS_GITHUB | 1 |  |
| PROVIDER_CLASS_GITHUB_APP | 2 |  |
| PROVIDER_CLASS_GHCR | 3 |  |
| PROVIDER_CLASS_DOCKERHUB | 4 |  |



<Enum id="minder-v1-ProviderType">ProviderType</Enum>

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



<Enum id="minder-v1-Relation">Relation</Enum>



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
| RELATION_ENTITY_RECONCILE | 36 |  |
| RELATION_ROLE_ASSIGNMENT_UPDATE | 37 |  |



<Enum id="minder-v1-Severity-Value">Severity.Value</Enum>

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



<Enum id="minder-v1-TargetResource">TargetResource</Enum>



| Name | Number | Description |
| ---- | ------ | ----------- |
| TARGET_RESOURCE_UNSPECIFIED | 0 |  |
| TARGET_RESOURCE_NONE | 1 |  |
| TARGET_RESOURCE_USER | 2 |  |
| TARGET_RESOURCE_PROJECT | 3 |  |





<Extension id="minder_v1_minder-proto-extensions">File-level Extensions</Extension>

| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| name | string | .google.protobuf.EnumValueOptions | 42445 |  |
| rpc_options | RpcOptions | .google.protobuf.MethodOptions | 51077 |  |





## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <ProtoType id="double">double</ProtoType> |  | double | double | float | float64 | double | float | Float |
| <ProtoType id="float">float</ProtoType> |  | float | float | float | float32 | float | float | Float |
| <ProtoType id="int32">int32</ProtoType> | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <ProtoType id="int64">int64</ProtoType> | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <ProtoType id="uint32">uint32</ProtoType> | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <ProtoType id="uint64">uint64</ProtoType> | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <ProtoType id="sint32">sint32</ProtoType> | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <ProtoType id="sint64">sint64</ProtoType> | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <ProtoType id="fixed32">fixed32</ProtoType> | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <ProtoType id="fixed64">fixed64</ProtoType> | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <ProtoType id="sfixed32">sfixed32</ProtoType> | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <ProtoType id="sfixed64">sfixed64</ProtoType> | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <ProtoType id="bool">bool</ProtoType> |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <ProtoType id="string">string</ProtoType> | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <ProtoType id="bytes">bytes</ProtoType> | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

