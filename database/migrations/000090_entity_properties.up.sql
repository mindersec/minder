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

CREATE TABLE IF NOT EXISTS entity_instances (
	id          UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
	entity_type entities NOT NULL, 
	name        TEXT NOT NULL,
	project_id  UUID NOT NULL REFERENCES projects(id),
	provider_id UUID NOT NULL REFERENCES providers(id),
	created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	originated_from UUID REFERENCES entity_instances(id) ON DELETE CASCADE, -- this is for entities that originate from other entities
	UNIQUE(project_id, provider_id, entity_type, name)
);

CREATE TABLE IF NOT EXISTS properties(
	id          UUID PRIMARY KEY, -- surrogate ID
	entity_id   UUID NOT NULL REFERENCES entity_instances(id) ON DELETE CASCADE,
	key         TEXT NOT NULL, -- we need to validate and ensure there are no dots
	value       JSONB NOT NULL,
	updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
	UNIQUE (entity_id, key)
);

COMMIT;