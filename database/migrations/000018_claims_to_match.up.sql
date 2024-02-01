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

CREATE TABLE IF NOT EXISTS mapped_role_grants(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    claim_mappings JSONB NOT NULL,
    resolved_subject TEXT REFERENCES users(identity_subject) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- index claims_mappings so we can query it
CREATE INDEX IF NOT EXISTS idx_mapped_role_grant_claim_mappings ON mapped_role_grants USING GIN(claim_mappings);

-- index project + role + resolved_subject so we can query it
CREATE INDEX IF NOT EXISTS idx_mapped_role_grant_project_role_resolved_subject ON mapped_role_grants (project_id, role, resolved_subject) WHERE resolved_subject IS NOT NULL;