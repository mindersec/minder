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

-- artifact versions table
CREATE TABLE artifact_versions (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    tags TEXT,
    sha TEXT NOT NULL,
    signature_verification JSONB,       -- see /proto/minder/v1/minder.proto#L82
    github_workflow JSONB,              -- see /proto/minder/v1/minder.proto#L75
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX artifact_versions_idx ON artifact_versions (artifact_id, sha);
