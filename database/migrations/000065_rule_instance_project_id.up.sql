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

-- add column, leave as not null until we populate the column
ALTER TABLE rule_instances ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE;

-- populate by joining on profiles table
UPDATE rule_instances AS ri
SET project_id = pf.project_id
FROM profiles AS pf
WHERE ri.profile_id = pf.id;

-- now we can add the not null constraint
ALTER TABLE rule_instances ALTER COLUMN project_id SET NOT NULL;

COMMIT;
