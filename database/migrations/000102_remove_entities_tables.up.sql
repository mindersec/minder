-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

BEGIN;

-- remove the repositories, artifacts and pull_request tables

DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS pull_requests;

CREATE VIEW repositories AS
SELECT
    ei.id,
    ei.project_id,
    pr.name AS provider,
    ei.provider_id,
    (prop_owner.value->>'text')::TEXT AS repo_owner,
    (prop_name.value->>'text')::TEXT AS repo_name,
    (prop_repo_id.value->>'number')::BIGINT AS repo_id,
    (prop_is_private.value->>'boolean')::BOOLEAN AS is_private,
    (prop_is_fork.value->>'boolean')::BOOLEAN AS is_fork,
    (prop_webhook_id.value->>'number')::BIGINT AS webhook_id,
    (prop_webhook_url.value->>'text')::TEXT AS webhook_url,
    (prop_deploy_url.value->>'text')::TEXT AS deploy_url,
    (prop_clone_url.value->>'text')::TEXT AS clone_url,
    (prop_default_branch.value->>'text')::TEXT AS default_branch,
    (prop_license.value->>'text')::TEXT AS license,
    ei.created_at
FROM
    entity_instances ei
    JOIN providers pr ON ei.provider_id = pr.id
    LEFT JOIN properties prop_owner ON ei.id = prop_owner.entity_id AND prop_owner.key = 'repo_owner'
    LEFT JOIN properties prop_name ON ei.id = prop_name.entity_id AND prop_name.key = 'repo_name'
    LEFT JOIN properties prop_repo_id ON ei.id = prop_repo_id.entity_id AND prop_repo_id.key = 'repo_id'
    LEFT JOIN properties prop_is_private ON ei.id = prop_is_private.entity_id AND prop_is_private.key = 'is_private'
    LEFT JOIN properties prop_is_fork ON ei.id = prop_is_fork.entity_id AND prop_is_fork.key = 'is_fork'
    LEFT JOIN properties prop_webhook_id ON ei.id = prop_webhook_id.entity_id AND prop_webhook_id.key = 'webhook_id'
    LEFT JOIN properties prop_webhook_url ON ei.id = prop_webhook_url.entity_id AND prop_webhook_url.key = 'webhook_url'
    LEFT JOIN properties prop_deploy_url ON ei.id = prop_deploy_url.entity_id AND prop_deploy_url.key = 'deploy_url'
    LEFT JOIN properties prop_clone_url ON ei.id = prop_clone_url.entity_id AND prop_clone_url.key = 'clone_url'
    LEFT JOIN properties prop_default_branch ON ei.id = prop_default_branch.entity_id AND prop_default_branch.key = 'default_branch'
    LEFT JOIN properties prop_license ON ei.id = prop_license.entity_id AND prop_license.key = 'license'
WHERE
    ei.entity_type = 'repository';

CREATE VIEW artifacts AS
SELECT
    ei.id,
    ei.project_id,
    pr.name AS provider_name,
    ei.provider_id,
    ei.originated_from AS repository_id,
    (prop_artifact_name.value->>'text')::TEXT AS artifact_name,
    (prop_artifact_type.value->>'text')::TEXT AS artifact_type,
    (prop_artifact_visibility.value->>'text')::TEXT AS artifact_visibility,
    ei.created_at
FROM
    entity_instances ei
    JOIN providers pr ON ei.provider_id = pr.id
    LEFT JOIN properties prop_artifact_name ON ei.id = prop_artifact_name.entity_id AND prop_artifact_name.key = 'artifact_name'
    LEFT JOIN properties prop_artifact_type ON ei.id = prop_artifact_type.entity_id AND prop_artifact_type.key = 'artifact_type'
    LEFT JOIN properties prop_artifact_visibility ON ei.id = prop_artifact_visibility.entity_id AND prop_artifact_visibility.key = 'artifact_visibility'
WHERE
    ei.entity_type = 'artifact';

CREATE VIEW pull_requests AS
SELECT
    ei.id,
    ei.originated_from AS repository_id,
    (prop_pr_number.value->>'number')::BIGINT AS pr_number,
    ei.created_at
FROM
    entity_instances ei
    LEFT JOIN properties prop_pr_number ON ei.id = prop_pr_number.entity_id AND prop_pr_number.key = 'pr_number'
WHERE
    ei.entity_type = 'pull_request';


COMMIT;