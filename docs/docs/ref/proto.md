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




<File id="minder_v1alpha_minder-proto">minder/v1alpha/minder.proto</File>


### Services


<Service id="minder-v1alpha-EvalResultsService">EvalResultsService</Service>



| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListEvaluationResults | [ListEvaluationResultsRequest](#minder-v1alpha-ListEvaluationResultsRequest) | [ListEvaluationResultsResponse](#minder-v1alpha-ListEvaluationResultsResponse) |  |
| ListEvaluationHistory | [ListEvaluationHistoryRequest](#minder-v1alpha-ListEvaluationHistoryRequest) | [ListEvaluationHistoryResponse](#minder-v1alpha-ListEvaluationHistoryResponse) |  |


### Messages


<Message id="minder-v1alpha-EvaluationHistory">EvaluationHistory</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | <TypeLink type="minder-v1alpha-EvaluationHistoryEntity">EvaluationHistoryEntity</TypeLink> |  | entity contains details of the entity which was evaluated. |
| rule | <TypeLink type="minder-v1alpha-EvaluationHistoryRule">EvaluationHistoryRule</TypeLink> |  | rule contains details of the rule which the entity was evaluated against. |
| status | <TypeLink type="minder-v1alpha-EvaluationHistoryStatus">EvaluationHistoryStatus</TypeLink> |  | status contains the evaluation status. |
| alert | <TypeLink type="minder-v1alpha-EvaluationHistoryAlert">EvaluationHistoryAlert</TypeLink> |  | alert contains details of the alerts for this evaluation. |
| remediation | <TypeLink type="minder-v1alpha-EvaluationHistoryRemediation">EvaluationHistoryRemediation</TypeLink> |  | remediation contains details of the remediation for this evaluation. |



<Message id="minder-v1alpha-EvaluationHistoryAlert">EvaluationHistoryAlert</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (on, off, error, skipped, not available) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the alert. the structure and contents are alert specific, and are subject to change. |



<Message id="minder-v1alpha-EvaluationHistoryEntity">EvaluationHistoryEntity</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | <TypeLink type="string">string</TypeLink> |  | id is the ID of the entity. |
| type | <TypeLink type="minder-v1-Entity">minder.v1.Entity</TypeLink> |  | type is the entity type. |
| name | <TypeLink type="string">string</TypeLink> |  | name is the entity name. |



<Message id="minder-v1alpha-EvaluationHistoryRemediation">EvaluationHistoryRemediation</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (success, error, failure, skipped, not available) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the remediation. the structure and contents are remediation specific, and are subject to change. |



<Message id="minder-v1alpha-EvaluationHistoryRule">EvaluationHistoryRule</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | <TypeLink type="string">string</TypeLink> |  | name is the name of the rule instance. |
| type | <TypeLink type="string">string</TypeLink> |  | type is the name of the rule type. |
| profile | <TypeLink type="string">string</TypeLink> |  | profile is the name of the profile which contains the rule. |



<Message id="minder-v1alpha-EvaluationHistoryStatus">EvaluationHistoryStatus</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | <TypeLink type="string">string</TypeLink> |  | status is one of (success, error, failure, skipped) not using enums to mirror the behaviour of the existing API contracts. |
| details | <TypeLink type="string">string</TypeLink> |  | details contains optional details about the evaluation. the structure and contents are rule type specific, and are subject to change. |



<Message id="minder-v1alpha-ListEvaluationHistoryRequest">ListEvaluationHistoryRequest</Message>

ListEvaluationHistoryRequest represents a request message for the
ListEvaluationHistory RPC.

Most of its fields are used for filtering, except for `cursor`
which is used for pagination.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">minder.v1.Context</TypeLink> |  |  |
| entity_type | <TypeLink type="string">string</TypeLink> | repeated | List of entity types to retrieve. |
| entity_name | <TypeLink type="string">string</TypeLink> | repeated | List of entity names to retrieve. |
| profile_name | <TypeLink type="string">string</TypeLink> | repeated | List of profile names to retrieve. |
| status | <TypeLink type="string">string</TypeLink> | repeated | List of evaluation statuses to retrieve. |
| remediation | <TypeLink type="string">string</TypeLink> | repeated | List of remediation statuses to retrieve. |
| alert | <TypeLink type="string">string</TypeLink> | repeated | List of alert statuses to retrieve. |
| from | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | Timestamp representing the start time of the selection window. |
| to | <TypeLink type="google-protobuf-Timestamp">google.protobuf.Timestamp</TypeLink> |  | Timestamp representing the end time of the selection window. |
| cursor | <TypeLink type="minder-v1-Cursor">minder.v1.Cursor</TypeLink> |  | Cursor object to select the "page" of data to retrieve. |



<Message id="minder-v1alpha-ListEvaluationHistoryResponse">ListEvaluationHistoryResponse</Message>

ListEvaluationHistoryResponse represents a response message for the
ListEvaluationHistory RPC.

It ships a collection of records retrieved and pointers to get to
the next and/or previous pages of data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | <TypeLink type="minder-v1alpha-EvaluationHistory">EvaluationHistory</TypeLink> | repeated | List of records retrieved. |
| page | <TypeLink type="minder-v1-CursorPage">minder.v1.CursorPage</TypeLink> |  | Metadata of the current page and pointers to next and/or previous pages. |



<Message id="minder-v1alpha-ListEvaluationResultsRequest">ListEvaluationResultsRequest</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| context | <TypeLink type="minder-v1-Context">minder.v1.Context</TypeLink> |  | context is the context in which the evaluation results are evaluated. |
| new_field | <TypeLink type="string">string</TypeLink> |  |  |
| profile | <TypeLink type="string">string</TypeLink> |  | ID can contain either a profile name or an ID |
| label_filter | <TypeLink type="string">string</TypeLink> |  | Filter profiles to only those matching the specified labels.

The default is to return all user-created profiles; the string "*" can be used to select all profiles, including system profiles. This syntax may be expanded in the future. |
| entity | <TypeLink type="minder-v1-EntityTypedId">minder.v1.EntityTypedId</TypeLink> | repeated | If set, only return evaluation results for the named entities. If empty, return evaluation results for all entities |
| rule_name | <TypeLink type="string">string</TypeLink> | repeated | If set, only return evaluation results for the named rules. If empty, return evaluation results for all rules |



<Message id="minder-v1alpha-ListEvaluationResultsResponse">ListEvaluationResultsResponse</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entities | <TypeLink type="minder-v1alpha-ListEvaluationResultsResponse-EntityEvaluationResults">ListEvaluationResultsResponse.EntityEvaluationResults</TypeLink> | repeated | Each entity selected by the list request will have _single_ entry in entities which contains results of all evaluations for each profile. |



<Message id="minder-v1alpha-ListEvaluationResultsResponse-EntityEvaluationResults">ListEvaluationResultsResponse.EntityEvaluationResults</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entity | <TypeLink type="minder-v1-EntityTypedId">minder.v1.EntityTypedId</TypeLink> |  |  |
| profiles | <TypeLink type="minder-v1alpha-ListEvaluationResultsResponse-EntityProfileEvaluationResults">ListEvaluationResultsResponse.EntityProfileEvaluationResults</TypeLink> | repeated |  |



<Message id="minder-v1alpha-ListEvaluationResultsResponse-EntityProfileEvaluationResults">ListEvaluationResultsResponse.EntityProfileEvaluationResults</Message>




| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| profile_status | <TypeLink type="minder-v1-ProfileStatus">minder.v1.ProfileStatus</TypeLink> |  | profile_status is the status of the profile - id, name, status, last_updated |
| results | <TypeLink type="minder-v1-RuleEvaluationStatus">minder.v1.RuleEvaluationStatus</TypeLink> | repeated | Note that some fields like profile_id and entity might be empty Eventually we might replace this type with another one that fits the API better |


| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| rpc_options | minder.v1.RpcOptions | .google.protobuf.MethodOptions | 51078 |  |








<Extension id="minder_v1alpha_minder-proto-extensions">File-level Extensions</Extension>

| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| rpc_options | minder.v1.RpcOptions | .google.protobuf.MethodOptions | 51078 |  |





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

