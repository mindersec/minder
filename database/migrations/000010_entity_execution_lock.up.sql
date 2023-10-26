-- Copyright 2023 Stacklok, Inc
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


--- This implements two tables:
---  * The entity execution lock table, which is used to prevent multiple
---    instances of the same entity from running at the same time.
---  * The flush cache table, which is used to cache entities to be executed
---    once the lock is released.

CREATE TABLE IF NOT EXISTS entity_execution_lock (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    entity entities NOT NULL,
    locked_by UUID NOT NULL,
    last_lock_time TIMESTAMP NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE,
    pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS entity_execution_lock_idx ON entity_execution_lock(
    entity,
    repository_id,
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));

CREATE TABLE IF NOT EXISTS flush_cache (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    entity entities NOT NULL,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE,
    pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE,
    queued_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS flush_cache_idx ON flush_cache(
    entity,
    repository_id,
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID),
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));